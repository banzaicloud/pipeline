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

package istiofeature

import (
	"emperror.dev/errors"
	"github.com/ghodss/yaml"

	"github.com/banzaicloud/pipeline/src/cluster"
)

func (m *MeshReconciler) ReconcileCanaryOperator(desiredState DesiredState, c cluster.CommonCluster) error {
	m.logger.Debug("reconciling canary-operator")
	defer m.logger.Debug("canary-operator reconciled")

	if desiredState == DesiredStatePresent {
		k8sclient, err := m.getRuntimeK8sClient(c)
		if err != nil {
			return errors.WrapIf(err, "could not get k8s client")
		}
		err = m.waitForPod(k8sclient, istioOperatorNamespace, map[string]string{"app": "istiod"}, "")
		if err != nil {
			return errors.WrapIf(err, "error while waiting for running sidecar injector")
		}

		return errors.WrapIf(m.installCanaryOperator(c, prometheusURL), "could not install canary-operator")
	}

	return errors.WrapIf(m.uninstallCanaryOperator(c), "could not remove canary-operator")
}

// uninstallCanaryOperator removes canary-operator from a cluster
func (m *MeshReconciler) uninstallCanaryOperator(c cluster.CommonCluster) error {
	m.logger.Debug("removing istio release operator")

	return errors.WrapIf(deleteDeployment(c, canaryOperatorReleaseName), "could not remove canary-operator")
}

// installCanaryOperator installs canary-operator to a cluster
func (m *MeshReconciler) installCanaryOperator(c cluster.CommonCluster, prometheusURL string) error {
	m.logger.Debug("installing canary-operator")

	type operator struct {
		Image      imageChartValue      `json:"image,omitempty"`
		Prometheus prometheusChartValue `json:"prometheus,omitempty"`
	}

	type Values struct {
		Operator operator `json:"operator,omitempty"`
	}

	values := Values{
		Operator: operator{
			Image: imageChartValue{},
			Prometheus: prometheusChartValue{
				URL: prometheusURL,
			},
		},
	}

	canaryChart := m.Configuration.internalConfig.Charts.CanaryOperator

	if canaryChart.Values.Operator.Image.Repository != "" {
		values.Operator.Image.Repository = canaryChart.Values.Operator.Image.Repository
	}
	if canaryChart.Values.Operator.Image.Tag != "" {
		values.Operator.Image.Tag = canaryChart.Values.Operator.Image.Tag
	}

	valuesOverride, err := yaml.Marshal(values)
	if err != nil {
		return errors.WrapIf(err, "could not marshal chart value overrides")
	}

	err = installOrUpgradeDeployment(
		c,
		canaryOperatorNamespace,
		canaryChart.Chart,
		canaryOperatorReleaseName,
		valuesOverride,
		canaryChart.Version,
		true,
		true,
	)

	return errors.WrapIf(err, "could not install canary-operator")
}
