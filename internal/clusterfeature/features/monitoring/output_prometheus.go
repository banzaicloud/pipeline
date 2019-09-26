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

type outputPrometheus struct {
	baseOutput
}

func newPrometheusOutputHelper(
	spec featureSpec,
) outputPrometheus {
	return outputPrometheus{
		baseOutput: baseOutput{
			ingress:  spec.Prometheus.Public,
			secretID: spec.Prometheus.SecretId,
			enabled:  spec.Prometheus.Enabled,
		},
	}
}

func (outputPrometheus) getOutputType() string {
	return "Prometheus"
}

func (outputPrometheus) getTopLevelDeploymentKey() string {
	return "prometheus"
}

func (outputPrometheus) getDeploymentValueParentKey() string {
	return "prometheusSpec"
}

func (outputPrometheus) getGeneratedSecretName(clusterID uint) string {
	return getPrometheusSecretName(clusterID)
}
