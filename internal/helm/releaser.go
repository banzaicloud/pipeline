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
)

type KubeConfigBytes = []byte

// utility for providing input arguments ...
func (ri Release) NameAndChartSlice() []string {
	if ri.ReleaseName == "" {
		return []string{ri.ChartName}
	}
	return []string{ri.ReleaseName, ri.ChartName}
}

// Releaser interface collecting operations related to releases
type Releaser interface {
	// Install installs the specified chart using to a cluster identified by the kubeConfig  argument
	Install(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput Release, options Options) (string, error)
	// Uninstall removes the  specified release from the cluster
	Uninstall(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseName string, options Options) error
	// List lists releases
	List(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, options Options) ([]Release, error)
	// Get gets the given release details
	Get(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput Release, options Options) (Release, error)
	// Upgrade upgrades the given release
	Upgrade(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput Release, options Options) (string, error)
	// Resources retrieves the kubernetes resources belonging to the release
	Resources(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput Release, options Options) ([]ReleaseResource, error)
}
