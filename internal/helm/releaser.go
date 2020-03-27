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

// ReleaseInput struct encapsulating information about the release to be created
type ReleaseInput struct {
	ReleaseName string
	ChartName   string
	Namespace   string
	Values      []string // TODO is this type OK?
	// TODO repo here?
}

// utility for providing input arguments ...
func (ri ReleaseInput) NameAndChartSlice() []string {
	if ri.ReleaseName == "" {
		return []string{ri.ChartName}
	}
	return []string{ri.ReleaseName, ri.ChartName}
}

// ReleaserOptions placeholder for releaser directives
type ReleaserOptions struct {
}

// Releaser interface collecting operations related to releases
type Releaser interface {
	// Install installs the specified chart using to a cluster identified by the kubeConfig  argument
	Install(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput ReleaseInput, options ReleaserOptions) (string, error)
}
