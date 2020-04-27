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

	"github.com/banzaicloud/pipeline/internal/helm"

	"github.com/banzaicloud/pipeline/src/cluster"
)

func (m *MeshReconciler) ReconcileNodeExporter(desiredState DesiredState, c cluster.CommonCluster) error {
	m.logger.Debug("reconciling Node exporter")
	defer m.logger.Debug("Node exporter reconciled")

	if desiredState == DesiredStatePresent {
		return errors.WrapIf(m.installNodeExporter(c), "could not install Node exporter")
	}

	return errors.WrapIf(m.uninstallNodeExporter(c), "could not remove Node exporter")
}

// uninstallNodeExporter removes node exporter from a cluster
func (m *MeshReconciler) uninstallNodeExporter(c cluster.CommonCluster) error {
	m.logger.Debug("removing Node exporter")

	return errors.WrapIf(m.helmService.Delete(c, nodeExporterReleaseName, backyardsNamespace), "could not remove Node exporter")
}

// installNodeExporter installs node exporter on a cluster
func (m *MeshReconciler) installNodeExporter(c cluster.CommonCluster) error {
	m.logger.Debug("installing Node exporter")

	type Values struct {
		Service struct {
			Type        string            `json:"type,omitempty"`
			Port        int               `json:"port,omitempty"`
			TargetPort  int               `json:"targetPort,omitempty"`
			NodePort    int               `json:"nodePort,omitempty"`
			Annotations map[string]string `json:"annotations,omitempty"`
		} `json:"service,omitempty"`
	}

	values := Values{}
	values.Service.Port = 19100
	values.Service.TargetPort = 19100

	valuesOverride, err := ConvertStructure(values)
	if err != nil {
		return errors.WrapIf(err, "could not marshal chart value overrides")
	}

	nodeExporterConfig := m.Configuration.internalConfig.Charts.NodeExporter

	err = m.helmService.InstallOrUpgrade(
		c,
		helm.Release{
			ReleaseName: nodeExporterReleaseName,
			ChartName:   nodeExporterConfig.Chart,
			Namespace:   backyardsNamespace,
			Values:      valuesOverride,
			Version:     nodeExporterConfig.Version,
		},
		helm.Options{
			Namespace: backyardsNamespace,
			Wait:      true,
			Install:   true,
		},
	)

	return errors.WrapIf(err, "could not install Node exporter")
}
