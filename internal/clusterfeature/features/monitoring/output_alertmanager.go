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

type outputAlertmanager struct {
	baseOutput
}

func newAlertmanagerOutputHelper(
	kubeConfig []byte,
	spec featureSpec,
) outputAlertmanager {
	return outputAlertmanager{
		baseOutput: baseOutput{
			ingress:              spec.Alertmanager.Ingress.baseIngressSpec,
			secretID:             spec.Alertmanager.Ingress.SecretId,
			enabled:              spec.Alertmanager.Enabled,
			k8sConfig:            kubeConfig,
		},
	}
}

func (outputAlertmanager) getOutputType() string {
	return "Alertmanager"
}

func (outputAlertmanager) getTopLevelDeploymentKey() string {
	return "alertmanager"
}

func (outputAlertmanager) getDeploymentValueParentKey() string {
	return "alertmanagerSpec"
}

func (outputAlertmanager) getGeneratedSecretName(clusterID uint) string {
	return getAlertmanagerSecretName(clusterID)
}

func (outputAlertmanager) getServiceName() string {
	return "monitor-prometheus-operato-alertmanager"
}
