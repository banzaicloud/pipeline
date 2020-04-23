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

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"

	"github.com/banzaicloud/pipeline/internal/helm"
	legacyHelm "github.com/banzaicloud/pipeline/src/helm"
)

// Helm related configurations
type Config struct {
	Repositories map[string]string
}

func NewConfig(defaultRepos map[string]string) Config {
	return Config{Repositories: defaultRepos}
}

// helmEnvService component in charge to operate the helm env on the filesystem
type helmEnvService struct {
	config Config
	logger Logger
}

func NewHelmEnvService(config Config, logger Logger) helm.EnvService {
	return helmEnvService{
		config: config,
		logger: logger,
	}
}

func (h helmEnvService) ListCharts(_ context.Context, helmEnv helm.HelmEnv, filter helm.ChartFilter) (helm.ChartList, error) {
	envSettings := environment.EnvSettings{Home: helmpath.Home(helmEnv.GetHome())}

	legacyChartSlice, err := legacyHelm.ChartsGet(envSettings, filter.StrictNameFilter(), filter.RepoFilter(), filter.VersionFilter(), filter.KeywordFilter())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get chart list")
	}

	// transform the list
	var chartList helm.ChartList
	for _, legacyChart := range legacyChartSlice {
		chartList = append(chartList, legacyChart)
	}

	h.logger.Debug("successfully retrieved chart list", map[string]interface{}{"filter": filter})
	return chartList, nil
}

func (h helmEnvService) GetChart(_ context.Context, helmEnv helm.HelmEnv, filter helm.ChartFilter) (helm.ChartDetails, error) {
	envSettings := environment.EnvSettings{Home: helmpath.Home(helmEnv.GetHome())}

	legacyChart, err := legacyHelm.ChartGet(envSettings, filter.RepoFilter(), filter.NameFilter(), filter.VersionFilter())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get chart list")
	}

	var chart helm.ChartDetails
	if err := mapstructure.Decode(*legacyChart, &chart); err != nil {
		return nil, errors.WrapIf(err, "failed to transform legacy chart details")
	}

	h.logger.Debug("successfully retrieved chart details", map[string]interface{}{"filter": filter})
	return chart, nil
}

func (h helmEnvService) AddRepository(_ context.Context, helmEnv helm.HelmEnv, repository helm.Repository) error {
	envSettings := environment.EnvSettings{Home: helmpath.Home(helmEnv.GetHome())}

	if err := legacyHelm.EnsureDirectories(envSettings); err != nil {
		return errors.WrapIfWithDetails(err, "failed to install helm environment", "path", helmEnv.GetHome())
	}

	entry, err := h.repositoryToEntry(repository)
	if err != nil {
		return errors.WrapIf(err, "failed to resolve helm entry data")
	}

	if _, err = legacyHelm.ReposAdd(envSettings, &entry); err != nil {
		return errors.WrapIf(err, "failed to set up environment for repository")
	}

	h.logger.Debug("helm repository successfully added", map[string]interface{}{"helmEnv": helmEnv.GetHome(),
		"repository": repository.Name})
	return nil
}

func (h helmEnvService) ListRepositories(_ context.Context, helmEnv helm.HelmEnv) ([]helm.Repository, error) {
	h.logger.Debug("returning default helm repository list", map[string]interface{}{"helmEnv": helmEnv.GetHome()})

	envSettings := environment.EnvSettings{Home: helmpath.Home(helmEnv.GetHome())}

	filesystemRepos, err := legacyHelm.ReposGet(envSettings)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to list helm repos")
	}

	repos := make([]helm.Repository, 0, len(filesystemRepos))
	for _, r := range filesystemRepos {
		repos = append(repos, helm.Repository{
			Name: r.Name,
			URL:  r.URL,
		})
	}
	return repos, nil
}

func (h helmEnvService) DeleteRepository(_ context.Context, helmEnv helm.HelmEnv, repoName string) error {
	envSettings := environment.EnvSettings{Home: helmpath.Home(helmEnv.GetHome())}

	if err := legacyHelm.ReposDelete(envSettings, repoName); err != nil {
		if errors.Cause(err).Error() == legacyHelm.ErrRepoNotFound.Error() {
			return nil
		}

		return errors.WrapIf(err, "failed to remove helm repository")
	}

	h.logger.Debug("helm repository successfully removed", map[string]interface{}{"helmEnv": helmEnv.GetHome()})
	return nil
}

func (h helmEnvService) PatchRepository(_ context.Context, helmEnv helm.HelmEnv, repository helm.Repository) error {
	envSettings := environment.EnvSettings{Home: helmpath.Home(helmEnv.GetHome())}

	entry, err := h.repositoryToEntry(repository)
	if err != nil {
		return errors.WrapIf(err, "failed to resolve helm entry data")
	}

	if err = legacyHelm.ReposModify(envSettings, repository.Name, &entry); err != nil {
		return errors.WrapIf(err, "failed to set up environment for repository")
	}

	h.logger.Debug("helm repository successfully patched", map[string]interface{}{"helmEnv": helmEnv.GetHome(),
		"repository": repository.Name})
	return nil
}

func (h helmEnvService) UpdateRepository(_ context.Context, helmEnv helm.HelmEnv, repository helm.Repository) error {
	envSettings := environment.EnvSettings{Home: helmpath.Home(helmEnv.GetHome())}

	if err := legacyHelm.ReposUpdate(envSettings, repository.Name); err != nil {
		return errors.WrapIf(err, "failed to set up environment for repository")
	}

	h.logger.Debug("helm repository successfully updated", map[string]interface{}{"helmEnv": helmEnv.GetHome(),
		"repository": repository.Name})
	return nil
}

func (h helmEnvService) repositoryToEntry(repository helm.Repository) (repo.Entry, error) {
	entry := repo.Entry{
		Name: repository.Name,
		URL:  repository.URL,
	}

	return entry, nil
}

func (h helmEnvService) EnsureEnv(_ context.Context, helmEnv helm.HelmEnv, defaultRepos []helm.Repository) (helm.HelmEnv, error) {
	_, err := legacyHelm.GenerateHelmRepoEnvOnPath(helmEnv.GetHome())
	return helmEnv, err
}
