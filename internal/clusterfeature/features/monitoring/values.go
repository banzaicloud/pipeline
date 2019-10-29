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
	Grafana          *grafanaValues         `json:"grafana"`
	Alertmanager     *alertmanagerValues    `json:"alertmanager"`
	Prometheus       *prometheusValues      `json:"prometheus"`
	KubeStateMetrics kubeStateMetricsValues `json:"kubeStateMetrics"`
	NodeExporter     nodeExporterValues     `json:"nodeExporter"`
}

type prometheusPushgatewayValues struct {
	affinityValues
	tolerationValues

	Annotations map[string]interface{} `json:"annotations"`
	Ingress     ingressValues          `json:"ingress"`
}

type baseValues struct {
	Enabled bool          `json:"enabled"`
	Ingress ingressValues `json:"ingress"`
}

type grafanaValues struct {
	baseValues
	affinityValues
	tolerationValues

	AdminUser                string           `json:"adminUser"`
	AdminPassword            string           `json:"adminPassword"`
	GrafanaIni               grafanaIniValues `json:"grafana.ini"`
	DefaultDashboardsEnabled bool             `json:"defaultDashboardsEnabled"`
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
	Spec   SpecValues    `json:"alertmanagerSpec"`
	Config *configValues `json:"config"`
}

type configValues struct {
	Global configGlobalValues `json:"global"`
}

type configGlobalValues struct {
	Receivers []receiverItemValues `json:"receivers"`
}

type receiverItemValues struct {
	Name             string                  `json:"name"`
	SlackConfigs     []slackConfigValues     `json:"slack_configs"`
	PagerdutyConfigs []pagerdutyConfigValues `json:"pagerduty_config"`
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

type SpecValues struct {
	tolerationValues
	affinityValues

	RoutePrefix   string                 `json:"routePrefix"`
	RetentionSize string                 `json:"retentionSize"`
	Retention     string                 `json:"retention"`
	StorageSpec   map[string]interface{} `json:"storageSpec"`
}

type prometheusValues struct {
	baseValues
	Spec        SpecValues             `json:"prometheusSpec"`
	Annotations map[string]interface{} `json:"annotations"`
}

type kubeStateMetricsValues struct {
	Enabled bool `json:"enabled"`
	SpecValues
}

type nodeExporterValues struct {
	Enabled bool `json:"enabled"`
}

type affinityValues struct {
	Affinity interface{} `json:"affinity"`
}

type tolerationValues struct {
	Tolerations interface{} `json:"tolerations"`
}

type ingressValues struct {
	Enabled     bool                   `json:"enabled"`
	Hosts       []string               `json:"hosts"`
	Path        string                 `json:"path,omitempty"`
	Paths       []string               `json:"paths,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}
