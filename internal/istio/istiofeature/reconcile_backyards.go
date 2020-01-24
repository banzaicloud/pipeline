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
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/pkg/backoff"
	"github.com/banzaicloud/pipeline/src/cluster"
)

type monitoringConfig struct {
	hostname string
	url      string
}

func (m *MeshReconciler) ReconcileBackyards(desiredState DesiredState) error {
	m.logger.Debug("reconciling Backyards")
	defer m.logger.Debug("Backyards reconciled")

	if desiredState == DesiredStatePresent {
		client, err := m.getRuntimeK8sClient(m.Master)
		if err != nil {
			return errors.WrapIf(err, "could not get api extension client")
		}

		err = m.waitForCRD("instances.config.istio.io", client)
		if err != nil {
			return errors.WrapIf(err, "error while waiting for metric CRD")
		}

		k8sclient, err := m.getMasterK8sClient()
		if err != nil {
			return errors.WrapIf(err, "could not get k8s client")
		}
		err = m.waitForSidecarInjectorPod(k8sclient)
		if err != nil {
			return errors.WrapIf(err, "error while waiting for running sidecar injector")
		}

		err = m.installBackyards(m.Master, monitoringConfig{
			hostname: prometheusHostname,
			url:      prometheusURL,
		})
		if err != nil {
			return errors.WrapIf(err, "could not install Backyards")
		}
	} else {
		err := m.uninstallBackyards(m.Master)
		if err != nil {
			return errors.WrapIf(err, "could not remove Backyards")
		}
	}

	return nil
}

// waitForSidecarInjectorPod waits for Sidecar Injector Pods to be running
func (m *MeshReconciler) waitForSidecarInjectorPod(client runtimeclient.Client) error {
	m.logger.Debug("waiting for sidecar injector pod")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	err := backoff.Retry(func() error {
		var pods corev1.PodList
		err := client.List(context.Background(), &pods, runtimeclient.MatchingLabels(map[string]string{"app": "istio-sidecar-injector"}), runtimeclient.MatchingFields(fields.Set(map[string]string{"status.phase": "Running"})))
		if err != nil {
			return errors.WrapIf(err, "could not list pods")
		}

		if len(pods.Items) == 0 {
			return errors.New("could not find any running sidecar injector")
		}

		return nil
	}, backoffPolicy)

	return err
}

// waitForCRD waits for CRD to be present in the cluster
func (m *MeshReconciler) waitForCRD(name string, client runtimeclient.Client) error {
	m.logger.WithField("name", name).Debug("waiting for CRD")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	var crd apiextensionsv1beta1.CustomResourceDefinition
	err := backoff.Retry(func() error {
		err := client.Get(context.Background(), types.NamespacedName{
			Name: name,
		}, &crd)
		if err != nil {
			return errors.WrapIf(err, "could not get CRD")
		}

		return nil
	}, backoffPolicy)

	return err
}

// uninstallIstioOperator removes istio-operator from a cluster
func (m *MeshReconciler) uninstallBackyards(c cluster.CommonCluster) error {
	m.logger.Debug("removing Backyards")

	err := deleteDeployment(c, backyardsReleaseName)
	if err != nil {
		return errors.WrapIf(err, "could not remove Backyards")
	}

	return nil
}

// installIstioOperator installs istio-operator on a cluster
func (m *MeshReconciler) installBackyards(c cluster.CommonCluster, monitoring monitoringConfig) error {
	m.logger.Debug("installing Backyards")

	type istio struct {
		CRname    string `json:"CRName,omitempty"`
		Namespace string `json:"namespace,omitempty"`
	}

	type application struct {
		Image imageChartValue   `json:"image,omitempty"`
		Env   map[string]string `json:"env,omitempty"`
	}

	type Values struct {
		Istio       istio                `json:"istio,omitempty"`
		Application application          `json:"application,omitempty"`
		Prometheus  prometheusChartValue `json:"prometheus,omitempty"`
		Ingress     struct {
			Enabled bool `json:"enabled"`
		} `json:"ingress,omitempty"`
		Web struct {
			Enabled bool              `json:"enabled"`
			Image   imageChartValue   `json:"image,omitempty"`
			Env     map[string]string `json:"env,omitempty"`
		} `json:"web,omitempty"`
		Autoscaling struct {
			Enabled bool `json:"enabled"`
		} `json:"autoscaling,omitempty"`
		Grafana struct {
			Enabled  bool `json:"enabled"`
			Security struct {
				Enabled bool `json:"enabled"`
			} `json:"security"`
			ExternalURL string `json:"externalUrl"`
		} `json:"grafana"`
	}

	values := Values{
		Application: application{
			Image: imageChartValue{},
			Env: map[string]string{
				"APP_CANARYENABLED": "true",
			},
		},
		Istio: istio{
			CRname:    m.Configuration.name,
			Namespace: istioOperatorNamespace,
		},
		Prometheus: prometheusChartValue{
			Enabled: true,
		},
	}

	values.Autoscaling.Enabled = false
	values.Prometheus.ExternalURL = prometheusExternalURL
	values.Ingress.Enabled = false
	values.Web.Enabled = true
	values.Grafana.Enabled = true
	values.Grafana.ExternalURL = "/grafana"
	values.Grafana.Security.Enabled = false
	values.Web.Env = map[string]string{
		"API_URL": "api",
	}

	if m.Configuration.internalConfig.backyards.imageRepository != "" {
		values.Application.Image.Repository = m.Configuration.internalConfig.backyards.imageRepository
	}
	if m.Configuration.internalConfig.backyards.imageTag != "" {
		values.Application.Image.Tag = m.Configuration.internalConfig.backyards.imageTag
	}

	if m.Configuration.internalConfig.backyards.webImageTag != "" {
		values.Web.Image.Tag = m.Configuration.internalConfig.backyards.webImageTag
	}

	valuesOverride, err := yaml.Marshal(values)
	if err != nil {
		return errors.WrapIf(err, "could not marshal chart value overrides")
	}

	err = installOrUpgradeDeployment(
		c,
		backyardsNamespace,
		m.Configuration.internalConfig.backyards.chartName,
		backyardsReleaseName,
		valuesOverride,
		m.Configuration.internalConfig.backyards.chartVersion,
		true,
		true,
	)
	if err != nil {
		return errors.WrapIf(err, "could not install Backyards")
	}

	return nil
}
