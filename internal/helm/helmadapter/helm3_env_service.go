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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/gofrs/flock"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"

	"github.com/banzaicloud/pipeline/internal/helm"
)

// helm3EnvService component struct for helm3 repository management
type helm3EnvService struct {
	logger Logger
}

func NewHelm3EnvService(logger Logger) helm.EnvService {
	return helm3EnvService{
		logger: logger,
	}
}

func (h helm3EnvService) AddRepository(ctx context.Context, helmEnv helm.HelmEnv, repository helm.Repository) error {
	repoFile := helmEnv.GetHome() // TODO add another field to the env instead???

	//Ensure the file directory exists as it is required for file locking
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
		return errors.NewWithDetails("repository name (%s) already exists, please specify a different name", repository.Name)
	}

	c := repo.Entry{ //TODO extend this with credentials
		Name: repository.Name,
		URL:  repository.URL,
		//InsecureSkipTLSverify: o.insecureSkipTLSverify,
	}

	envSettings := h.processEnvSettings(helmEnv)
	r, err := repo.NewChartRepository(&c, getter.All(envSettings))
	if err != nil {
		return err
	}

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

func (h helm3EnvService) ListRepositories(ctx context.Context, helmEnv helm.HelmEnv) ([]helm.Repository, error) {
	return []helm.Repository{}, nil
}

func (h helm3EnvService) DeleteRepository(ctx context.Context, helmEnv helm.HelmEnv, repoName string) error {
	repoFile := helmEnv.GetHome()
	r, err := repo.LoadFile(repoFile)
	if err != nil {
		if os.IsNotExist(errors.Cause(err)) || len(r.Repositories) == 0 {
			return errors.New("no repositories configured")
		}
	}

	if !r.Remove(repoName) {
		return errors.Errorf("no repo named %q found", repoName)
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
	settings := h.processEnvSettings(helmEnv)

	f, err := repo.LoadFile(helmEnv.GetHome())
	if os.IsNotExist(errors.Cause(err)) || len(f.Repositories) == 0 {
		return errors.New("no repositories found. You must add one before updating")
	}
	var repos []*repo.ChartRepository
	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(settings))
		if err != nil {
			return err
		}
		repos = append(repos, r)
	}

	h.updateCharts(repos, os.Stdin)

	return nil
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
