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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/banzaicloud/pipeline/internal/helm"

	"github.com/banzaicloud/pipeline/src/cluster"
)

func (m *MeshReconciler) ReconcileIstioOperator(desiredState DesiredState, c cluster.CommonCluster) error {
	m.logger.Debug("reconciling Istio operator")
	defer m.logger.Debug("Istio operator reconciled")

	if desiredState == DesiredStatePresent {
		return errors.WrapIf(m.installIstioOperator(c), "could not install Istio operator")
	}

	return errors.WrapIf(m.uninstallIstioOperator(c), "could not remove Istio operator")
}

// uninstallIstioOperator removes istio-operator from a cluster
func (m *MeshReconciler) uninstallIstioOperator(c cluster.CommonCluster) error {
	m.logger.Debug("removing Istio operator")

	return errors.WrapIf(m.helmService.Delete(c, istioOperatorReleaseName, istioOperatorNamespace), "could not remove Istio operator")
}

// installIstioOperator installs istio-operator on a cluster
func (m *MeshReconciler) installIstioOperator(c cluster.CommonCluster) error {
	m.logger.Debug("installing Istio operator")

	type operator struct {
		Image     imageChartValue             `json:"image,omitempty"`
		Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	}

	type Values struct {
		OperatorComponentSuffix string   `json:"operatorComponentSuffix"`
		Operator                operator `json:"operator,omitempty"`
	}

	values := Values{
		OperatorComponentSuffix: "-operator",
		Operator: operator{
			Image: imageChartValue{},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
			},
		},
	}

	istioChart := m.Configuration.internalConfig.Charts.IstioOperator

	if istioChart.Values.Operator.Image.Repository != "" {
		values.Operator.Image.Repository = istioChart.Values.Operator.Image.Repository
	}
	if istioChart.Values.Operator.Image.Tag != "" {
		values.Operator.Image.Tag = istioChart.Values.Operator.Image.Tag
	}

	valuesOverride, err := ConvertStructure(values)
	if err != nil {
		return errors.WrapIf(err, "could not marshal chart value overrides")
	}

	err = m.helmService.InstallOrUpgrade(c,
		helm.Release{
			ReleaseName: istioOperatorReleaseName,
			ChartName:   istioChart.Chart,
			Namespace:   istioOperatorNamespace,
			Values:      valuesOverride,
			Version:     istioChart.Version,
		},
		helm.Options{
			Namespace: istioOperatorNamespace,
			Wait:      true,
			Install:   true,
		},
	)

	return errors.WrapIf(err, "could not install Istio operator")
}
