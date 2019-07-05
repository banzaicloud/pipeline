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

package federation

import (
	"context"
	"strings"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	pConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/ghodss/yaml"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	apiv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/repo"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

type OperatorImage struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

func (m *FederationReconciler) ReconcileController(desiredState DesiredState) error {
	m.logger.Debug("start reconciling Federation controller")
	defer m.logger.Debug("finished reconciling Federation controller")

	if desiredState == DesiredStatePresent {
		err := m.installFederationController(m.Host, m.logger)
		if err != nil {
			return emperror.Wrap(err, "could not install Federation controller")
		}
	} else {

		err := m.removeFederationCRDs()
		if err != nil {
			return emperror.Wrap(err, "could not remove Federation CRD's")
		}

		err = m.deleteFederatedTypeConfigs()
		if err != nil {
			return emperror.Wrap(err, "could not remove Federation type configs")
		}

		err = m.uninstallFederationController(m.Host, m.logger)
		if err != nil {
			return emperror.Wrap(err, "could not remove Federation controller")
		}

	}

	return nil
}

func (m *FederationReconciler) deleteFederatedTypeConfigs() error {
	m.logger.Debug("start deleting Federation type configs")
	defer m.logger.Debug("finished deleting Federation type configs")

	client, err := m.getGenericClient()
	if err != nil {
		return err
	}

	list := &fedv1b1.FederatedTypeConfigList{}
	err = client.List(context.TODO(), list, m.Configuration.TargetNamespace)
	if err != nil {
		if strings.Contains(err.Error(), "no matches for kind") {
			m.logger.Warnf("no FederatedTypeConfig found")
		} else {
			return err
		}
	}

	for _, fedTypeConfig := range list.Items {
		err = client.Delete(context.TODO(), &fedTypeConfig, m.Configuration.TargetNamespace, fedTypeConfig.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *FederationReconciler) removeFederationCRDs() error {

	m.logger.Debug("start deleting Federation CRD's")
	defer m.logger.Debug("finished deleting Federation CRD's")

	clientConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return err
	}
	cl, err := v1beta1.NewForConfig(clientConfig)
	if err != nil {
		return err
	}
	crdList, err := cl.CustomResourceDefinitions().List(apiv1.ListOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "no matches for kind") {
			m.logger.Warnf("no CRD's found")
		} else {
			return err
		}
	}

	for _, crd := range crdList.Items {
		if strings.HasSuffix(crd.Name, federationCRDSuffix) {
			pp := apiv1.DeletePropagationBackground
			var secs int64
			secs = 180
			err = cl.CustomResourceDefinitions().Delete(crd.Name, &apiv1.DeleteOptions{
				PropagationPolicy:  &pp,
				GracePeriodSeconds: &secs,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// uninstallFederationController removes Federation controller from a cluster
func (m *FederationReconciler) uninstallFederationController(c cluster.CommonCluster, logger logrus.FieldLogger) error {
	logger.Debug("removing Federation controller")

	err := DeleteDeployment(c, federationReleaseName)
	if err != nil {
		return emperror.Wrap(err, "could not remove Federation controller")
	}

	return nil
}

// installFederationController installs Federation controller on a cluster
func (m *FederationReconciler) installFederationController(c cluster.CommonCluster, logger logrus.FieldLogger) error {
	logger.Debug("installing Federation controller")
	scope := apiextv1b1.ClusterScoped
	if !m.Configuration.GlobalScope {
		scope = apiextv1b1.NamespaceScoped
	}
	schedulerPreferences := "Enabled"
	if !m.Configuration.SchedulerPreferences {
		schedulerPreferences = "Disabled"
	}
	crossClusterServiceDiscovery := "Enabled"
	if !m.Configuration.CrossClusterServiceDiscovery {
		crossClusterServiceDiscovery = "Disabled"
	}
	federatedIngress := "Enabled"
	if !m.Configuration.FederatedIngress {
		federatedIngress = "Disabled"
	}

	values := map[string]interface{}{
		"global": map[string]interface{}{
			"scope": scope,
		},
		"controllermanager": map[string]interface{}{
			"featureGates": map[string]interface{}{
				"SchedulerPreferences":         schedulerPreferences,
				"CrossClusterServiceDiscovery": crossClusterServiceDiscovery,
				"FederatedIngress":             federatedIngress,
			},
		},
	}

	valuesOverride, err := yaml.Marshal(values)
	if err != nil {
		return emperror.Wrap(err, "could not marshal chart value overrides")
	}

	//ensure repo
	org, err := auth.GetOrganizationById(c.GetOrganizationId())
	if err != nil {
		return emperror.Wrap(err, "could not get organization")
	}

	env := helm.GenerateHelmRepoEnv(org.Name)
	_, err = helm.ReposAdd(env, &repo.Entry{
		Name: "kubefed-charts",
		URL:  "https://raw.githubusercontent.com/banzaicloud/kubefed/master/charts",
	})
	if err != nil {
		return emperror.WrapWith(err, "failed to add kube-chart repo")
	}

	err = InstallOrUpgradeDeployment(
		c,
		m.Configuration.TargetNamespace,
		//pkgHelm.BanzaiRepository+"/"+
		viper.GetString(pConfig.FederationChartName),
		federationReleaseName,
		valuesOverride,
		viper.GetString(pConfig.FederationChartVersion),
		true,
		true,
	)
	if err != nil {
		return emperror.Wrap(err, "could not install Federation controller")
	}

	return nil
}
