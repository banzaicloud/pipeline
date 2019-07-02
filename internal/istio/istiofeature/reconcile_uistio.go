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
	"time"

	"github.com/ghodss/yaml"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/cluster"
	pConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/backoff"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
)

type monitoringConfig struct {
	hostname string
	url      string
}

func (m *MeshReconciler) ReconcileUistio(desiredState DesiredState) error {
	m.logger.Debug("reconciling Uistio")
	defer m.logger.Debug("Uistio reconciled")

	if desiredState == DesiredStatePresent {
		apiextclient, err := m.getApiExtensionK8sClient(m.Master)
		if err != nil {
			return emperror.Wrap(err, "could not get api extension client")
		}

		err = m.waitForMetricCRD("metrics.config.istio.io", apiextclient)
		if err != nil {
			return emperror.Wrap(err, "error while waiting for metric CRD")
		}

		k8sclient, err := m.getMasterK8sClient()
		if err != nil {
			return emperror.Wrap(err, "could not get k8s client")
		}
		err = m.waitForSidecarInjectorPod(k8sclient)
		if err != nil {
			return emperror.Wrap(err, "error while waiting for running sidecar injector")
		}

		err = m.installUistio(m.Master, monitoringConfig{
			hostname: prometheusHostname,
			url:      prometheusURL,
		}, m.logger)
		if err != nil {
			return emperror.Wrap(err, "could not install Uistio")
		}
	} else {
		err := m.uninstallUistio(m.Master, m.logger)
		if err != nil {
			return emperror.Wrap(err, "could not remove Uistio")
		}
	}

	return nil
}

// waitForSidecarInjectorPod waits for Sidecar Injector Pods to be running
func (m *MeshReconciler) waitForSidecarInjectorPod(client *kubernetes.Clientset) error {
	m.logger.Debug("waiting for sidecar injector pod")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(&backoffConfig)

	err := backoff.Retry(func() error {
		pods, err := client.CoreV1().Pods(istioOperatorNamespace).List(metav1.ListOptions{
			LabelSelector: "app=istio-sidecar-injector",
			FieldSelector: "status.phase=Running",
		})

		if err != nil {
			return emperror.Wrap(err, "could not list pods")
		}

		if len(pods.Items) == 0 {
			return errors.New("could not find any running sidecar injector")
		}

		return nil
	}, backoffPolicy)

	return err
}

// waitForMetricCRD waits for Metric CRD to be present in the cluster
func (m *MeshReconciler) waitForMetricCRD(name string, client *apiextensionsclient.Clientset) error {
	m.logger.WithField("name", name).Debug("waiting for metric CRD")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(&backoffConfig)

	err := backoff.Retry(func() error {
		_, err := client.ApiextensionsV1beta1().CustomResourceDefinitions().Get(name, metav1.GetOptions{})
		if err != nil {
			return emperror.Wrap(err, "could not get metric CRD")
		}

		return nil
	}, backoffPolicy)

	return err
}

// uninstallIstioOperator removes istio-operator from a cluster
func (m *MeshReconciler) uninstallUistio(c cluster.CommonCluster, logger logrus.FieldLogger) error {
	logger.Debug("removing Uistio")

	err := deleteDeployment(c, uistioReleaseName)
	if err != nil {
		return emperror.Wrap(err, "could not remove Uistio")
	}

	return nil
}

// installIstioOperator installs istio-operator on a cluster
func (m *MeshReconciler) installUistio(c cluster.CommonCluster, monitoring monitoringConfig, logger logrus.FieldLogger) error {
	logger.Debug("installing Uistio")

	values := map[string]interface{}{
		"affinity":    cluster.GetHeadNodeAffinity(c),
		"tolerations": cluster.GetHeadNodeTolerations(),
		"istio": map[string]interface{}{
			"CRName":    m.Configuration.name,
			"namespace": istioOperatorNamespace,
		},
		"prometheus": map[string]interface{}{
			"enabled": true,
			"host":    monitoring.hostname,
			"url":     monitoring.url,
		},
	}

	valuesOverride, err := yaml.Marshal(values)
	if err != nil {
		return emperror.Wrap(err, "could not marshal chart value overrides")
	}

	err = installOrUpgradeDeployment(
		c,
		uistioNamespace,
		pkgHelm.BanzaiRepository+"/"+viper.GetString(pConfig.UistioChartName),
		uistioReleaseName,
		valuesOverride,
		viper.GetString(pConfig.UistioChartVersion),
		true,
		true,
	)
	if err != nil {
		return emperror.Wrap(err, "could not install Uistio")
	}

	return nil
}
