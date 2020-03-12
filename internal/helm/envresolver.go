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
)

const platformHelmHome = "pipeline"

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

// HelmEnvResolver interface to abstract resolving helm homes
type EnvResolver interface {
	// ResolveHelmEnv resolves the helm home for the passed in organization name
	// if the orgName parameter is empty the platfrom helm env home is returned
	ResolveHelmEnv(ctx context.Context, orgName string) (HelmEnv, error)
}

type helmEnvResolver struct {
	// helmHomes the configurable directory location where helm homes are to be set up
	helmHomes string
	logger    Logger
}

func (h helmEnvResolver) ResolveHelmEnv(ctx context.Context, orgName string) (HelmEnv, error) {
	if orgName == "" {
		h.logger.Debug("resolving platform helm env home")

		return HelmEnv{
			home:     path.Join(h.helmHomes, platformHelmHome),
			platform: true,
		}, nil
	}

	h.logger.Debug("resolving organization helm env home")
	return HelmEnv{
		home:     path.Join(h.helmHomes, orgName),
		platform: false,
	}, nil
}

func NewHelmEnvResolver(helmHome string, logger Logger) EnvResolver {
	return helmEnvResolver{
		helmHomes: helmHome,
		logger:    logger,
	}
}
