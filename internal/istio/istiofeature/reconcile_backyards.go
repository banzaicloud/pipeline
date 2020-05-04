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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	internalHelm "github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/pkg/backoff"
	"github.com/banzaicloud/pipeline/src/cluster"
)

type monitoringConfig struct {
	hostname string
	url      string
}

func (m *MeshReconciler) ReconcileBackyards(desiredState DesiredState, c cluster.CommonCluster, remote bool) error {
	m.logger.Debug("reconciling Backyards")
	defer m.logger.Debug("Backyards reconciled")

	if desiredState == DesiredStatePresent {
		client, err := m.getRuntimeK8sClient(c)
		if err != nil {
			return errors.WrapIf(err, "could not get api extension client")
		}

		if !remote {
			err = m.waitForCRD("instances.config.istio.io", client)
			if err != nil {
				return errors.WrapIf(err, "error while waiting for metric CRD")
			}
		}

		podLabels := map[string]string{"app": "istiod"}
		imageWithTag := m.Configuration.internalConfig.Istio.PilotImage
		if remote {
			podLabels = map[string]string{"app": "istio-sidecar-injector"}
			imageWithTag = m.Configuration.internalConfig.Istio.SidecarInjectorImage
		}
		err = m.waitForPod(client, istioOperatorNamespace, podLabels, imageWithTag)
		if err != nil {
			return errors.WrapIf(err, "error while waiting for running sidecar injector")
		}

		err = m.installBackyards(c, monitoringConfig{
			hostname: prometheusHostname,
			url:      prometheusURL,
		}, remote)
		if err != nil {
			return errors.WrapIf(err, "could not install Backyards")
		}

		return nil
	}

	return errors.WrapIf(m.uninstallBackyards(c), "could not remove Backyards")
}

// waitForPod waits for pods to be running
func (m *MeshReconciler) waitForPod(client runtimeclient.Client, namespace string, labels map[string]string, containerImageWithTag string) error {
	m.logger.Debug("waiting for pod")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(backoffDelaySeconds) * time.Second,
		MaxRetries: backoffMaxretries,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	err := backoff.Retry(func() error {
		var pods corev1.PodList
		o := &runtimeclient.ListOptions{}
		runtimeclient.InNamespace(namespace).ApplyToList(o)
		runtimeclient.MatchingLabels(labels).ApplyToList(o)

		err := client.List(context.Background(), &pods, o)
		if err != nil {
			return errors.WrapIf(err, "could not list pods")
		}

		for _, pod := range pods.Items {
			if containerImageWithTag != "" {
				match := false
				for _, container := range pod.Spec.Containers {
					if container.Image == containerImageWithTag {
						match = true
						break
					}
				}
				if !match {
					continue
				}
			}
			if pod.Status.Phase == v1.PodRunning {
				readyContainers := 0
				for _, cs := range pod.Status.ContainerStatuses {
					if cs.Ready {
						readyContainers++
					}
				}
				if readyContainers == len(pod.Status.ContainerStatuses) {
					return nil
				}
			}
		}

		return errors.New("could not find running and healthy pods")
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

		for _, condition := range crd.Status.Conditions {
			if condition.Type == apiextensionsv1beta1.Established {
				if condition.Status == apiextensionsv1beta1.ConditionTrue {
					return nil
				}
			}
		}

		return errors.New("CRD is not established yet")
	}, backoffPolicy)

	return err
}

// uninstallIstioOperator removes istio-operator from a cluster
func (m *MeshReconciler) uninstallBackyards(c cluster.CommonCluster) error {
	m.logger.Debug("removing Backyards")

	return errors.WrapIf(m.helmService.Delete(c, backyardsReleaseName, backyardsNamespace), "could not remove Backyards")
}

// installIstioOperator installs istio-operator on a cluster
func (m *MeshReconciler) installBackyards(c cluster.CommonCluster, monitoring monitoringConfig, remote bool) error {
	m.logger.Debug("installing Backyards")

	type istio struct {
		CRname    string `json:"CRName,omitempty"`
		Namespace string `json:"namespace,omitempty"`
	}

	type application struct {
		Enabled bool              `json:"enabled"`
		Image   imageChartValue   `json:"image,omitempty"`
		Env     map[string]string `json:"env,omitempty"`
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
		ALS struct {
			Enabled bool `json:"enabled"`
		} `json:"als,omitempty"`
		IngressGateway struct {
			Enabled bool `json:"enabled"`
		} `json:"ingressgateway,omitempty"`
		AuditSink struct {
			Enabled bool `json:"enabled"`
		} `json:"auditsink,omitempty"`
		KubeStateMetrics struct {
			Enabled bool `json:"enabled"`
		} `json:"kubestatemetrics,omitempty"`
		Tracing struct {
			Enabled bool `json:"enabled"`
		} `json:"tracing,omitempty"`
		UseIstioResources bool `json:"useIstioResources"`
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
	values.Application.Enabled = true
	values.Web.Enabled = true
	values.Grafana.Enabled = true
	values.Grafana.ExternalURL = "/grafana"
	values.Grafana.Security.Enabled = false
	values.Web.Env = map[string]string{
		"API_URL": "api",
	}
	values.IngressGateway.Enabled = true
	values.KubeStateMetrics.Enabled = true
	values.ALS.Enabled = true
	values.Grafana.Enabled = true
	values.Tracing.Enabled = true
	values.UseIstioResources = true

	backyardsChart := m.Configuration.internalConfig.Charts.Backyards

	if backyardsChart.Values.Application.Image.Repository != "" {
		values.Application.Image.Repository = backyardsChart.Values.Application.Image.Repository
	}
	if backyardsChart.Values.Application.Image.Tag != "" {
		values.Application.Image.Tag = backyardsChart.Values.Application.Image.Tag
	}

	if backyardsChart.Values.Web.Image.Repository != "" {
		values.Web.Image.Repository = backyardsChart.Values.Web.Image.Repository
	}
	if backyardsChart.Values.Web.Image.Tag != "" {
		values.Web.Image.Tag = backyardsChart.Values.Web.Image.Tag
	}

	if remote {
		values.Application.Enabled = false
		values.IngressGateway.Enabled = false
		values.ALS.Enabled = false
		values.Web.Enabled = false
		values.AuditSink.Enabled = false
		values.Autoscaling.Enabled = false
		values.Grafana.Enabled = false
		values.Tracing.Enabled = false
		values.Prometheus.Enabled = true
		values.Prometheus.ClusterName = c.GetName()
		values.UseIstioResources = false
	}

	mapStringValues, err := ConvertStructure(values)
	if err != nil {
		return errors.WrapIf(err, "failed to convert backyards chart values")
	}

	err = m.helmService.InstallOrUpgrade(
		c,
		internalHelm.Release{
			ReleaseName: backyardsReleaseName,
			ChartName:   backyardsChart.Chart,
			Namespace:   backyardsNamespace,
			Values:      mapStringValues,
			Version:     backyardsChart.Version,
		},
		internalHelm.Options{
			Namespace: backyardsNamespace,
			Wait:      true,
			Install:   true,
		},
	)

	return errors.WrapIf(err, "could not install Backyards")
}
