// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helmadapter

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/gofrs/flock"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mitchellh/mapstructure"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"

	"github.com/banzaicloud/pipeline/internal/helm"
)

// helm3EnvService component struct for helm3 repository management
type helm3EnvService struct {
	secretStore helm.SecretStore
	logger      Logger
}

func NewHelm3EnvService(secretStore helm.SecretStore, logger Logger) helm.EnvService {
	return helm3EnvService{
		secretStore: secretStore,
		logger:      logger,
	}
}

func (h helm3EnvService) AddRepository(ctx context.Context, helmEnv helm.HelmEnv, repository helm.Repository) error {
	repoFile := helmEnv.GetHome() // TODO add another field to the env instead???

	// Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(helmEnv.GetHome()), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		// TODO inject an error handler! (alternatives?)
		defer emperror.NoopHandler{}.Handle(fileLock.Unlock())
	}
	if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}

	if f.Has(repository.Name) {
		return errors.Errorf("repository name (%s) already exists, please specify a different name", repository.Name)
	}

	c := repo.Entry{
		Name: repository.Name,
		URL:  repository.URL,
	}
	if repository.PasswordSecretID != "" {
		passwordSecret, err := h.secretStore.ResolvePasswordSecrets(ctx, repository.PasswordSecretID)
		if err != nil {
			return errors.WrapIf(err, "failed to resolve repo credentials")
		}

		c.Username = passwordSecret.UserName
		c.Password = passwordSecret.Password
	}

	envSettings := h.processEnvSettings(helmEnv)
	r, err := repo.NewChartRepository(&c, getter.All(envSettings))
	if err != nil {
		return err
	}

	// override the wired repository cache
	r.CachePath = envSettings.RepositoryCache
	if _, err := r.DownloadIndexFile(); err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", repository.URL)
	}

	f.Update(&c)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		return err
	}
	h.logger.Info("repository has been added", map[string]interface{}{"repository": repository.Name})
	return nil
}

func (h helm3EnvService) ListRepositories(_ context.Context, helmEnv helm.HelmEnv) ([]helm.Repository, error) {
	f, err := repo.LoadFile(helmEnv.GetHome())
	if isNotExist(err) || len(f.Repositories) == 0 {
		return nil, nil
	}

	repos := make([]helm.Repository, 0, len(f.Repositories))
	for _, entry := range f.Repositories {
		repos = append(repos, helm.Repository{
			Name: entry.Name,
			URL:  entry.URL,
			// TODO warning! do not propagate sensitive data!
		})
	}

	return repos, nil
}

func (h helm3EnvService) DeleteRepository(_ context.Context, helmEnv helm.HelmEnv, repoName string) error {
	repoFile := helmEnv.GetHome()
	r, err := repo.LoadFile(repoFile)
	if err != nil {
		if os.IsNotExist(errors.Cause(err)) || len(r.Repositories) == 0 {
			h.logger.Warn("no  repositories configured, nothing to do")
			return nil
		}
	}

	if !r.Remove(repoName) {
		h.logger.Debug("repository not  found", map[string]interface{}{"repository": repoName})
		return nil
	}
	if err := r.WriteFile(repoFile, 0644); err != nil {
		return err
	}

	if err := removeRepoCache(helmEnv.GetRepoCache(), repoName); err != nil {
		return err
	}

	h.logger.Info("repository has been removed", map[string]interface{}{"repository": repoName})

	return nil
}

func (h helm3EnvService) PatchRepository(ctx context.Context, helmEnv helm.HelmEnv, repository helm.Repository) error {
	return h.UpdateRepository(ctx, helmEnv, repository)
}

func (h helm3EnvService) UpdateRepository(ctx context.Context, helmEnv helm.HelmEnv, repository helm.Repository) error {
	if repository.URL == "" {
		return errors.NewWithDetails("failed to update repository; missing repository URL")
	}

	if err := h.DeleteRepository(ctx, helmEnv, repository.Name); err != nil {
		return errors.WrapIf(err, "failed to remove repo before update")
	}

	if err := h.AddRepository(ctx, helmEnv, repository); err != nil {
		return errors.WrapIf(err, "failed re-add repository during update")
	}

	return nil
}

// ListCharts finds the charts based on the provided filter
func (h helm3EnvService) ListCharts(ctx context.Context, helmEnv helm.HelmEnv, filter helm.ChartFilter) (helm.ChartList, error) {
	// map: {repoName -> [[{chart1:v1},..,{chart1:vn}],.., [{chartm:v1},..,{chartm:vp}]}
	repoChartVersionsSliceMap, err := h.listCharts(ctx, helmEnv, filter)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve chart list")
	}

	// adapt it to the existing api
	type repoChartType struct {
		Name   string        `json:"name" mapstructure:"name"`
		Charts []interface{} `json:"charts" mapstructure:"charts"`
	}

	adaptedList := make(helm.ChartList, 0, 0)

	for repoName, repoChartVersionsSlice := range repoChartVersionsSliceMap {
		repoCharts := repoChartType{
			Name:   repoName,
			Charts: make([]interface{}, 0, 0),
		}

		//[[{chart1:v1},..,{chart1:vn}],..,[{chartm:v1},..,{chartm:vp}]}
		for _, chartVersions := range repoChartVersionsSlice {
			versions := make([]interface{}, 0, len(chartVersions))

			// [{chart1:v1},..,{chart1:vn}]
			for _, chartVersion := range chartVersions {
				versions = append(versions, chartVersion)
			}

			repoCharts.Charts = append(repoCharts.Charts, versions)
		}

		if len(repoCharts.Charts) > 0 {
			adaptedList = append(adaptedList, repoCharts)
		}
	}

	return adaptedList, nil
}

func (h helm3EnvService) GetChart(ctx context.Context, helmEnv helm.HelmEnv, filter helm.ChartFilter) (helm.ChartDetails, error) {
	repoChartsSlice, err := h.listCharts(ctx, helmEnv, filter)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to look up charts in repo", "repo", filter.RepoFilter())
	}

	chartsInSlice, ok := repoChartsSlice[filter.RepoFilter()]
	if !ok {
		return nil, errors.WrapIfWithDetails(err, "failed get charts for repo", "repo", filter.RepoFilter())
	}

	if len(chartsInSlice) == 0 {
		h.logger.Debug("chart not found", map[string]interface{}{"filter": filter})
		return helm.ChartDetails{}, nil
	}

	if len(chartsInSlice) > 1 {
		return nil, errors.New("found more than one repositories")
	}

	// transform the response
	detailedCharts, err := h.getDetailedCharts(ctx, helmEnv, chartsInSlice[0])
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get detailed charts")
	}

	return h.adaptChartDetailsResponse(detailedCharts, filter.RepoFilter(), chartsInSlice[0])
}

// processEnvSettings emulates an cli.EnvSettings instance based on the passed in data
func (h helm3EnvService) processEnvSettings(helmEnv helm.HelmEnv) *cli.EnvSettings {
	envSettings := cli.New()
	envSettings.RepositoryConfig = helmEnv.GetHome()
	envSettings.RepositoryCache = helmEnv.GetRepoCache()

	return envSettings
}

func removeRepoCache(root, name string) error {
	idx := filepath.Join(root, helmpath.CacheChartsFile(name))
	if _, err := os.Stat(idx); err == nil {
		os.Remove(idx)
	}

	idx = filepath.Join(root, helmpath.CacheIndexFile(name))
	if _, err := os.Stat(idx); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "can't remove index file %s", idx)
	}
	return os.Remove(idx)
}

func (h helm3EnvService) updateCharts(repos []*repo.ChartRepository, out io.Writer) {
	fmt.Fprintln(out, "Hang tight while we grab the latest from your chart repositories...")
	var wg sync.WaitGroup
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err := re.DownloadIndexFile(); err != nil {
				fmt.Fprintf(out, "...Unable to get an update from the %q chart repository (%s):\n\t%s\n", re.Config.Name, re.Config.URL, err)
			} else {
				fmt.Fprintf(out, "...Successfully got an update from the %q chart repository\n", re.Config.Name)
			}
		}(re)
	}
	wg.Wait()
	fmt.Fprintln(out, "Update Complete. ⎈ Happy Helming!⎈ ")
}

// matchesFilter checks whether the passed in value matches the given filter
// empty filter is considered "no filter"
// the value is treated as a regexp
func matchesFilter(filter string, value string) bool {
	if filter == "" {
		// there is no filter
		return true
	}

	matches, err := regexp.MatchString(filter, strings.ToLower(value))
	if err != nil {
		return false
	}

	return matches
}

// getRawFileContent returns the content of the passed in file in the chart details reference
func (h helm3EnvService) getRawChartFileContent(chartFileName string, chartPtr *chart.Chart) string {
	for _, chartFile := range chartPtr.Raw {
		if chartFile.Name == chartFileName {
			content := chartFile.Data
			if strings.HasSuffix(chartFileName, ".md") {
				content = bluemonday.UGCPolicy().SanitizeBytes(content)
			}
			return base64.StdEncoding.EncodeToString(content)
		}
	}

	h.logger.Debug("no chart file found", map[string]interface{}{"chartFile": chartFileName})
	return ""
}

// listCharts retrieves  charts based on the input data
// operates with h3 lib types
func (h helm3EnvService) listCharts(_ context.Context, helmEnv helm.HelmEnv, filter helm.ChartFilter) (map[string][]repo.ChartVersions, error) {
	chartVersionsSlice := make(map[string][]repo.ChartVersions)

	repoFile, err := repo.LoadFile(helmEnv.GetHome())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to load repo file")
	}

	if filter.RepoFilter() != "" && !repoFile.Has(filter.RepoFilter()) {
		return nil, errors.WrapIfWithDetails(err, "repository not found", "filter", filter)
	}

	for _, repoEntry := range repoFile.Repositories {
		if !matchesFilter(filter.StrictRepoFilter(), repoEntry.Name) {
			h.logger.Debug("repository name doesn't match the filter",
				map[string]interface{}{"filter": filter.RepoFilter(), "repoEntry": repoEntry.Name})
			// skip further processing
			continue
		}

		repoIndexFilePath := path.Join(helmEnv.GetRepoCache(), helmpath.CacheIndexFile(repoEntry.Name))
		repoIndexFile, err := repo.LoadIndexFile(repoIndexFilePath)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to load index file for repo")
		}

		for chartName, chartVersions := range repoIndexFile.Entries {
			filteredChartVersions := make(repo.ChartVersions, 0, 0)

			if !matchesFilter(filter.StrictNameFilter(), chartName) {
				h.logger.Debug("chart name doesn't match the filter, skipping the entry",
					map[string]interface{}{"filter": filter.StrictNameFilter(), "chart": chartName})
				// skip further processing
				continue
			}

			// special case: latest versions to be returned only, no need to iterate over versions
			if filter.VersionFilter() == "latest" {
				filteredChartVersions = append(filteredChartVersions, chartVersions[0])
				chartVersionsSlice[repoEntry.Name] = append(chartVersionsSlice[repoEntry.Name], filteredChartVersions)
				continue
			}

			// special case: all
			if filter.VersionFilter() == "all" {
				filteredChartVersions = append(filteredChartVersions, chartVersions...)
				chartVersionsSlice[repoEntry.Name] = append(chartVersionsSlice[repoEntry.Name], filteredChartVersions)
				continue
			}

			for _, chartVersion := range chartVersions {
				if !matchesFilter(filter.KeywordFilter(), strings.Join(chartVersion.Keywords, " ")) {
					h.logger.Debug("chart keywords don't match the filter, skipping the version",
						map[string]interface{}{"filter": filter.KeywordFilter(), "keywords": chartVersion.Keywords})
					// skip further processing
					continue
				}

				if !matchesFilter(filter.StrictVersionFilter(), chartVersion.Version) {
					h.logger.Debug("chart version doesn't match the filter, skipping the version",
						map[string]interface{}{"filter": filter.VersionFilter(), "version": chartVersion.Version})
					// skip further processing
					continue
				}

				filteredChartVersions = append(filteredChartVersions, chartVersion)
			}
			if len(filteredChartVersions) > 0 {
				chartVersionsSlice[repoEntry.Name] = append(chartVersionsSlice[repoEntry.Name], filteredChartVersions)
			}
		}
	}

	return chartVersionsSlice, nil
}

// getDetailedChart gets the chart details from the chart archive
func (h helm3EnvService) getDetailedCharts(_ context.Context, helmEnv helm.HelmEnv, repoVersions repo.ChartVersions) (map[string]*chart.Chart, error) {
	getters := getter.All(h.processEnvSettings(helmEnv))

	detailedCharts := make(map[string]*chart.Chart)
	for _, repoChartVersionPtr := range repoVersions {
		// todo check the other urls, other checks?
		repoUrl := repoChartVersionPtr.URLs[0]
		u, err := url.Parse(repoUrl)
		if err != nil {
			return nil, errors.Errorf("invalid chart URL format: %s", repoUrl)
		}

		client, err := getters.ByScheme(u.Scheme)
		if err != nil {
			return nil, errors.Errorf("could not find protocol handler for: %s", u.Scheme)
		}

		buffer, err := client.Get(repoUrl)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to get archive")
		}

		bufferedFilePtr, err := loader.LoadArchiveFiles(bytes.NewReader(buffer.Bytes()))
		if err != nil {
			return nil, errors.WrapIf(err, "failed to load archive files")
		}

		detailedChart, err := loader.LoadFiles(bufferedFilePtr)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to load archive")
		}

		detailedCharts[fmt.Sprintf("%s-%s", repoChartVersionPtr.Name, repoChartVersionPtr.Version)] = detailedChart
	}

	return detailedCharts, nil
}

func (h helm3EnvService) adaptChartDetailsResponse(charts map[string]*chart.Chart, repoName string, repoVersions repo.ChartVersions) (helm.ChartDetails, error) {
	// internal types to facilitate transformations  /adapt the response format to the API
	type (
		chartVersion struct {
			Chart  *repo.ChartVersion `json:"chart" mapstructure:"chart"`
			Values string             `json:"values" mapstructure:"values"`
			Readme string             `json:"readme" mapstructure:"readme"`
		}

		chartDetails struct {
			Name     string          `json:"name" mapstructure:"name"`
			Repo     string          `json:"repo" mapstructure:"repo"`
			Versions []*chartVersion `json:"versions" mapstructure:"versions"`
		}
	)

	response := chartDetails{
		Repo:     repoName,
		Versions: make([]*chartVersion, 0, 0),
	}

	for _, repoVersion := range repoVersions {
		chartPtr := charts[fmt.Sprintf("%s-%s", repoVersion.Name, repoVersion.Version)]
		repoChartVersion := chartVersion{
			Chart:  repoVersion,
			Values: h.getRawChartFileContent("values.yaml", chartPtr),
			Readme: h.getRawChartFileContent("README.md", chartPtr),
		}

		response.Versions = append(response.Versions, &repoChartVersion)
	}

	responseMap := make(map[string]interface{})
	if err := mapstructure.Decode(response, &responseMap); err != nil {
		return nil, errors.WrapIf(err, "failed to transform chart details response")
	}

	return responseMap, nil
}

func (h helm3EnvService) EnsureEnv(ctx context.Context, helmEnv helm.HelmEnv, defaultRepos []helm.Repository) (helm.HelmEnv, bool, error) {
	repoFile := helmEnv.GetHome()

	// Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(helmEnv.GetHome()), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return helm.HelmEnv{}, false, errors.WrapIf(err, "failed to ensure helm env")
	}

	// check the repofile
	if fileExists(repoFile) {
		h.logger.Debug("helm env ensured, helm env was already set up")
		return helmEnv, false, nil
	}

	// creating the repo file
	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		// file successfully locked
		defer emperror.NoopHandler{}.Handle(fileLock.Unlock())
	}
	if err != nil {
		return helm.HelmEnv{}, false, errors.WrapIf(err, "failed to lock the helm home dir for creating repo file")
	}

	f := repo.NewFile()
	if err := f.WriteFile(helmEnv.GetHome(), 0644); err != nil {
		return helm.HelmEnv{}, false, errors.WrapIf(err, "failed to create the repo file")
	}

	for _, repo := range defaultRepos {
		if err := h.AddRepository(ctx, helmEnv, repo); err != nil {
			// Notice the error, and proceed forward
			h.logger.Warn("failed to add default repository", map[string]interface{}{"helmEnv": helmEnv, "repo": repo})
		}
	}

	h.logger.Info("successfully ensured helm env")
	return helmEnv, true, nil
}

func (h helm3EnvService) CheckReleaseCharts(ctx context.Context, helmEnv helm.HelmEnv, releases []helm.Release) (map[string]bool, error) {
	// build chart version index
	repoCharts, err := h.listCharts(ctx, helmEnv, helm.ChartFilter{})
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve charts")
	}

	// chartName -> chartVersions
	supportedCharts := map[string]repo.ChartVersions{}
	for _, charts := range repoCharts {
		for _, chart := range charts {
			for _, chartVersion := range chart {
				supportedCharts[chartVersion.Name] = append(supportedCharts[chartVersion.Name], chartVersion)
			}
		}
	}

	// releaseName -> supported
	supportedMap := make(map[string]bool)
	for _, release := range releases {
		supportedMap[release.ReleaseName] = false
		releaseChartVersions, ok := supportedCharts[release.ChartName]
		if !ok {
			continue
		}

		for _, releaseChartVersion := range releaseChartVersions {
			if release.Version == releaseChartVersion.Version {
				supportedMap[release.ReleaseName] = true
				continue
			}
		}
	}

	return supportedMap, nil
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func isNotExist(err error) bool {
	return os.IsNotExist(errors.Cause(err))
}
