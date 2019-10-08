// Copyright © 2019 Banzai Cloud
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

package helm

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"

	"github.com/banzaicloud/pipeline/config"

	helmEnv "k8s.io/helm/pkg/helm/environment"

	phelm "github.com/banzaicloud/pipeline/pkg/helm"
)

// ChartDetails describes a chart details
type ChartDetails struct {
	Name     string          `json:"name"`
	Repo     string          `json:"repo"`
	Versions []*ChartVersion `json:"versions"`
}

// ChartVersion describes a chart verion
type ChartVersion struct {
	Chart  *repo.ChartVersion `json:"chart"`
	Values string             `json:"values"`
	Readme string             `json:"readme"`
}

// ChartList describe a chart list
type ChartList struct {
	Name   string               `json:"name"`
	Charts []repo.ChartVersions `json:"charts"`
}

// InstallLocalHelm install helm into the given path
func InstallLocalHelm(env helmEnv.EnvSettings) error {
	if err := InstallHelmClient(env); err != nil {
		return err
	}
	log.Info("Helm client install succeeded")

	if err := ensureDefaultRepos(env); err != nil {
		return errors.Wrap(err, "Setting up default repos failed!")
	}
	return nil
}

// InstallHelmClient Installs helm client on a given path
func InstallHelmClient(env helmEnv.EnvSettings) error {
	if err := EnsureDirectories(env); err != nil {
		return errors.Wrap(err, "Initializing helm directories failed!")
	}

	log.Info("Initializing helm client succeeded, happy helming!")
	return nil
}

// CreateEnvSettings Create env settings on a given path
func createEnvSettings(helmRepoHome string) helmEnv.EnvSettings {
	var settings helmEnv.EnvSettings
	settings.Home = helmpath.Home(helmRepoHome)
	return settings
}

// GenerateHelmRepoEnv Generate helm path based on orgName
func GenerateHelmRepoEnv(orgName string) (env helmEnv.EnvSettings) {
	var helmPath = config.GetHelmPath(orgName)
	env = createEnvSettings(fmt.Sprintf("%s/%s", helmPath, phelm.HelmPostFix))

	// check local helm
	if _, err := os.Stat(helmPath); os.IsNotExist(err) {
		log.Infof("Helm directories [%s] not exists", helmPath)
		InstallLocalHelm(env) // nolint: errcheck
	}

	return
}

// EnsureDirectories for helm repo local install
func EnsureDirectories(env helmEnv.EnvSettings) error {
	home := env.Home
	configDirectories := []string{
		home.String(),
		home.Repository(),
		home.Cache(),
		home.LocalRepository(),
		home.Plugins(),
		home.Starters(),
		home.Archive(),
	}

	log.Info("Setting up helm directories.")

	for _, p := range configDirectories {
		if fi, err := os.Stat(p); err != nil {
			log.Infof("Creating '%s'", p)
			if err := os.MkdirAll(p, 0755); err != nil {
				return errors.Wrapf(err, "Could not create '%s'", p)
			}
		} else if !fi.IsDir() {
			return errors.Errorf("'%s' must be a directory", p)
		}
	}
	return nil
}

func ensureDefaultRepos(env helmEnv.EnvSettings) error {

	stableRepositoryURL := viper.GetString("helm.stableRepositoryURL")
	banzaiRepositoryURL := viper.GetString("helm.banzaiRepositoryURL")

	log.Infof("Setting up default helm repos.")

	_, err := AddRepo(
		env,
		&repo.Entry{
			Name:  phelm.StableRepository,
			URL:   stableRepositoryURL,
			Cache: env.Home.CacheIndex(phelm.StableRepository),
		})
	if err != nil {
		return errors.Wrapf(err, "cannot init repo: %s", phelm.StableRepository)
	}
	_, err = AddRepo(
		env,
		&repo.Entry{
			Name:  phelm.BanzaiRepository,
			URL:   banzaiRepositoryURL,
			Cache: env.Home.CacheIndex(phelm.BanzaiRepository),
		})
	if err != nil {
		return errors.Wrapf(err, "cannot init repo: %s", phelm.BanzaiRepository)
	}
	return nil
}

// ListRepos returns repo
func ListRepos(env helmEnv.EnvSettings) ([]*repo.Entry, error) {

	repoPath := env.Home.RepositoryFile()
	log.Debugf("Helm repo path: %s", repoPath)

	f, err := repo.LoadRepositoriesFile(repoPath)
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return make([]*repo.Entry, 0), nil
	}

	return f.Repositories, nil
}

// AddRepo adds repo(s)
func AddRepo(env helmEnv.EnvSettings, helmRepo *repo.Entry) (bool, error) {
	repoFile := env.Home.RepositoryFile()
	var f *repo.RepoFile
	if _, err := os.Stat(repoFile); err != nil {
		log.Infof("Creating %s", repoFile)
		f = repo.NewRepoFile()
	} else {
		f, err = repo.LoadRepositoriesFile(repoFile)
		if err != nil {
			return false, errors.Wrap(err, "Cannot create a new ChartRepo")
		}
		log.Debugf("Profile file %q loaded.", repoFile)
	}

	for _, n := range f.Repositories {
		log.Debugf("repo: %s", n.Name)
		if n.Name == helmRepo.Name {
			return false, nil
		}
	}

	c := repo.Entry{
		Name:  helmRepo.Name,
		URL:   helmRepo.URL,
		Cache: env.Home.CacheIndex(helmRepo.Name),
	}
	r, err := repo.NewChartRepository(&c, getter.All(env))
	if err != nil {
		return false, errors.Wrap(err, "Cannot create a new ChartRepo")
	}
	log.Debugf("New repo added: %s", helmRepo.Name)

	errIdx := r.DownloadIndexFile("")
	if errIdx != nil {
		return false, errors.Wrap(errIdx, "Repo index download failed")
	}
	f.Add(&c)
	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return false, errors.Wrap(errW, "Cannot write helm repo profile file")
	}
	return true, nil
}

// DeleteRepo deletes repo(s)
func DeleteRepo(env helmEnv.EnvSettings, repoName string) error {
	repoFile := env.Home.RepositoryFile()
	log.Debugf("Repo File: %s", repoFile)

	r, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	if !r.Remove(repoName) {
		return ErrRepoNotFound
	}
	if err := r.WriteFile(repoFile, 0644); err != nil {
		return err
	}

	if _, err := os.Stat(env.Home.CacheIndex(repoName)); err == nil {
		err = os.Remove(env.Home.CacheIndex(repoName))
		if err != nil {
			return err
		}
	}
	return nil

}

// ModifyRepo modifies repo(s)
func ModifyRepo(env helmEnv.EnvSettings, repoName string, newRepo *repo.Entry) error {

	log.Debug("ModifyRepo")
	repoFile := env.Home.RepositoryFile()
	log.Debugf("Repo File: %s", repoFile)
	log.Debugf("New repo content: %#v", newRepo)

	f, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	if !f.Has(repoName) {
		return ErrRepoNotFound
	}

	var formerRepo *repo.Entry
	repos := f.Repositories
	for _, r := range repos {
		if r.Name == repoName {
			formerRepo = r
		}
	}

	if formerRepo != nil {
		if len(newRepo.Name) == 0 {
			newRepo.Name = formerRepo.Name
			log.Infof("new repo name field is empty, replaced with: %s", formerRepo.Name)
		}

		if len(newRepo.URL) == 0 {
			newRepo.URL = formerRepo.URL
			log.Infof("new repo url field is empty, replaced with: %s", formerRepo.URL)
		}

		if len(newRepo.Cache) == 0 {
			newRepo.Cache = formerRepo.Cache
			log.Infof("new repo cache field is empty, replaced with: %s", formerRepo.Cache)
		}
	}

	f.Update(newRepo)

	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return errors.Wrap(errW, "Cannot write helm repo profile file")
	}
	return nil
}

// UpdateRepo updates a repo(s)
func UpdateRepo(env helmEnv.EnvSettings, repoName string) error {

	repoFile := env.Home.RepositoryFile()
	log.Debugf("Repo File: %s", repoFile)

	f, err := repo.LoadRepositoriesFile(repoFile)

	if err != nil {
		return errors.Wrap(err, "Load ChartRepo")
	}

	for _, cfg := range f.Repositories {
		if cfg.Name == repoName {
			c, err := repo.NewChartRepository(cfg, getter.All(env))
			if err != nil {
				return errors.Wrap(err, "Cannot get ChartRepo")
			}
			errIdx := c.DownloadIndexFile("")
			if errIdx != nil {
				return errors.Wrap(errIdx, "Repo index download failed")
			}
			return nil

		}
	}

	return ErrRepoNotFound
}

// ListCharts returns chart list
func ListCharts(env helmEnv.EnvSettings, queryName, queryRepo, queryVersion, queryKeyword string) ([]ChartList, error) {
	log.Debugf("Helm repo path %s", env.Home.RepositoryFile())
	f, err := repo.LoadRepositoriesFile(env.Home.RepositoryFile())
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return nil, nil
	}
	cl := make([]ChartList, 0)

	for _, r := range f.Repositories {

		log.Debugf("Repository: %s", r.Name)
		i, errIndx := repo.LoadIndexFile(r.Cache)
		if errIndx != nil {
			return nil, errIndx
		}
		repoMatched, _ := regexp.MatchString(queryRepo, strings.ToLower(r.Name))
		if repoMatched || queryRepo == "" {
			log.Debugf("Repository: %s Matched", r.Name)
			c := ChartList{
				Name:   r.Name,
				Charts: make([]repo.ChartVersions, 0),
			}
			for n := range i.Entries {
				log.Debugf("Chart: %s", n)
				chartMatched, _ := regexp.MatchString("^"+queryName+"$", strings.ToLower(n))

				kwString := strings.ToLower(strings.Join(i.Entries[n][0].Keywords, " "))
				log.Debugf("kwString: %s", kwString)

				kwMatched, _ := regexp.MatchString(queryKeyword, kwString)
				if (chartMatched || queryName == "") && (kwMatched || queryKeyword == "") {
					log.Debugf("Chart: %s Matched", n)
					if queryVersion == "latest" {
						c.Charts = append(c.Charts, repo.ChartVersions{i.Entries[n][0]})
					} else {
						c.Charts = append(c.Charts, i.Entries[n])
					}
				}

			}
			cl = append(cl, c)

		}
	}
	return cl, nil
}

// GetChart returns chart details
func GetChart(env helmEnv.EnvSettings, chartRepo, chartName, chartVersion string) (details *ChartDetails, err error) {

	repoPath := env.Home.RepositoryFile()
	log.Debugf("Helm repo path: %s", repoPath)
	var f *repo.RepoFile
	f, err = repo.LoadRepositoriesFile(repoPath)
	if err != nil {
		return
	}

	if len(f.Repositories) == 0 {
		return
	}

	for _, repository := range f.Repositories {

		log.Debugf("Repository: %s", repository.Name)

		var i *repo.IndexFile
		i, err = repo.LoadIndexFile(repository.Cache)
		if err != nil {
			return
		}

		details = &ChartDetails{
			Name: chartName,
			Repo: chartRepo,
		}

		if repository.Name == chartRepo {

			for name, chartVersions := range i.Entries {
				log.Debugf("Chart: %s", name)
				if chartName == name {
					for _, v := range chartVersions {

						if v.Version == chartVersion || chartVersion == "" {

							var ver *ChartVersion
							ver, err = getChartVersion(v)
							if err != nil {
								return
							}
							details.Versions = []*ChartVersion{ver}

							return
						} else if chartVersion == versionAll {
							var ver *ChartVersion
							ver, err = getChartVersion(v)
							if err != nil {
								log.Warnf("error during getting chart[%s - %s]: %s", v.Name, v.Version, err.Error())
							} else {
								details.Versions = append(details.Versions, ver)
							}

						}

					}
					return
				}

			}

		}
	}
	return
}

func getChartVersion(v *repo.ChartVersion) (*ChartVersion, error) {
	log.Infof("get chart[%s - %s]", v.Name, v.Version)

	chartSource := v.URLs[0]
	log.Debugf("chartSource: %s", chartSource)
	reader, err := DownloadFile(chartSource)
	if err != nil {
		return nil, err
	}
	valuesStr, err := GetChartFile(reader, "values.yaml")
	if err != nil {
		return nil, err
	}
	log.Debugf("values hash: %s", valuesStr)

	readmeStr, err := GetChartFile(reader, "README.md")
	if err != nil {
		return nil, err
	}

	return &ChartVersion{
		Chart:  v,
		Values: valuesStr,
		Readme: readmeStr,
	}, nil
}

// GetChartFile fetches a file from the chart.
func GetChartFile(file []byte, fileName string) (string, error) {
	tarReader := tar.NewReader(bytes.NewReader(file))

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		// We search for explicit path and the root directory is unknown.
		// Apply regexp (<anything>/filename prevent false match like /root_dir/chart/abc/README.md
		match, _ := regexp.MatchString("^([^/]*)/"+fileName+"$", header.Name)
		if match {
			fileContent, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return "", err
			}

			if filepath.Ext(fileName) == ".md" {
				log.Debugf("Security transform: %s", fileName)
				log.Debugf("Origin: %s", fileContent)

				fileContent = bluemonday.UGCPolicy().SanitizeBytes(fileContent)
			}

			base64File := base64.StdEncoding.EncodeToString(fileContent)

			return base64File, nil
		}
	}

	return "", nil
}

// DownloadChartFromRepo download a given chart
func DownloadChartFromRepo(name, version string, env helmEnv.EnvSettings) (string, error) {
	dl := downloader.ChartDownloader{
		HelmHome: env.Home,
		Getters:  getter.All(env),
	}
	if _, err := os.Stat(env.Home.Archive()); os.IsNotExist(err) {
		log.Infof("Creating '%s' directory.", env.Home.Archive())
		os.MkdirAll(env.Home.Archive(), 0744) // nolint: errcheck
	}

	log.Infof("Downloading helm chart %q, version %q to %q", name, version, env.Home.Archive())
	filename, _, err := dl.DownloadTo(name, version, env.Home.Archive())
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, errors.Wrapf(err, "Could not create absolute path from %s", filename)
		}
		log.Debugf("Fetched helm chart %q, version %q to %q", name, version, filename)
		return lname, nil
	}

	return filename, errors.Wrapf(err, "Failed to download chart %q, version %q", name, version)
}
