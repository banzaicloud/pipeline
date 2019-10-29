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

type outputPushgateway struct {
	baseOutput
}

func newPushgatewayOutputHelper(
	kubeConfig []byte,
	spec featureSpec,
) outputPushgateway {
	return outputPushgateway{
		baseOutput: baseOutput{
			ingress:   spec.Pushgateway.Ingress.baseIngressSpec,
			secretID:  spec.Pushgateway.Ingress.SecretId,
			enabled:   spec.Pushgateway.Enabled,
			k8sConfig: kubeConfig,
		},
	}
}

func (outputPushgateway) getOutputType() string {
	return "Pushgateway"
}

func (outputPushgateway) getTopLevelDeploymentKey() string {
	return ""
}

func (outputPushgateway) getDeploymentValueParentKey() string {
	return ""
}

func (outputPushgateway) getGeneratedSecretName(clusterID uint) string {
	return getPushgatewaySecretName(clusterID)
}

func (outputPushgateway) getServiceName() string {
	return "pushgateway-prometheus-pushgateway"
}
