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

package helm

import (
	"context"
	"time"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
)

type Logger = common.Logger

// Repository represents a Helm chart repository.
type Repository struct {
	// Name is a unique identifier for the repository.
	Name string `json:"name"`

	// URL is the repository URL.
	URL string `json:"url"`

	// PasswordSecretID is the identifier of a password type secret that contains the credentials for a repository.
	PasswordSecretID string `json:"passwordSecretId,omitempty"`

	// TlsSecretID is the identifier of a TLS secret.
	//
	// If there is a Certificate Authority in the secret,
	// it will be used to verify the certificate presented by the repository server.
	//
	// If there is a client key pair in the secret,
	// it will be presented to the repository server.
	TlsSecretID string `json:"tlsSecretId,omitempty"`
}

// ReleaseInfo copy of the struct form the helm library
type ReleaseInfo struct {

	// FirstDeployed is when the release was first deployed.
	FirstDeployed time.Time `json:"first_deployed,omitempty"`
	// LastDeployed is when the release was last deployed.
	LastDeployed time.Time `json:"last_deployed,omitempty"`
	// Deleted tracks when this object was deleted.
	Deleted time.Time `json:"deleted"`
	// Description is human-friendly "log entry" about this release.
	Description string `json:"description,omitempty"`
	// Status is the current state of the release
	Status string
	// Contains the rendered templates/NOTES.txt if available
	Notes string
}

//  Release represents information related to a helm chart release
type Release struct {
	// ReleaseInput struct encapsulating information about the release to be created
	ReleaseName     string
	ChartName       string
	Namespace       string
	Values          map[string]interface{} //json representation
	Version         string
	ReleaseInfo     ReleaseInfo
	ReleaserOptions ReleaserOptions
}

// +kit:endpoint:errorStrategy=service
// +testify:mock:testOnly=true

// Service manages Helm chart repositories.
type Service interface {
	// helm repository management operations
	repository

	// release management operations
	releaser
}

// releaser collects and groups release related operations
// it's intended to be embedded in the "Helm Facade"
type repository interface {
	// AddRepository adds a new Helm chart repository.
	AddRepository(ctx context.Context, organizationID uint, repository Repository) error
	// ListRepositories lists Helm repositories.
	ListRepositories(ctx context.Context, organizationID uint) (repos []Repository, err error)
	// ListRepositories deletes a Helm repository
	DeleteRepository(ctx context.Context, organizationID uint, repoName string) error
	// PatchRepository patches an existing repository
	PatchRepository(ctx context.Context, organizationID uint, repository Repository) error
	// UpdateRepository updates an existing repository
	UpdateRepository(ctx context.Context, organizationID uint, repository Repository) error
}

// releaser collects and groups release related operations
// it's intended to be embedded in the "Helm Facade"
type releaser interface {
	// Install installs the release to the cluster with the given identifier
	InstallRelease(ctx context.Context, organizationID uint, clusterID uint, release Release) error

	// Delete deletes the  specified release
	DeleteRelease(ctx context.Context, organizationID uint, clusterID uint, release Release) error

	// List retrieves  releases in a given namespace, eventually applies the passed in filters
	ListReleases(ctx context.Context, organizationID uint, clusterID uint, filters interface{}) ([]Release, error)

	//// Get retrieves the release details for the given  release
	//GetRelease(ctx context.Context, organizationID uint, clusterID uint, release Release) (Release, error)
	//
	//// GetResources
	//GetResources(ctx context.Context, organizationID uint, clusterID uint, release Release) (interface{}, error)
	//
	//// ReleaseStatus
	//ReleaseStatus(ctx context.Context, organizationID uint, clusterID uint, release Release) (Release, error)
	//
	//// Upgrade upgrades the given release
	//UpgradeRelease(ctx context.Context, organizationID uint, clusterID uint, release Release) error
}

// +testify:mock:testOnly=true

// Service manages Helm chart repositories.
type EnvService interface {
	// AddRepository adds a new Helm chart repository.
	AddRepository(ctx context.Context, helmEnv HelmEnv, repository Repository) error
	// ListRepositories lists Helm repositories.
	ListRepositories(ctx context.Context, helmEnv HelmEnv) (repos []Repository, err error)
	// ListRepositories deletes a Helm repository
	DeleteRepository(ctx context.Context, helmEnv HelmEnv, repoName string) error
	// PatchRepository patches an existing repository
	PatchRepository(ctx context.Context, helmEnv HelmEnv, repository Repository) error
	// UpdateRepository updates an existing repository
	UpdateRepository(ctx context.Context, helmEnv HelmEnv, repository Repository) error
}

// +testify:mock:testOnly=true

// Store interface abstracting persistence operations
type Store interface {
	// Create persists the repository item for the given organisation
	Create(ctx context.Context, organizationID uint, repository Repository) error
	// Delete persists the repository item for the given organisation
	Delete(ctx context.Context, organizationID uint, repository Repository) error
	//List retrieves persisted repositories for the given organisation
	List(ctx context.Context, organizationID uint) ([]Repository, error)
	//Getretrieves a repository entry
	Get(ctx context.Context, organizationID uint, repository Repository) (Repository, error)
	// Patch patches the given repository
	Patch(ctx context.Context, organizationID uint, repository Repository) error
	// Update patches the given repository
	Update(ctx context.Context, organizationID uint, repository Repository) error
}

type PasswordSecret struct {
	UserName string
	Password string
}

type TlsSecret struct {
	CAFile   string
	CertFile string
	KeyFile  string
}

// +testify:mock:testOnly=true

// SecretStore abstracts secret related operations
type SecretStore interface {
	// CheckPasswordSecret checks the existence and the type of the secret
	CheckPasswordSecret(ctx context.Context, secretID string) error
	// CheckTLSSecret checks the existence and the type of the secret
	CheckTLSSecret(ctx context.Context, secretID string) error
	// ResolvePasswordSecrets resolves the password type secret values
	ResolvePasswordSecrets(ctx context.Context, secretID string) (PasswordSecret, error)
	// ResolveTlsSecrets resolves the tls type secret values
	ResolveTlsSecrets(ctx context.Context, secretID string) (TlsSecret, error)
}

// Cluster collects operations to extract  cluster related information
type ClusterService interface {
	// Retrieves the kuebernetes configuration as a slice of bytes
	GetKubeConfig(ctx context.Context, clusterID uint) ([]byte, error)
}

type service struct {
	store          Store
	secretStore    SecretStore
	repoValidator  RepoValidator
	envResolver    EnvResolver
	envService     EnvService
	releaser       Releaser
	clusterService ClusterService
	logger         Logger
}

// NewService returns a new Service.
func NewService(
	store Store,
	secretStore SecretStore,
	validator RepoValidator,
	envResolver EnvResolver,
	envService EnvService,
	releaser Releaser,
	clusterService ClusterService,
	logger Logger) Service {
	return service{
		store:          store,
		secretStore:    secretStore,
		repoValidator:  validator,
		envResolver:    envResolver,
		envService:     envService,
		releaser:       releaser,
		clusterService: clusterService,
		logger:         logger,
	}
}

func (s service) AddRepository(ctx context.Context, organizationID uint, repository Repository) error {
	if err := s.repoValidator.Validate(ctx, repository); err != nil {
		return errors.WrapIf(err, "failed to add new helm repository")
	}

	if repository.PasswordSecretID != "" {
		if err := s.secretStore.CheckPasswordSecret(ctx, repository.PasswordSecretID); err != nil {
			return ValidationError{message: err.Error(), violations: []string{"password secret must exist"}}
		}
	}

	if repository.TlsSecretID != "" {
		if err := s.secretStore.CheckTLSSecret(ctx, repository.TlsSecretID); err != nil {
			return ValidationError{message: err.Error(), violations: []string{"tls secret must exist"}}
		}
	}

	exists, err := s.repoExists(ctx, organizationID, repository)
	if err != nil {
		return errors.WrapIf(err, "failed to add helm repository")
	}

	if exists {
		return AlreadyExistsError{
			RepositoryName: repository.Name,
			OrganizationID: organizationID,
		}
	}

	if err := s.store.Create(ctx, organizationID, repository); err != nil {
		return errors.WrapIf(err, "failed to add helm repository")
	}

	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	if err := s.envService.AddRepository(ctx, helmEnv, repository); err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	s.logger.Debug("created helm repository", map[string]interface{}{"orgID": organizationID, "helm repository": repository.Name})
	return nil
}

func (s service) ListRepositories(ctx context.Context, organizationID uint) (repos []Repository, err error) {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to set up helm repository environment")
	}

	defaultRepos, err := s.envService.ListRepositories(ctx, helmEnv)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve default repositories")
	}

	persistedRepos, err := s.store.List(ctx, organizationID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve persisted repositories")
	}

	return mergeDefaults(defaultRepos, persistedRepos), nil
}

func (s service) DeleteRepository(ctx context.Context, organizationID uint, repoName string) error {
	repoExists, err := s.repoExists(ctx, organizationID, Repository{Name: repoName})
	if err != nil {
		return err
	}

	if !repoExists {
		return nil
	}

	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	if err := s.envService.DeleteRepository(ctx, helmEnv, repoName); err != nil {
		return errors.WrapIf(err, "failed to delete helm repository environment")
	}

	if err := s.store.Delete(ctx, organizationID, Repository{Name: repoName}); err != nil {
		return errors.WrapIf(err, "failed to delete helm repository")
	}

	s.logger.Debug("deleted helm repository", map[string]interface{}{"orgID": organizationID, "helm repository": repoName})
	return nil
}

func (s service) PatchRepository(ctx context.Context, organizationID uint, repository Repository) error {
	if repository.PasswordSecretID != "" {
		if err := s.secretStore.CheckPasswordSecret(ctx, repository.PasswordSecretID); err != nil {
			return ValidationError{message: err.Error(), violations: []string{"password secret must exist"}}
		}
	}

	if repository.TlsSecretID != "" {
		if err := s.secretStore.CheckTLSSecret(ctx, repository.TlsSecretID); err != nil {
			return ValidationError{message: err.Error(), violations: []string{"tls secret must exist"}}
		}
	}

	exists, err := s.repoExists(ctx, organizationID, Repository{Name: repository.Name})
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to retrieve helm repository",
			"orgID", organizationID, "repoName", repository.Name)
	}

	if !exists {
		return NotFoundError{
			RepositoryName: repository.Name,
			OrganizationID: organizationID,
		}
	}

	if err := s.store.Patch(ctx, organizationID, repository); err != nil {
		return errors.WrapIf(err, "failed to add helm repository")
	}

	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	if err := s.envService.PatchRepository(ctx, helmEnv, repository); err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	s.logger.Debug("created helm repository", map[string]interface{}{"orgID": organizationID, "helm repository": repository.Name})
	return nil
}

func (s service) UpdateRepository(ctx context.Context, organizationID uint, repository Repository) error {
	if err := s.repoValidator.Validate(ctx, repository); err != nil {
		return errors.WrapIf(err, "failed to add new helm repository")
	}

	if repository.PasswordSecretID != "" {
		if err := s.secretStore.CheckPasswordSecret(ctx, repository.PasswordSecretID); err != nil {
			return ValidationError{message: err.Error(), violations: []string{"password secret must exist"}}
		}
	}

	if repository.TlsSecretID != "" {
		if err := s.secretStore.CheckTLSSecret(ctx, repository.TlsSecretID); err != nil {
			return ValidationError{message: err.Error(), violations: []string{"tls secret must exist"}}
		}
	}

	exists, err := s.repoExists(ctx, organizationID, Repository{Name: repository.Name})
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to retrieve helm repository",
			"orgID", organizationID, "repoName", repository.Name)
	}

	if !exists {
		return NotFoundError{
			RepositoryName: repository.Name,
			OrganizationID: organizationID,
		}
	}

	if err := s.store.Update(ctx, organizationID, repository); err != nil {
		return errors.WrapIf(err, "failed to add helm repository")
	}

	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	if err := s.envService.PatchRepository(ctx, helmEnv, repository); err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	s.logger.Debug("created helm repository", map[string]interface{}{"orgID": organizationID, "helm repository": repository.Name})
	return nil
}

func (s service) InstallRelease(ctx context.Context, organizationID uint, clusterID uint, release Release) error {
	// TODO should this come from the api?
	releaserOptions := ReleaserOptions{}

	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	kubeKonfig, err := s.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster configuration")
	}

	if _, err := s.releaser.Install(ctx, helmEnv, kubeKonfig, release, releaserOptions); err != nil {
		return errors.WrapIf(err, "failed to install release")
	}

	return nil
}

func (s service) DeleteRelease(ctx context.Context, organizationID uint, clusterID uint, release Release) error {
	// TODO should this come from the api?
	releaserOptions := ReleaserOptions{}

	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	kubeKonfig, err := s.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster configuration")
	}

	if err := s.releaser.Uninstall(ctx, helmEnv, kubeKonfig, release, releaserOptions); err != nil {
		return errors.WrapIf(err, "failed to uninstall release")
	}

	return nil
}

func (s service) ListReleases(ctx context.Context, organizationID uint, clusterID uint, filters interface{}) ([]Release, error) {
	releaserOptions := ReleaserOptions{}

	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to set up helm repository environment")
	}

	kubeKonfig, err := s.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster configuration")
	}

	releases, err := s.releaser.List(ctx, helmEnv, kubeKonfig, releaserOptions)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to list releases")
	}

	return releases, nil
}

func (s service) repoExists(ctx context.Context, orgID uint, repository Repository) (bool, error) {
	_, err := s.store.Get(ctx, orgID, repository)

	if err != nil {
		// TODO refine this implementation, separate results by error type
		return false, nil
	}

	return true, nil
}

// mergeDefaults adds the defaults to the list of repositories if not already added
func mergeDefaults(defaultRepos []Repository, storedRepos []Repository) []Repository {
	merged := storedRepos
	for _, defaultRepo := range defaultRepos {
		if !contains(defaultRepo.Name, storedRepos) {
			merged = append(merged, defaultRepo)
		}
	}
	return merged
}

func contains(repoName string, repos []Repository) bool {
	for _, repo := range repos {
		if repo.Name == repoName {
			return true
		}
	}
	return false
}
