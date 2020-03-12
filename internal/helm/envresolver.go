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

const platformHelmHome = "pipeline"

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
}

func (e HelmEnv) GetHome() string {
	return e.home
}

func (e HelmEnv) IsPlatform() bool {
	return e.platform
}

// +testify:mock:testOnly=true

// HelmEnvResolver interface to abstract resolving helm homes
type EnvResolver interface {
	// ResolveHelmEnv resolves the helm home for the passed in organization ID
	// if the orgName parameter is empty the platform helm env home is returned
	ResolveHelmEnv(ctx context.Context, organizationID uint) (HelmEnv, error)

	ResolvePlatformEnv(ctx context.Context) (HelmEnv, error)
}

type helmEnvResolver struct {
	// helmHomes the configurable directory location where helm homes are to be set up
	helmHomes  string
	orgService OrgService
	logger     Logger
}

func NewHelmEnvResolver(helmHome string, orgService OrgService, logger Logger) EnvResolver {
	return helmEnvResolver{
		helmHomes:  helmHome,
		orgService: orgService,
		logger:     logger,
	}
}

func (h helmEnvResolver) ResolveHelmEnv(ctx context.Context, organizationID uint) (HelmEnv, error) {
	h.logger.Debug("resolving organization helm env home")
	orgName, err := h.orgService.GetOrgNameByOrgID(ctx, organizationID)
	if err != nil {
		return HelmEnv{}, errors.WrapIfWithDetails(err, "failed to get organization name for ID",
			"organizationID", organizationID)
	}

	return HelmEnv{
		home:     path.Join(h.helmHomes, orgName),
		platform: false,
	}, nil
}

func (h helmEnvResolver) ResolvePlatformEnv(ctx context.Context) (HelmEnv, error) {
	return HelmEnv{
		home:     path.Join(h.helmHomes, platformHelmHome),
		platform: true,
	}, nil
}
