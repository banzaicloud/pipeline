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

package monitoring

type prometheusOperatorValues struct {
	PrometheusOperator operatorSpecValues     `json:"prometheusOperator"`
	Grafana            *grafanaValues         `json:"grafana"`
	Alertmanager       *alertmanagerValues    `json:"alertmanager"`
	Prometheus         *prometheusValues      `json:"prometheus"`
	KubeStateMetrics   kubeStateMetricsValues `json:"kubeStateMetrics"`
	NodeExporter       nodeExporterValues     `json:"nodeExporter"`
	KsmValues          *ksmValues             `json:"kube-state-metrics"`
	NeValues           *neValues              `json:"prometheus-node-exporter"`
}

type operatorSpecValues struct {
	Image                 imageValues `json:"image"`
	CleanupCustomResource bool        `json:"cleanupCustomResource"`
}

type imageValues struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

type prometheusPushgatewayValues struct {
	Image imageValues `json:"image"`
}

type baseValues struct {
	Enabled bool          `json:"enabled"`
	Ingress ingressValues `json:"ingress"`
}

type grafanaValues struct {
	baseValues

	AdminUser                string            `json:"adminUser"`
	AdminPassword            string            `json:"adminPassword"`
	GrafanaIni               grafanaIniValues  `json:"grafana.ini"`
	DefaultDashboardsEnabled bool              `json:"defaultDashboardsEnabled"`
	Image                    imageValues       `json:"image"`
	Persistence              persistenceValues `json:"persistence"`
	Sidecar                  sidecar           `json:"sidecar"`
}

type sidecar struct {
	Datasources datasources `json:"datasources"`
}

type datasources struct {
	Enabled         bool   `json:"enabled"`
	Label           string `json:"label"`
	SearchNamespace string `json:"searchNamespace"`
}

type persistenceValues struct {
	Enabled bool `json:"enabled"`
}

type grafanaIniValues struct {
	Server grafanaIniServerValues `json:"server"`
}

type grafanaIniServerValues struct {
	RootUrl          string `json:"root_url"`
	ServeFromSubPath bool   `json:"serve_from_sub_path"`
}

type alertmanagerValues struct {
	baseValues
	Spec   baseSpecValues `json:"alertmanagerSpec"`
	Config *configValues  `json:"config"`
}

type configValues struct {
	Receivers []receiverItemValues `json:"receivers"`
	Route     routeValues          `json:"route"`
}

type routeValues struct {
	Receiver string        `json:"receiver"`
	Routes   []interface{} `json:"routes"`
}

type receiverItemValues struct {
	Name             string                  `json:"name"`
	SlackConfigs     []slackConfigValues     `json:"slack_configs,omitempty"`
	PagerdutyConfigs []pagerdutyConfigValues `json:"pagerduty_config,omitempty"`
}

type slackConfigValues struct {
	ApiUrl       string `json:"api_url"`
	Channel      string `json:"channel"`
	SendResolved bool   `json:"send_resolved"`
}

type pagerdutyConfigValues struct {
	RoutingKey   string `json:"routing_key"`
	ServiceKey   string `json:"service_key"`
	Url          string `json:"url"`
	SendResolved bool   `json:"send_resolved"`
}

type baseSpecValues struct {
	RoutePrefix string      `json:"routePrefix"`
	Image       imageValues `json:"image"`
}

type PrometheusSpecValues struct {
	baseSpecValues
	RetentionSize                           string                 `json:"retentionSize"`
	Retention                               string                 `json:"retention"`
	StorageSpec                             map[string]interface{} `json:"storageSpec"`
	ServiceMonitorSelectorNilUsesHelmValues bool                   `json:"serviceMonitorSelectorNilUsesHelmValues"`
}

type prometheusValues struct {
	baseValues
	Spec        PrometheusSpecValues   `json:"prometheusSpec"`
	Annotations map[string]interface{} `json:"annotations"`
}

type kubeStateMetricsValues struct {
	Enabled bool `json:"enabled"`
}

type ksmValues struct {
	Image imageValues `json:"image"`
}

type neValues struct {
	Image imageValues `json:"image"`
}

type nodeExporterValues struct {
	Enabled bool `json:"enabled"`
}

type ingressValues struct {
	Enabled     bool                   `json:"enabled"`
	Hosts       []string               `json:"hosts"`
	Path        string                 `json:"path,omitempty"`
	Paths       []string               `json:"paths,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}
