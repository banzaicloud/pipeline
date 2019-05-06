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

package istio

import (
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/banzaicloud/pipeline/utils"
	prometheus "github.com/banzaicloud/prometheus-config"
	"github.com/goph/emperror"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	yamlv2 "gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
)

const (
	promConfigEntry = "prometheus.yml"
	promCmName      = "-prometheus-server"
)

func RemovePrometheusTargets(log logrus.FieldLogger, client kubernetes.Interface) error {
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)

	currPromConfStr, err := k8sutil.GetConfigMapEntry(client, pipelineSystemNamespace, config.MonitorReleaseName+promCmName, promConfigEntry)
	if err != nil {
		return emperror.Wrap(err, "failed to get Prometheus config")
	}

	var currPromConf prometheus.Config
	err = yamlv2.Unmarshal([]byte(currPromConfStr), &currPromConf)
	if err != nil {
		return emperror.Wrap(err, "failed to patch Prometheus config")
	}

	istioScrapeConfigs := make([]*prometheus.ScrapeConfig, 0)

	for _, scrapeConfig := range currPromConf.ScrapeConfigs {
		switch scrapeConfig.JobName {
		case "istio-mesh", "istio-policy", "istio-telemetry", "pilot", "galley":
			continue
		default:
			istioScrapeConfigs = append(istioScrapeConfigs, scrapeConfig)
		}
	}

	newPromConf := currPromConf
	newPromConf.ScrapeConfigs = istioScrapeConfigs

	newPromConfStr, err := yamlv2.Marshal(newPromConf)
	if err != nil {
		return emperror.Wrap(err, "failed to patch Prometheus config")
	}

	err = k8sutil.PatchConfigMapDataEntry(log, client, pipelineSystemNamespace, config.MonitorReleaseName+promCmName, promConfigEntry, string(newPromConfStr))
	if err != nil {
		return emperror.Wrap(err, "failed to patch Prometheus config")
	}
	return nil
}

func AddPrometheusTargets(log logrus.FieldLogger, client kubernetes.Interface) error {
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)

	currPromConfStr, err := k8sutil.GetConfigMapEntry(client, pipelineSystemNamespace, config.MonitorReleaseName+promCmName, promConfigEntry)
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
		istioServiceScrapeConfig("istio-mesh", "istio-telemetry;prometheus"),
		istioServiceScrapeConfig("istio-policy", "istio-policy;http-monitoring"),
		istioServiceScrapeConfig("istio-telemetry", "istio-telemetry;http-monitoring"),
		istioServiceScrapeConfig("pilot", "istio-pilot;http-monitoring"),
		istioServiceScrapeConfig("galley", "istio-galley;http-monitoring"),
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

	err = k8sutil.PatchConfigMapDataEntry(log, client, pipelineSystemNamespace, config.MonitorReleaseName+promCmName, promConfigEntry, string(newPromConfStr))
	if err != nil {
		return emperror.Wrap(err, "failed to patch Prometheus config")
	}
	return nil
}

func collectJobNames(scrapeConfigs []*prometheus.ScrapeConfig) []string {
	var jobNames []string
	for _, sc := range scrapeConfigs {
		jobNames = append(jobNames, sc.JobName)
	}
	return jobNames
}

// nolint: gochecknoglobals
var kubernetesSDEndpointsRole = prometheus.ServiceDiscoveryConfig{
	KubernetesSDConfigs: []*prometheus.KubernetesSDConfig{
		{
			Role: "endpoints",
			NamespaceDiscovery: prometheus.NamespaceDiscovery{
				Names: []string{Namespace},
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
		JobName:     "envoy-stats",
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
