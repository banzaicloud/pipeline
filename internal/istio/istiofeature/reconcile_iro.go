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
	"github.com/ghodss/yaml"
	"github.com/goph/emperror"

	"github.com/banzaicloud/pipeline/cluster"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	corev1 "k8s.io/api/core/v1"
)

func (m *MeshReconciler) ReconcileIRO(desiredState DesiredState) error {
	m.logger.Debug("reconciling istio-release-operator")
	defer m.logger.Debug("istio-release-operator reconciled")

	if desiredState == DesiredStatePresent {
		k8sclient, err := m.getMasterK8sClient()
		if err != nil {
			return emperror.Wrap(err, "could not get k8s client")
		}
		err = m.waitForSidecarInjectorPod(k8sclient)
		if err != nil {
			return emperror.Wrap(err, "error while waiting for running sidecar injector")
		}

		err = m.installIRO(m.Master, prometheusURL)
		if err != nil {
			return emperror.Wrap(err, "could not install istio-release-operator")
		}
	} else {
		err := m.uninstallIRO(m.Master)
		if err != nil {
			return emperror.Wrap(err, "could not remove istio-release-operator")
		}
	}

	return nil
}

// uninstallIRO removes istio-release-operator from a cluster
func (m *MeshReconciler) uninstallIRO(c cluster.CommonCluster) error {
	m.logger.Debug("removing istio release operator")

	err := deleteDeployment(c, iroReleaseName)
	if err != nil {
		return emperror.Wrap(err, "could not remove istio-release-operator")
	}

	return nil
}

// installIRO installs istio-release-operator to a cluster
func (m *MeshReconciler) installIRO(c cluster.CommonCluster, prometheusURL string) error {
	m.logger.Debug("installing istio-release-operator")

	type operator struct {
		Image      imageChartValue      `json:"image,omitempty"`
		Prometheus prometheusChartValue `json:"prometheus,omitempty"`
	}

	type Values struct {
		Affinity    corev1.Affinity     `json:"affinity,omitempty"`
		Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
		Operator    operator            `json:"operator,omitempty"`
	}

	values := Values{
		Affinity:    cluster.GetHeadNodeAffinity(c),
		Tolerations: cluster.GetHeadNodeTolerations(),
		Operator: operator{
			Image: imageChartValue{},
			Prometheus: prometheusChartValue{
				URL: prometheusURL,
			},
		},
	}

	if m.Configuration.internalConfig.iro.imageRepository != "" {
		values.Operator.Image.Repository = m.Configuration.internalConfig.iro.imageRepository
	}
	if m.Configuration.internalConfig.iro.imageTag != "" {
		values.Operator.Image.Tag = m.Configuration.internalConfig.iro.imageTag
	}

	valuesOverride, err := yaml.Marshal(values)
	if err != nil {
		return emperror.Wrap(err, "could not marshal chart value overrides")
	}

	err = installOrUpgradeDeployment(
		c,
		meshNamespace,
		pkgHelm.BanzaiRepository+"/"+m.Configuration.internalConfig.iro.chartName,
		iroReleaseName,
		valuesOverride,
		m.Configuration.internalConfig.iro.chartVersion,
		true,
		true,
	)
	if err != nil {
		return emperror.Wrap(err, "could not install istio-release-operator")
	}

	return nil
}
