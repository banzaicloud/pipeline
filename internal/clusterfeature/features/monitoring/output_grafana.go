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

package monitoring

type outputGrafana struct {
	baseOutput
}

func newGrafanaOutputHelper(
	k8sConfig []byte,
	spec featureSpec,
) outputGrafana {
	return outputGrafana{
		baseOutput: baseOutput{
			ingress:   spec.Grafana.Ingress,
			secretID:  spec.Grafana.SecretId,
			enabled:   spec.Grafana.Enabled,
			k8sConfig: k8sConfig,
		},
	}
}

func (outputGrafana) getOutputType() string {
	return "Grafana"
}

func (outputGrafana) getTopLevelDeploymentKey() string {
	return ""
}

func (outputGrafana) getDeploymentValueParentKey() string {
	return "grafana"
}

func (outputGrafana) getGeneratedSecretName(clusterID uint) string {
	return getGrafanaSecretName(clusterID)
}

func (outputGrafana) getServiceName() string {
	return "monitor-grafana"
}
