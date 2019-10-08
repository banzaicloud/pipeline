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

	"emperror.dev/emperror"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/repo"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	ctlutil "sigs.k8s.io/kubefed/pkg/controller/util"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	pConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
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
		err := m.uninstallFederationController(m.Host, m.logger)
		if err != nil {
			return emperror.Wrap(err, "could not remove Federation controller")
		}

	}

	return nil
}

func (m *FederationReconciler) ReconcileServiceDiscovery(desiredState DesiredState) error {
	if desiredState == DesiredStatePresent {
		return nil
	}

	err := m.deleteFederatedResources(m.ingressDNSRecordResource)
	if err != nil {
		return emperror.Wrap(err, "could not remove ingressDNSRecord(s)")
	}

	err = m.deleteFederatedResources(m.serviceDNSRecordResource)
	if err != nil {
		return emperror.Wrap(err, "could not remove serviceDNSRecord(s)")
	}

	return nil
}

func (m *FederationReconciler) ReconcileFederatedTypes(desiredState DesiredState) error {
	if desiredState == DesiredStatePresent {
		return nil
	}

	m.logger.Debug("start removing federated resources and FederatedTypes")
	defer m.logger.Debug("finished removing federated resources and FederatedTypes")

	err := m.deleteFederatedResourcesAndTypeConfigs()
	if err != nil {
		return emperror.Wrap(err, "could not remove Federation resources and typeConfigs")
	}

	err = m.removeFederationCRDs(true)
	if err != nil {
		return emperror.Wrap(err, "could not remove Federation CRD's")
	}

	return nil
}

func (m *FederationReconciler) deleteFederatedResourcesAndTypeConfigs() error {
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
		apiResource := fedTypeConfig.GetFederatedType()
		err := m.deleteFederatedResources(&apiResource)
		if err != nil {
			return err
		}
	}

	for _, fedTypeConfig := range list.Items {
		m.logger.Debugf("delete fedTypeConfig %s", fedTypeConfig.Name)
		err = client.Delete(context.TODO(), &fedTypeConfig, m.Configuration.TargetNamespace, fedTypeConfig.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *FederationReconciler) federatedResourcesExists(resource *metav1.APIResource) (bool, error) {
	clientConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return false, err
	}

	client, err := ctlutil.NewResourceClient(clientConfig, resource)
	if err != nil {
		return false, err
	}

	list, err := client.Resources("").List(metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			m.logger.Debugf("no %s found", resource.Name)
		} else {
			return false, err
		}
	}
	if list != nil && len(list.Items) > 0 {
		return true, nil
	}

	return false, nil
}

func (m *FederationReconciler) deleteFederatedResources(resource *metav1.APIResource) error {

	m.logger.Debugf("start deleting resource %s", resource.Name)
	defer m.logger.Debugf("finished deleting resource %s", resource.Name)

	clientConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return err
	}

	client, err := ctlutil.NewResourceClient(clientConfig, resource)
	if err != nil {
		return err
	}

	list, err := client.Resources("").List(metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			m.logger.Debugf("no %s found", resource.Name)
		} else {
			return err
		}
	}
	if list == nil || len(list.Items) == 0 {
		m.logger.Debugf("no %s found", resource.Name)
	} else {
		for _, fn := range list.Items {
			m.logger.Debugf("delete %s %s", fn.GetName(), fn.GetKind())
			err = client.Resources(fn.GetNamespace()).Delete(fn.GetName(), &metav1.DeleteOptions{}, "")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *FederationReconciler) removeFederationCRDs(all bool) error {

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
		if strings.HasSuffix(crd.Name, federationCRDSuffix) &&
			(strings.HasPrefix(crd.Name, "federated") || all) {
			pp := apiv1.DeletePropagationBackground
			var secs int64
			secs = 180
			m.logger.Debugf("removing CRD %s", crd.Name)
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

	fedImageTag := viper.GetString(pConfig.FederationImageTag)
	fedImageRepo := viper.GetString(pConfig.FederationImageRepo)
	values := map[string]interface{}{
		"global": map[string]interface{}{
			"scope": scope,
		},
		"controllermanager": map[string]interface{}{
			"repository": fedImageRepo,
			"tag":        fedImageTag,
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

	// ensure repo
	org, err := auth.GetOrganizationById(c.GetOrganizationId())
	if err != nil {
		return emperror.Wrap(err, "could not get organization")
	}

	env := helm.GenerateHelmRepoEnv(org.Name)
	_, err = helm.AddRepo(env, &repo.Entry{
		Name: "kubefed-charts",
		URL:  "https://raw.githubusercontent.com/banzaicloud/kubefed/helm_chart/charts",
	})
	if err != nil {
		return emperror.WrapWith(err, "failed to add kube-chart repo")
	}

	err = InstallOrUpgradeDeployment(
		c,
		m.Configuration.TargetNamespace,
		// pkgHelm.BanzaiRepository+"/"+
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
