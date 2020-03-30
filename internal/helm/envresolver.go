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
)

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
	helmHomes  string
	orgService OrgService
	logger     Logger
}

func NewHelm2EnvResolver(helmHome string, orgService OrgService, logger Logger) EnvResolver {
	return helm2EnvResolver{
		helmHomes:  helmHome,
		orgService: orgService,
		logger:     logger,
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
		home:     path.Join(h2r.helmHomes, orgName, helmPostFix),
		platform: false,
	}, nil
}

func (h2r helm2EnvResolver) ResolvePlatformEnv(ctx context.Context) (HelmEnv, error) {
	return HelmEnv{
		home:     path.Join(h2r.helmHomes, PlatformHelmHome, helmPostFix),
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
	env, err := h3r.delegate.ResolveHelmEnv(ctx, organizationID)
	if err != nil {
		return HelmEnv{}, errors.WrapIf(err, "failed to get helm env")
	}
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
