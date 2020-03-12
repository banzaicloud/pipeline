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
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"

	"github.com/banzaicloud/pipeline/internal/helm"
	legacyHelm "github.com/banzaicloud/pipeline/src/helm"
)

// OrgService interface for decoupling organization related operations
type OrgService interface {
	// GetOrgNameByOrgID retrieves organization name for the provided ID
	GetOrgNameByOrgID(ctx context.Context, orgID uint) (string, error)
}

// Helm related configurations
type Config struct {
	Repositories map[string]string
}

func NewConfig(defaultRepos map[string]string) Config {
	return Config{Repositories: defaultRepos}
}

// envService component implementing operations related to the helm environment
// This implementation relies on the legacy helm implementation
type envService struct {
	config      Config
	orgService  OrgService
	secretStore helm.SecretStore
	logger      Logger
}

func NewEnvService(config Config, orgService OrgService, secretStore helm.SecretStore, logger Logger) helm.Service {
	return envService{
		config:      config,
		orgService:  orgService,
		secretStore: secretStore,
		logger:      logger,
	}
}

// AddRepository sets up the environment for the passed in repository
func (e envService) AddRepository(ctx context.Context, organizationID uint, repository helm.Repository) error {
	orgName, err := e.orgService.GetOrgNameByOrgID(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to add repository")
	}

	helmEnv := legacyHelm.GenerateHelmRepoEnv(orgName)

	entry, err := e.transform(ctx, repository)
	if err != nil {
		return errors.WrapIf(err, "failed to resolve helm entry data")
	}

	_, err = legacyHelm.ReposAdd(helmEnv, &entry)
	if err != nil {
		return errors.WrapIf(err, "failed to set up environment for repository")
	}

	return nil
}

// ListRepositories noop implementation (env details not returned
func (e envService) ListRepositories(_ context.Context, organizationID uint) (repos []helm.Repository, err error) {
	defaultRepos := make([]helm.Repository, 0, len(e.config.Repositories))
	for name, repository := range e.config.Repositories {
		defaultRepos = append(defaultRepos, helm.Repository{
			Name: name,
			URL:  repository,
		})
	}

	return defaultRepos, nil
}

// DeleteRepository deletes the  helm repository environment
func (e envService) DeleteRepository(ctx context.Context, organizationID uint, repoName string) error {
	orgName, err := e.orgService.GetOrgNameByOrgID(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to add repository")
	}

	helmEnv := legacyHelm.GenerateHelmRepoEnv(orgName)

	if err := legacyHelm.ReposDelete(helmEnv, repoName); err != nil {
		if errors.Cause(err).Error() == legacyHelm.ErrRepoNotFound.Error() {
			return nil
		}

		return errors.WrapIf(err, "failed to delete helm repository environment")
	}

	return nil
}

func (e envService) PatchRepository(ctx context.Context, organizationID uint, repository helm.Repository) error {
	orgName, err := e.orgService.GetOrgNameByOrgID(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to add repository")
	}

	helmEnv := legacyHelm.GenerateHelmRepoEnv(orgName)

	entry, err := e.transform(ctx, repository)
	if err != nil {
		return errors.WrapIf(err, "failed to resolve helm entry data")
	}

	if err := legacyHelm.ReposModify(helmEnv, repository.Name, &entry); err != nil {
		return errors.WrapIf(err, "failed to update helm repository environment")
	}

	return nil
}

func (e envService) UpdateRepository(ctx context.Context, organizationID uint, repository helm.Repository) error {
	return e.PatchRepository(ctx, organizationID, repository)
}

func (e envService) transform(ctx context.Context, repository helm.Repository) (repo.Entry, error) {
	entry := repo.Entry{
		Name: repository.Name,
		URL:  repository.URL,
	}

	if repository.PasswordSecretID != "" {
		passwordSecrets, passErr := e.secretStore.ResolvePasswordSecrets(ctx, repository.PasswordSecretID)
		if passErr != nil {
			return repo.Entry{}, errors.WrapIf(passErr, "failed to transform password values")
		}

		entry.Username = passwordSecrets.UserName
		entry.Password = passwordSecrets.Password
	}

	if repository.TlsSecretID != "" {
		// TODO tls support needs to be finalized here (too)
		tlsSecrets, tlsErr := e.secretStore.ResolveTlsSecrets(ctx, repository.TlsSecretID)
		if tlsErr != nil {
			return repo.Entry{}, errors.WrapIf(tlsErr, "failed to transform tls values")
		}

		entry.CAFile = tlsSecrets.CAFile
		entry.CertFile = tlsSecrets.CertFile
		entry.KeyFile = tlsSecrets.KeyFile
	}

	return entry, nil
}

// helmEnvService component in charge to operate the helm env on the filesystem
type helmEnvService struct {
	logger Logger
}

func NewHelmEnvService(logger Logger) helm.EnvService {
	return helmEnvService{logger: logger}
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

func (h helmEnvService) ListRepositories(_ context.Context, helmEnv helm.HelmEnv) (repos []helm.Repository, err error) {
	h.logger.Debug("returning empty helm repository list", map[string]interface{}{"helmEnv": helmEnv.GetHome()})

	// no data from the env returned
	return []helm.Repository{}, nil
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

	entry, err := h.repositoryToEntry(repository)
	if err != nil {
		return errors.WrapIf(err, "failed to resolve helm entry data")
	}

	if err = legacyHelm.ReposModify(envSettings, repository.Name, &entry); err != nil {
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
