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
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/cluster"
	pConfig "github.com/banzaicloud/pipeline/config"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
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

		err = m.installIRO(m.Master, prometheusURL, m.logger)
		if err != nil {
			return emperror.Wrap(err, "could not install istio-release-operator")
		}
	} else {
		err := m.uninstallIRO(m.Master, m.logger)
		if err != nil {
			return emperror.Wrap(err, "could not remove istio-release-operator")
		}
	}

	return nil
}

// uninstallIRO removes istio-release-operator from a cluster
func (m *MeshReconciler) uninstallIRO(c cluster.CommonCluster, logger logrus.FieldLogger) error {
	logger.Debug("removing istio release operator")

	err := deleteDeployment(c, iroReleaseName)
	if err != nil {
		return emperror.Wrap(err, "could not remove istio-release-operator")
	}

	return nil
}

// installIRO installs istio-release-operator to a cluster
func (m *MeshReconciler) installIRO(c cluster.CommonCluster, prometheusURL string, logger logrus.FieldLogger) error {
	logger.Debug("installing istio-release-operator")

	image := &OperatorImage{}

	imageRepository := viper.GetString(pConfig.IROImageRepository)
	if imageRepository != "" {
		image.Repository = imageRepository
	}
	imageTag := viper.GetString(pConfig.IROImageTag)
	if imageTag != "" {
		image.Tag = imageTag
	}

	values := map[string]interface{}{
		"affinity":    cluster.GetHeadNodeAffinity(c),
		"tolerations": cluster.GetHeadNodeTolerations(),
		"operator": map[string]interface{}{
			"image": image,
			"prometheus": map[string]string{
				"url": prometheusURL,
			},
		},
	}

	valuesOverride, err := yaml.Marshal(values)
	if err != nil {
		return emperror.Wrap(err, "could not marshal chart value overrides")
	}

	err = installOrUpgradeDeployment(
		c,
		meshNamespace,
		pkgHelm.BanzaiRepository+"/"+viper.GetString(pConfig.IROChartName),
		iroReleaseName,
		valuesOverride,
		viper.GetString(pConfig.IROChartVersion),
		true,
		true,
	)
	if err != nil {
		return emperror.Wrap(err, "could not install istio-release-operator")
	}

	return nil
}
