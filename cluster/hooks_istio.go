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

package cluster

import (
	"fmt"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/banzaicloud/pipeline/utils"
	prometheus "github.com/banzaicloud/prometheus-config"
	"github.com/ghodss/yaml"
	"github.com/goph/emperror"
	"github.com/prometheus/common/model"
	"github.com/spf13/viper"
	yamlv2 "gopkg.in/yaml.v2"
)

const (
	promConfigEntry   = "prometheus.yml"
	cmName            = "-prometheus-server"
	istioNamespace    = "istio-system"
	istioMeshJob      = "istio-mesh"
	envoyStatsJob     = "envoy-stats"
	istioPolicyJob    = "istio-policy"
	istioTelemetryJob = "istio-telemetry"
	pilotJob          = "pilot"
	galleyJob         = "galley"
)

var nsLabels = map[string]string{
	"istio-injection": "enabled",
}

// InstallServiceMeshParams describes InstallServiceMesh posthook params
type InstallServiceMeshParams struct {
	// AutoSidecarInjectNamespaces list of namespaces that will be labelled with istio-injection=enabled
	AutoSidecarInjectNamespaces []string `json:"autoSidecarInjectNamespaces,omitempty"`
	// EnableMtls signals if mutual TLS is enabled in the service mesh
	EnableMtls bool `json:"mtls,omitempty"`
}

// InstallServiceMesh is a posthook for installing Istio on a cluster
func InstallServiceMesh(cluster CommonCluster, param cluster.PostHookParam) error {
	var params InstallServiceMeshParams
	err := castToPostHookParam(&param, &params)
	if err != nil {
		return emperror.Wrap(err, "failed to cast posthook param")
	}

	log.Infof("istio params: %#v", params)

	values := map[string]interface{}{}

	marshalledValues, err := yaml.Marshal(values)
	if err != nil {
		return emperror.Wrap(err, "failed to marshal yaml values")
	}

	err = installDeployment(
		cluster,
		istioNamespace,
		pkgHelm.BanzaiRepository+"/istio",
		"istio",
		marshalledValues,
		"",
		false,
	)
	if err != nil {
		return emperror.Wrap(err, "installing Istio failed")
	}

	err = labelNamespaces(cluster, params.AutoSidecarInjectNamespaces)
	if err != nil {
		return emperror.Wrap(err, "failed to label namespace")
	}

	if cluster.GetMonitoring() {
		err = addPrometheusTargets(cluster)
		if err != nil {
			log.WithError(err).Infof("wat")
			return emperror.Wrap(err, "failed to add prometheus targets")
		}
	}

	cluster.SetServiceMesh(true)
	return nil
}

func labelNamespaces(cluster CommonCluster, namespaces []string) error {
	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return emperror.Wrap(err, "failed to get kubeconfig")
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to create client from kubeconfig")
	}

	for _, ns := range namespaces {
		err = k8sutil.LabelNamespaceIgnoreNotFound(log, client, ns, nsLabels)
		if err != nil {
			return emperror.Wrap(err, "failed to label namespace")
		}
	}
	return nil
}

func addPrometheusTargets(cluster CommonCluster) error {
	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return emperror.Wrap(err, "failed to get kubeconfig")
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to create client from kubeconfig")
	}

	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)

	currPromConfStr, err := k8sutil.GetConfigMapEntry(client, pipelineSystemNamespace, config.MonitorReleaseName+cmName, promConfigEntry)
	if err != nil {
		return emperror.Wrap(err, "failed to get Prometheus config")
	}

	var currPromConf prometheus.Config
	err = yamlv2.Unmarshal([]byte(currPromConfStr), &currPromConf)
	if err != nil {
		return emperror.Wrap(err, "failed to patch Prometheus config")
	}

	scrapeConfigs := currPromConf.ScrapeConfigs
	jobNames := collectJobNames(scrapeConfigs)

	istioScrapeConfigs := []*prometheus.ScrapeConfig{
		istioServiceScrapeConfig(istioMeshJob, "istio-telemetry;prometheus"),
		istioServiceScrapeConfig(istioPolicyJob, "istio-policy;http-monitoring"),
		istioServiceScrapeConfig(istioTelemetryJob, "istio-telemetry;http-monitoring"),
		istioServiceScrapeConfig(pilotJob, "istio-pilot;http-monitoring"),
		istioServiceScrapeConfig(galleyJob, "istio-galley;http-monitoring"),
		envoyStatsScrapeConfig(),
	}

	for _, sc := range istioScrapeConfigs {
		if !utils.Contains(jobNames, sc.JobName) {
			scrapeConfigs = append(scrapeConfigs, sc)
		}
	}

	newPromConf := currPromConf
	newPromConf.ScrapeConfigs = scrapeConfigs

	newPromConfStr, err := yamlv2.Marshal(newPromConf)
	if err != nil {
		return emperror.Wrap(err, "failed to patch Prometheus config")
	}

	err = k8sutil.PatchConfigMap(log, client, pipelineSystemNamespace, config.MonitorReleaseName+cmName, promConfigEntry, string(newPromConfStr))
	if err != nil {
		return emperror.Wrap(err, "failed to patch Prometheus config")
	}
	return nil
}

func collectJobNames(scrapeConfigs []*prometheus.ScrapeConfig) []string {
	var jobNames []string
	for _, sc := range scrapeConfigs {
		if sc.JobName == envoyStatsJob {
			for _, rc := range sc.RelabelConfigs {
				fmt.Println("*2**", rc.Regex.String())
			}
		}
		jobNames = append(jobNames, sc.JobName)
	}
	return jobNames
}

var kubernetesSDEndpointsRole = prometheus.ServiceDiscoveryConfig{
	KubernetesSDConfigs: []*prometheus.KubernetesSDConfig{
		{
			Role: "endpoints",
			NamespaceDiscovery: prometheus.NamespaceDiscovery{
				Names: []string{istioNamespace},
			},
		},
	},
}

func endpointsRoleRelabelConfigs(regex string) []*prometheus.RelabelConfig {
	return []*prometheus.RelabelConfig{
		{
			Action: "keep",
			Regex:  prometheus.MustNewRegexp(regex),
			SourceLabels: model.LabelNames{
				"__meta_kubernetes_service_name",
				"__meta_kubernetes_endpoint_port_name",
			},
		},
	}
}

func istioServiceScrapeConfig(jobName string, relabelConfigRegex string) *prometheus.ScrapeConfig {
	return &prometheus.ScrapeConfig{
		JobName:                jobName,
		ServiceDiscoveryConfig: kubernetesSDEndpointsRole,
		RelabelConfigs:         endpointsRoleRelabelConfigs(relabelConfigRegex),
	}
}

func envoyStatsScrapeConfig() *prometheus.ScrapeConfig {
	return &prometheus.ScrapeConfig{
		JobName:     envoyStatsJob,
		MetricsPath: "/stats/prometheus",
		ServiceDiscoveryConfig: prometheus.ServiceDiscoveryConfig{
			KubernetesSDConfigs: []*prometheus.KubernetesSDConfig{
				{
					Role: "pod",
				},
			},
		},
		RelabelConfigs: []*prometheus.RelabelConfig{
			{
				Action: "keep",
				Regex:  prometheus.MustNewRegexp(".*-envoy-prom"),
				SourceLabels: model.LabelNames{
					"__meta_kubernetes_pod_container_port_name",
				},
			},
			{
				Action: "replace",
				Regex:  prometheus.MustNewRegexp("([^:]+)(?::\\d+)?;(\\d+)"),
				SourceLabels: model.LabelNames{
					"__address__",
					"__meta_kubernetes_pod_annotation_prometheus_io_port",
				},
				Replacement: "$1:15090",
				TargetLabel: "__address__",
			},
			{
				Action: "labelmap",
				Regex:  prometheus.MustNewRegexp("__meta_kubernetes_pod_label_(.+)"),
			},
			{
				Action:      "replace",
				TargetLabel: "namespace",
				SourceLabels: model.LabelNames{
					"__meta_kubernetes_namespace",
				},
			},
			{
				Action:      "replace",
				TargetLabel: "pod_name",
				SourceLabels: model.LabelNames{
					"__meta_kubernetes_pod_name",
				},
			},
		},
		MetricRelabelConfigs: []*prometheus.RelabelConfig{
			{
				Action: "drop",
				Regex:  prometheus.MustNewRegexp("(outbound|inbound|prometheus_stats).*"),
				SourceLabels: model.LabelNames{
					"cluster_name",
				},
			},
			{
				Action: "drop",
				Regex:  prometheus.MustNewRegexp("(outbound|inbound|prometheus_stats).*"),
				SourceLabels: model.LabelNames{
					"tcp_prefix",
				},
			},
			{
				Action: "drop",
				Regex:  prometheus.MustNewRegexp("(.+)"),
				SourceLabels: model.LabelNames{
					"listener_address",
				},
			},
			{
				Action: "drop",
				Regex:  prometheus.MustNewRegexp("(.+)"),
				SourceLabels: model.LabelNames{
					"http_conn_manager_listener_prefix",
				},
			},
			{
				Action: "drop",
				Regex:  prometheus.MustNewRegexp("(.+)"),
				SourceLabels: model.LabelNames{
					"http_conn_manager_prefix",
				},
			},
			{
				Action: "drop",
				Regex:  prometheus.MustNewRegexp("envoy_tls.*"),
				SourceLabels: model.LabelNames{
					"__name__",
				},
			},
			{
				Action: "drop",
				Regex:  prometheus.MustNewRegexp("envoy_tcp_downstream.*"),
				SourceLabels: model.LabelNames{
					"__name__",
				},
			},
			{
				Action: "drop",
				Regex:  prometheus.MustNewRegexp("envoy_http_(stats|admin).*"),
				SourceLabels: model.LabelNames{
					"__name__",
				},
			},
			{
				Action: "drop",
				Regex:  prometheus.MustNewRegexp("envoy_cluster_(lb|retry|bind|internal|max|original).*"),
				SourceLabels: model.LabelNames{
					"__name__",
				},
			},
		},
	}
}
