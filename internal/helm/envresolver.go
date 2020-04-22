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
	"path"

	"emperror.dev/errors"
)

const (
	PlatformHelmHome = "pipeline"
	helmPostFix      = "helm"
	noOrg            = 0 // signals that no organization id is provided
)

// +testify:mock:testOnly=true

// OrgService interface for decoupling organization related operations
type OrgService interface {
	// GetOrgNameByOrgID retrieves organization name for the provided ID
	GetOrgNameByOrgID(ctx context.Context, orgID uint) (string, error)
}

// HelmEnv helm environment settings abstraction
type HelmEnv struct {
	// home path pointing to a helm home
	home string

	// platform signals whether the instance represents a platform environment (as opposed to an org bound one)
	platform bool

	repoCacheDir string
}

func (e HelmEnv) GetHome() string {
	return e.home
}

func (e HelmEnv) IsPlatform() bool {
	return e.platform
}

func (e HelmEnv) GetRepoCache() string {
	return e.repoCacheDir
}

// +testify:mock:testOnly=true

// HelmEnvResolver interface to abstract resolving helm homes
type EnvResolver interface {
	// ResolveHelmEnv resolves the helm home for the passed in organization ID
	// if the orgName parameter is empty the platform helm env home is returned
	ResolveHelmEnv(ctx context.Context, organizationID uint) (HelmEnv, error)

	ResolvePlatformEnv(ctx context.Context) (HelmEnv, error)
}

type helm2EnvResolver struct {
	// helmHomes the configurable directory location where helm homes are to be set up
	helmHomesDir string
	orgService   OrgService
	logger       Logger
}

func NewHelm2EnvResolver(helmHomesDir string, orgService OrgService, logger Logger) EnvResolver {
	return helm2EnvResolver{
		helmHomesDir: helmHomesDir,
		orgService:   orgService,
		logger:       logger,
	}
}

func (h2r helm2EnvResolver) ResolveHelmEnv(ctx context.Context, organizationID uint) (HelmEnv, error) {
	h2r.logger.Debug("resolving organization helm env home")
	orgName, err := h2r.orgService.GetOrgNameByOrgID(ctx, organizationID)
	if err != nil {
		return HelmEnv{}, errors.WrapIfWithDetails(err, "failed to get organization name for ID",
			"organizationID", organizationID)
	}

	return HelmEnv{
		home:     path.Join(h2r.helmHomesDir, orgName, helmPostFix),
		platform: false,
	}, nil
}

func (h2r helm2EnvResolver) ResolvePlatformEnv(ctx context.Context) (HelmEnv, error) {
	return HelmEnv{
		home:     path.Join(h2r.helmHomesDir, PlatformHelmHome, helmPostFix),
		platform: true,
	}, nil
}

// helm3EnvResolver helm env resolver to be used for resolving helm 3 environments
type helm3EnvResolver struct {
	delegate EnvResolver
}

func NewHelm3EnvResolver(delegate EnvResolver) EnvResolver {
	return helm3EnvResolver{delegate: delegate}
}

func (h3r helm3EnvResolver) ResolveHelmEnv(ctx context.Context, organizationID uint) (HelmEnv, error) {
	if organizationID == noOrg {
		// fallback to the platform / builtin helm env
		return h3r.ResolvePlatformEnv(ctx)
	}
	env, err := h3r.delegate.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return HelmEnv{}, errors.WrapIf(err, "failed to get helm env")
	}

	// add helm3 setup to the env
	return decorateEnv(env), nil
}

func (h3r helm3EnvResolver) ResolvePlatformEnv(ctx context.Context) (HelmEnv, error) {
	env, err := h3r.delegate.ResolvePlatformEnv(ctx)
	if err != nil {
		return HelmEnv{}, errors.WrapIf(err, "failed to get helm env")
	}

	return decorateEnv(env), nil
}

func decorateEnv(env HelmEnv) HelmEnv {
	newEnv := env
	newEnv.home = path.Join(env.home, "repository", "repositories.yaml")
	newEnv.repoCacheDir = path.Join(env.home, "repository", "cache")
	return newEnv
}

// EnvReconciler component interface for reconciling helm environments
type EnvReconciler interface {
	Reconcile(ctx context.Context, helmEnv HelmEnv) error
}

// Env reconciler for the builtin/platform helm env
// - creates the helm home for the platform
// - adds the configured default repositories
type builtinEnvReconciler struct {
	defaultRepos map[string]string
	envService   EnvService

	logger Logger
}

// NewBuiltinEnvReconciler creates a new platform helm env reconciler instance
func NewBuiltinEnvReconciler(builtinRepos map[string]string, envService EnvService, logger Logger) EnvReconciler {
	return builtinEnvReconciler{
		defaultRepos: builtinRepos,
		envService:   envService,

		logger: logger,
	}
}

// Reconcile adds the configured default repos to the platform helm env
func (b builtinEnvReconciler) Reconcile(ctx context.Context, helmEnv HelmEnv) error {
	for repoName, repoURL := range b.defaultRepos {
		if err := b.envService.AddRepository(ctx, helmEnv, Repository{Name: repoName, URL: repoURL}); err != nil {
			b.logger.Warn("failed to add builtin repository, it might already exist", map[string]interface{}{"repoName": repoName})
		}
	}

	b.logger.Info("platform helm environment set up, builtin repos added")
	return nil
}

// ensuringEnvResolver component that ensures the resolved environment is set up (on the filesystem)
// it decorates an existing envResolver with env service logic that checks and sets up the environement
type ensuringEnvResolver struct {
	// envresolver instance that gets decorated with the new functionality
	envResolver EnvResolver
	envService  EnvService
	logger      Logger
}

func NewEnsuringEnvResolver(envResolver EnvResolver, envService EnvService, logger Logger) EnvResolver {
	return ensuringEnvResolver{
		envResolver: envResolver,
		envService:  envService,
		logger:      logger,
	}
}

// ResolveHelmEnv resolves the helm environment for theh passed in organization; it also creates it if required
func (e ensuringEnvResolver) ResolveHelmEnv(ctx context.Context, organizationID uint) (HelmEnv, error) {
	helmEnv, err := e.envResolver.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return HelmEnv{}, errors.WrapIf(err, "failed to resolve helm env")
	}

	env, err := e.envService.EnsureEnv(ctx, helmEnv)
	if err != nil {
		return HelmEnv{}, errors.WrapIf(err, "failed to ensure helm environment")
	}
	e.logger.Debug("successfully resolved helm environment", map[string]interface{}{"orgID": organizationID, "helmEnv": helmEnv})
	return env, nil
}

func (e ensuringEnvResolver) ResolvePlatformEnv(ctx context.Context) (HelmEnv, error) {
	helmEnv, err := e.envResolver.ResolvePlatformEnv(ctx)
	if err != nil {
		return HelmEnv{}, errors.WrapIf(err, "failed to resolve platform helm env")
	}

	env, err := e.envService.EnsureEnv(ctx, helmEnv)
	if err != nil {
		return HelmEnv{}, errors.WrapIf(err, "failed to ensure platform helm environment")
	}

	e.logger.Debug("successfully resolved platform helm environment")
	return env, nil
}
