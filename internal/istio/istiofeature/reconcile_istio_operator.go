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

const istioOperatorNamespace = "istio-system"
const istioOperatorReleaseName = "istio-operator"
const istioVersion = "1.1"

type OperatorImage struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

func (m *MeshReconciler) ReconcileIstioOperator(desiredState DesiredState) error {
	m.logger.Debug("reconciling Istio operator")
	defer m.logger.Debug("Istio operator reconciled")

	if desiredState == DesiredStatePresent {
		err := m.installIstioOperator(m.Master, m.logger)
		if err != nil {
			return emperror.Wrap(err, "could not install Istio operator")
		}
	} else {
		err := m.uninstallIstioOperator(m.Master, m.logger)
		if err != nil {
			return emperror.Wrap(err, "could not remove Istio operator")
		}
	}

	return nil
}

// uninstallIstioOperator removes istio-operator from a cluster
func (m *MeshReconciler) uninstallIstioOperator(c cluster.CommonCluster, logger logrus.FieldLogger) error {
	logger.Debug("removing Istio operator")

	err := deleteDeployment(c, istioOperatorReleaseName)
	if err != nil {
		return emperror.Wrap(err, "could not remove Istio operator")
	}

	return nil
}

// installIstioOperator installs istio-operator on a cluster
func (m *MeshReconciler) installIstioOperator(c cluster.CommonCluster, logger logrus.FieldLogger) error {
	logger.Debug("installing Istio operator")

	image := &OperatorImage{}
	values := map[string]interface{}{
		"affinity":    cluster.GetHeadNodeAffinity(c),
		"tolerations": cluster.GetHeadNodeTolerations(),
		"operator": map[string]*OperatorImage{
			"image": image,
		},
	}

	imageRepository := viper.GetString(pConfig.IstioOperatorImageRepository)
	if imageRepository != "" {
		image.Repository = imageRepository
	}
	imageTag := viper.GetString(pConfig.IstioOperatorImageTag)
	if imageTag != "" {
		image.Tag = imageTag
	}

	valuesOverride, err := yaml.Marshal(values)
	if err != nil {
		return emperror.Wrap(err, "could not marshal chart value overrides")
	}

	err = installDeployment(
		c,
		istioOperatorNamespace,
		pkgHelm.BanzaiRepository+"/"+viper.GetString(pConfig.IstioOperatorChartName),
		istioOperatorReleaseName,
		valuesOverride,
		viper.GetString(pConfig.IstioOperatorChartVersion),
		true,
		m.logger,
	)
	if err != nil {
		return emperror.Wrap(err, "could not install Istio operator")
	}

	return nil
}
