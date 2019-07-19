// Copyright © 2019 Banzai Cloud
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

package clusterfeature

import (
	"context"
)

// HelmService interface for helm operations
type HelmService interface {
	// InstallDeployment installs a feature to the given cluster
	InstallDeployment(ctx context.Context, orgName string, kubeConfig []byte, namespace string,
		deploymentName string, releaseName string, values []byte, chartVersion string, wait bool) error

	// DeleteDeployment removes a feature to the given cluster
	DeleteDeployment(ctx context.Context, kubeConfig []byte, releaseName string) error

	UpdateDeployment(ctx context.Context, orgName string, kubeConfig []byte, namespace string,
		deploymentName string, releaseName string, values []byte, chartVersion string) error
}
