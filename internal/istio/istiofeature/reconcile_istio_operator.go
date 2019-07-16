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
	corev1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/cluster"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
)

func (m *MeshReconciler) ReconcileIstioOperator(desiredState DesiredState) error {
	m.logger.Debug("reconciling Istio operator")
	defer m.logger.Debug("Istio operator reconciled")

	if desiredState == DesiredStatePresent {
		err := m.installIstioOperator(m.Master)
		if err != nil {
			return emperror.Wrap(err, "could not install Istio operator")
		}
	} else {
		err := m.uninstallIstioOperator(m.Master)
		if err != nil {
			return emperror.Wrap(err, "could not remove Istio operator")
		}
	}

	return nil
}

// uninstallIstioOperator removes istio-operator from a cluster
func (m *MeshReconciler) uninstallIstioOperator(c cluster.CommonCluster) error {
	m.logger.Debug("removing Istio operator")

	err := deleteDeployment(c, istioOperatorReleaseName)
	if err != nil {
		return emperror.Wrap(err, "could not remove Istio operator")
	}

	return nil
}

// installIstioOperator installs istio-operator on a cluster
func (m *MeshReconciler) installIstioOperator(c cluster.CommonCluster) error {
	m.logger.Debug("installing Istio operator")

	type operator struct {
		Image imageChartValue `json:"image,omitempty"`
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
		},
	}

	if m.Configuration.internalConfig.istioOperator.imageRepository != "" {
		values.Operator.Image.Repository = m.Configuration.internalConfig.istioOperator.imageRepository
	}
	if m.Configuration.internalConfig.istioOperator.imageTag != "" {
		values.Operator.Image.Tag = m.Configuration.internalConfig.istioOperator.imageTag
	}

	valuesOverride, err := yaml.Marshal(values)
	if err != nil {
		return emperror.Wrap(err, "could not marshal chart value overrides")
	}

	err = installOrUpgradeDeployment(
		c,
		istioOperatorNamespace,
		pkgHelm.BanzaiRepository+"/"+m.Configuration.internalConfig.istioOperator.chartName,
		istioOperatorReleaseName,
		valuesOverride,
		m.Configuration.internalConfig.istioOperator.chartVersion,
		true,
		true,
	)
	if err != nil {
		return emperror.Wrap(err, "could not install Istio operator")
	}

	return nil
}
