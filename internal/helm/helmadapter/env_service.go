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
	"k8s.io/helm/pkg/repo"

	"github.com/banzaicloud/pipeline/internal/helm"
	legacyHelm "github.com/banzaicloud/pipeline/src/helm"
)

// envService component implementing operations related to the helm environment
// This implementation relies on the legacy helm implementation
type envService struct {
	orgService  OrgService
	secretStore helm.SecretStore
	logger      Logger
}

func NewEnvService(orgService OrgService, secretStore helm.SecretStore, logger Logger) envService {
	return envService{
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
func (e envService) ListRepositories(ctx context.Context, organizationID uint) (repos []helm.Repository, err error) {
	// TODO revise this (should anything be returned here?)
	return nil, nil
}

// DeleteRepository deletes the  helm repository environment
func (e envService) DeleteRepository(ctx context.Context, organizationID uint, repoName string) error {
	orgName, err := e.orgService.GetOrgNameByOrgID(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to add repository")
	}

	helmEnv := legacyHelm.GenerateHelmRepoEnv(orgName)

	if err := legacyHelm.ReposDelete(helmEnv, repoName); err != nil {
		if err.Error() == legacyHelm.ErrRepoNotFound.Error() {
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
