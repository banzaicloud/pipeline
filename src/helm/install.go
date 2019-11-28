// Copyright © 2018 Banzai Cloud
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
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	helmEnv "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"

	"github.com/banzaicloud/pipeline/internal/global"
	phelm "github.com/banzaicloud/pipeline/pkg/helm"
)

// CreateEnvSettings Create env settings on a given path
func CreateEnvSettings(helmRepoHome string) helmEnv.EnvSettings {
	var settings helmEnv.EnvSettings
	settings.Home = helmpath.Home(helmRepoHome)
	return settings
}

// GenerateHelmRepoEnv Generate helm path based on orgName
func GenerateHelmRepoEnv(orgName string) (env helmEnv.EnvSettings) {
	var helmPath = global.GetHelmPath(orgName)
	env = CreateEnvSettings(fmt.Sprintf("%s/%s", helmPath, phelm.HelmPostFix))

	// check local helm
	if _, err := os.Stat(helmPath); os.IsNotExist(err) {
		log.Infof("Helm directories [%s] not exists", helmPath)
		InstallLocalHelm(env) // nolint: errcheck
	}

	return
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

// InstallHelmClient Installs helm client on a given path
func InstallHelmClient(env helmEnv.EnvSettings) error {
	if err := EnsureDirectories(env); err != nil {
		return errors.Wrap(err, "Initializing helm directories failed!")
	}

	log.Info("Initializing helm client succeeded, happy helming!")
	return nil
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
	var repos = []struct {
		name string
		url  string
	}{
		{
			name: phelm.StableRepository,
			url:  global.Config.Helm.Repositories[phelm.StableRepository],
		},
		{
			name: phelm.BanzaiRepository,
			url:  global.Config.Helm.Repositories[phelm.BanzaiRepository],
		},
		{
			name: phelm.LokiRepository,
			url:  global.Config.Helm.Repositories[phelm.LokiRepository],
		},
	}

	log.Infof("Setting up default helm repos.")

	for _, r := range repos {
		_, err := ReposAdd(
			env,
			&repo.Entry{
				Name:  r.name,
				URL:   r.url,
				Cache: env.Home.CacheIndex(r.name),
			})
		if err != nil {
			return errors.Wrapf(err, "cannot init repo: %s", r.name)
		}
	}

	return nil
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
