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

//  TODO is this exaggerate?
type KubeConfigBytes = []byte

// utility for providing input arguments ...
func (ri Release) NameAndChartSlice() []string {
	if ri.ReleaseName == "" {
		return []string{ri.ChartName}
	}
	return []string{ri.ReleaseName, ri.ChartName}
}

// ReleaserOptions placeholder for releaser directives
type ReleaserOptions struct {
	DryRun       bool
	GenerateName bool
	Wait         bool
	Namespace    string
}

// Releaser interface collecting operations related to releases
type Releaser interface {
	// Install installs the specified chart using to a cluster identified by the kubeConfig  argument
	Install(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput Release, options ReleaserOptions) (string, error)
	// Uninstall removes the  specified release from the cluster
	Uninstall(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput Release, options ReleaserOptions) error
	// Lists releases
	List(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, options ReleaserOptions) ([]Release, error)
}
