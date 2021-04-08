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

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
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

// Options struct holding directives for driving helm operations (similar to command line flags)
// extend this as required eventually build a more sophisticated solution for it
type Options struct {
	Namespace    string  `json:"namespace,omitempty"`
	DryRun       bool    `json:"dryRun,omitempty"`
	GenerateName bool    `json:"generateName,omitempty"`
	Wait         bool    `json:"wait,omitempty"`
	Timeout      int64   `json:"timeout,omitempty"`
	ReuseValues  bool    `json:"reuseValues,omitempty"`
	Install      bool    `json:"install,omitempty"`
	Filter       *string `json:"filter,omitempty"`
	SkipCRDs     bool    `json:"skipCRDs,omitempty"`
}

// +kit:endpoint:errorStrategy=service
// +testify:mock:testOnly=true

// Service manages Helm repositories, charts and releases
type Service interface {
	// helm repository management operations
	repository

	// release management operations
	releaser

	// chart related operations
	charter
}

type ClusterDataProvider interface {
	GetK8sConfig() ([]byte, error)
	GetID() uint
}

// UnifiedReleaser unifies different helm release interfaces into a single interface
type UnifiedReleaser interface {
	// integrated services style
	ApplyDeployment(
		ctx context.Context,
		clusterID uint,
		namespace string,
		chartName string,
		releaseName string,
		values []byte,
		upgradeschartVersion string,
	) error

	ApplyDeploymentReuseValues(
		ctx context.Context,
		clusterID uint,
		namespace string,
		chartName string,
		releaseName string,
		values []byte,
		chartVersion string,
		reuseValues bool,
	) error

	ApplyDeploymentSkipCRDs(
		ctx context.Context,
		clusterID uint,
		namespace string,
		chartName string,
		releaseName string,
		values []byte,
		upgradeschartVersion string,
	) error

	// cluster setup style
	InstallDeployment(
		ctx context.Context,
		clusterID uint,
		namespace string,
		chartName string,
		releaseName string,
		values []byte,
		chartVersion string,
		wait bool,
	) error

	// DeleteDeployment deletes a deployment from a specific cluster.
	DeleteDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) error

	// GetDeployment gets a deployment by release name from a specific cluster.
	GetDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) (*pkgHelm.GetDeploymentResponse, error)

	// Covers Federation and Backyards style implementation
	InstallOrUpgrade(
		orgID uint,
		c ClusterDataProvider,
		release Release,
		opts Options,
	) error

	GetRelease(c ClusterDataProvider, releaseName, namespace string) (Release, error)

	Delete(c ClusterDataProvider, releaseName, namespace string) error
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
	// ModifyRepository overwrites an existing repository with new values
	ModifyRepository(ctx context.Context, organizationID uint, repository Repository) error
	// UpdateRepository updates an existing repository
	UpdateRepository(ctx context.Context, organizationID uint, repository Repository) error
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
	// ListCharts lists charts matching the given filter
	ListCharts(ctx context.Context, helmEnv HelmEnv, chartFilter ChartFilter) (chartList ChartList, err error)
	// GetChart retrieves the details of the passed in chart
	GetChart(ctx context.Context, helmEnv HelmEnv, chartFilter ChartFilter) (chartDetails ChartDetails, err error)
	// CheckReleaseCharts checks whether the charts for the passed in release can be found in the org's helm env
	CheckReleaseCharts(ctx context.Context, helmEnv HelmEnv, releases []Release) (map[string]bool, error)

	// EnsureEnv ensures the helm environment represented by the input.
	// If theh environment exists (on the filesystem) it does nothing
	EnsureEnv(ctx context.Context, helmEnv HelmEnv, defaultRepos []Repository) (HelmEnv, bool, error)
}

// +testify:mock:testOnly=true

// Store interface abstracting persistence operations
type Store interface {
	// Create persists the repository item for the given organisation
	Create(ctx context.Context, organizationID uint, repository Repository) error
	// Delete persists the repository item for the given organisation
	Delete(ctx context.Context, organizationID uint, repository Repository) error
	// List retrieves persisted repositories for the given organisation
	List(ctx context.Context, organizationID uint) ([]Repository, error)
	// Getretrieves a repository entry
	Get(ctx context.Context, organizationID uint, repository Repository) (Repository, error)
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

type ClusterKubeConfigFunc func(ctx context.Context, clusterID uint) ([]byte, error)

func (c ClusterKubeConfigFunc) GetKubeConfig(ctx context.Context, clusterID uint) ([]byte, error) {
	return c(ctx, clusterID)
}

type service struct {
	config         Config
	clusterCharts  []ChartConfig
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
	config Config,
	clusterCharts []ChartConfig,
	store Store,
	secretStore SecretStore,
	validator RepoValidator,
	envResolver EnvResolver,
	envService EnvService,
	releaser Releaser,
	clusterService ClusterService,
	logger Logger) Service {
	return service{
		config:         config,
		clusterCharts:  clusterCharts,
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

	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	exists, err := s.repoExists(ctx, repository, helmEnv)
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

	envRepos, err := s.envService.ListRepositories(ctx, helmEnv)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve default repositories")
	}

	return s.decorateRepos(ctx, organizationID, envRepos), nil
}

func (s service) DeleteRepository(ctx context.Context, organizationID uint, repoName string) error {
	for defaultRepoName := range s.config.Repositories {
		if defaultRepoName == repoName {
			return NewValidationError("default repositories cannot be deleted", nil)
		}
	}

	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	repoExists, err := s.repoExists(ctx, Repository{Name: repoName}, helmEnv)
	if err != nil {
		return err
	}

	if !repoExists {
		return nil
	}

	// Remove from store first so that the call can be retried on failure
	if err := s.store.Delete(ctx, organizationID, Repository{Name: repoName}); err != nil {
		return errors.WrapIf(err, "failed to delete helm repository")
	}

	if err := s.envService.DeleteRepository(ctx, helmEnv, repoName); err != nil {
		return errors.WrapIf(err, "failed to delete helm repository environment")
	}

	s.logger.Debug("deleted helm repository", map[string]interface{}{"orgID": organizationID, "helm repository": repoName})
	return nil
}

func (s service) ModifyRepository(ctx context.Context, organizationID uint, repository Repository) error {
	for repoName := range s.config.Repositories {
		if repoName == repository.Name {
			return NewValidationError("default repositories cannot be modified", nil)
		}
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

	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to resolve helm repository environment")
	}

	exists, err := s.repoExists(ctx, Repository{Name: repository.Name}, helmEnv)
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

	if err := s.envService.UpdateRepository(ctx, helmEnv, repository); err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	s.logger.Debug("created helm repository", map[string]interface{}{"orgID": organizationID, "helm repository": repository.Name})
	return nil
}

func (s service) UpdateRepository(ctx context.Context, organizationID uint, repository Repository) error {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to resolve helm repository environment")
	}

	// repo exists under the orgs helm env
	exists, err := s.repoExists(ctx, Repository{Name: repository.Name}, helmEnv)
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

	repoToUpdate, err := s.getRepoForUpdate(ctx, organizationID, repository)
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to retrieve repository to update",
			"orgID", organizationID, "repoName", repository.Name)
	}

	if err := s.envService.UpdateRepository(ctx, helmEnv, repoToUpdate); err != nil {
		return errors.WrapIfWithDetails(err, "failed to update repository",
			"orgID", organizationID, "repoName", repository.Name)
	}

	s.logger.Debug("helm repository successfully updated", map[string]interface{}{"orgID": organizationID, "helm repository": repository.Name})
	return nil
}

func (s service) InstallRelease(ctx context.Context, organizationID uint, clusterID uint, releaseInput Release, options Options) (release Release, err error) {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return release, errors.WrapIf(err, "failed to set up helm repository environment")
	}

	kubeKonfig, err := s.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return release, errors.WrapIf(err, "failed to get cluster configuration")
	}

	release, err = s.releaser.Install(ctx, helmEnv, kubeKonfig, releaseInput, options)
	if err != nil {
		return Release{}, errors.WrapIf(err, "failed to install release")
	}

	return release, nil
}

func (s service) DeleteRelease(ctx context.Context, organizationID uint, clusterID uint, releaseName string, options Options) error {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	kubeKonfig, err := s.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster configuration")
	}

	if err := s.releaser.Uninstall(ctx, helmEnv, kubeKonfig, releaseName, options); err != nil {
		return errors.WrapIf(err, "failed to uninstall release")
	}

	return nil
}

func (s service) ListReleases(ctx context.Context, organizationID uint, clusterID uint, filters ReleaseFilter, options Options) ([]Release, error) {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to set up helm repository environment")
	}

	kubeKonfig, err := s.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster configuration")
	}

	if filters.Filter != nil {
		options.Filter = filters.Filter
	}
	releases, err := s.releaser.List(ctx, helmEnv, kubeKonfig, options)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to list releases")
	}

	return releases, nil
}

func (s service) GetRelease(ctx context.Context, organizationID uint, clusterID uint, releaseName string, options Options) (Release, error) {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return Release{}, errors.WrapIf(err, "failed to set up helm repository environment")
	}

	kubeKonfig, err := s.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return Release{}, errors.WrapIf(err, "failed to get cluster configuration")
	}

	input := Release{ReleaseName: releaseName}
	release, err := s.releaser.Get(ctx, helmEnv, kubeKonfig, input, options)
	if err != nil {
		return Release{}, errors.WrapIfWithDetails(err, "failed to get release", "releaseName", releaseName)
	}

	return release, nil
}

func (s service) UpgradeRelease(ctx context.Context, organizationID uint, clusterID uint, releaseInput Release, options Options) (Release, error) {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return Release{}, errors.WrapIf(err, "failed to set up helm repository environment")
	}

	kubeKonfig, err := s.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return Release{}, errors.WrapIf(err, "failed to get cluster configuration")
	}

	release, err := s.releaser.Upgrade(ctx, helmEnv, kubeKonfig, releaseInput, options)
	if err != nil {
		return Release{}, errors.WrapIfWithDetails(err, "failed to upgrade release", "releaseName", releaseInput.ReleaseName)
	}

	return release, nil
}

func (s service) ListCharts(ctx context.Context, organizationID uint, filter ChartFilter, _ Options) (charts ChartList, err error) {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to set up helm repository environment")
	}

	chartList, err := s.envService.ListCharts(ctx, helmEnv, filter)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to list charts")
	}

	return chartList, nil
}

func (s service) GetChart(ctx context.Context, organizationID uint, chartFilter ChartFilter, _options Options) (chartDetails ChartDetails, err error) {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to set up helm repository environment")
	}

	details, err := s.envService.GetChart(ctx, helmEnv, chartFilter)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get helm chart details")
	}

	if len(details) == 0 {
		return nil, ChartNotFoundError{
			ChartInfo: chartFilter.String(),
			OrgID:     organizationID,
		}
	}

	return details, nil
}

// ListClusterCharts lists the Helm charts (with details) currently available
// for Pipeline managed clusters.
func (s service) ListClusterCharts(ctx context.Context, _ uint, _ Options) (charts ChartList, err error) {
	for _, clusterChart := range s.clusterCharts {
		charts = append(charts, clusterChart)
	}

	return charts, nil
}

func (s service) GetReleaseResources(ctx context.Context, organizationID uint, clusterID uint, release Release, options Options) ([]ReleaseResource, error) {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to set up helm repository environment")
	}

	kubeKonfig, err := s.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster configuration")
	}

	resources, err := s.releaser.Resources(ctx, helmEnv, kubeKonfig, release, options)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to retrieve release resources ", "releaseName", release.ReleaseName)
	}

	return resources, nil
}

func (s service) CheckRelease(ctx context.Context, organizationID uint, clusterID uint, releaseName string, options Options) (string, error) {
	release, err := s.GetRelease(ctx, organizationID, clusterID, releaseName, options)
	if err != nil {
		return "", errors.WrapIf(err, "failed to retrieve release")
	}

	return release.ReleaseInfo.Status, nil
}

func (s service) CheckReleases(ctx context.Context, organizationID uint, releases []Release) (map[string]bool, error) {
	helmEnv, err := s.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to resolve helm env releases")
	}

	supportedChartMap, err := s.envService.CheckReleaseCharts(ctx, helmEnv, releases)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve charts")
	}

	return supportedChartMap, nil
}

func (s service) repoExists(ctx context.Context, repository Repository, helmEnv HelmEnv) (bool, error) {
	repos, err := s.envService.ListRepositories(ctx, helmEnv)
	if err != nil {
		// TODO refine this implementation, separate results by error type
		return false, nil
	}

	for _, r := range repos {
		if r.Name == repository.Name {
			return true, nil
		}
	}

	return false, nil
}

// decorateRepos retrieves secretReferences for the repo
func (s service) decorateRepos(ctx context.Context, orgID uint, repos []Repository) []Repository {
	persistedRepos, err := s.store.List(ctx, orgID)
	if err != nil {
		s.logger.Warn("failed to decorate repositories with secret references")
		return repos
	}

	if len(persistedRepos) == 0 {
		s.logger.Debug("no persisted repos found, no secret references to add to the repo list")
		return repos
	}

	decorated := make([]Repository, 0, len(repos))
	for _, repo := range repos {
		for _, persistedRepo := range persistedRepos {
			if repo.Name == persistedRepo.Name {
				repo.PasswordSecretID = persistedRepo.PasswordSecretID
			}
		}
		decorated = append(decorated, repo)
	}

	return decorated
}

func (s service) getRepoForUpdate(ctx context.Context, orgID uint, repository Repository) (Repository, error) {
	repoURL, ok := s.config.Repositories[repository.Name]
	if ok {
		s.logger.Debug("updating builtin helm repo", map[string]interface{}{"repoName": repository.Name})
		return Repository{
			Name: repository.Name,
			URL:  repoURL,
		}, nil
	}

	repo, err := s.store.Get(ctx, orgID, repository)
	if err != nil {
		return Repository{}, errors.WrapIf(err, "failed to get persisted repo for update")
	}
	s.logger.Debug("updating org helm repo", map[string]interface{}{"repoName": repository.Name})
	return repo, nil
}
