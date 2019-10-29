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

import (
	"fmt"
)

const (
	featureName                      = "monitoring"
	prometheusOperatorReleaseName    = "monitor"
	prometheusPushgatewayReleaseName = "pushgateway"
	grafanaSecretTag                 = "app:grafana"
	prometheusSecretTag              = "app:prometheus"
	alertmanagerSecretTag            = "app:alertmanager"
	pushgatewaySecretTag             = "app:pushgateway"
	featureSecretTag                 = "feature:monitoring"
	generatedSecretUsername          = "admin"
	alertManagerProviderConfigName   = "pipeline-monitoring-feature-providers"

	ingressTypeGrafana      = "Grafana"
	ingressTypePrometheus   = "Prometheus"
	ingressTypeAlertmanager = "Alertmanager"
	ingressTypePushgateway  = "Pushgateway"

	pagerDutyIntegrationEventApiV2 = "eventsApiV2"
	pagerDutyIntegrationPrometheus = "prometheus"

	alertmanagerProviderSlack     = "slack"
	alertmanagerProviderPagerDuty = "pagerDuty"
)

func getClusterNameSecretTag(clusterName string) string {
	return fmt.Sprintf("cluster:%s", clusterName)
}

func getClusterUIDSecretTag(clusterUID string) string {
	return fmt.Sprintf("clusterUID:%s", clusterUID)
}

func getReleaseSecretTag() string {
	return fmt.Sprintf("release:%s", prometheusOperatorReleaseName)
}

func getPrometheusSecretName(clusterID uint) string {
	return fmt.Sprintf("cluster-%d-prometheus", clusterID)
}

func getAlertmanagerSecretName(clusterID uint) string {
	return fmt.Sprintf("cluster-%d-alertmanager", clusterID)
}

func getPushgatewaySecretName(clusterID uint) string {
	return fmt.Sprintf("cluster-%d-pushgateway", clusterID)
}

func getGrafanaSecretName(clusterID uint) string {
	return fmt.Sprintf("cluster-%d-grafana", clusterID)
}

func generateAnnotations(secretName string) map[string]interface{} {
	return map[string]interface{}{
		"traefik.ingress.kubernetes.io/auth-type":   "basic",
		"traefik.ingress.kubernetes.io/auth-secret": secretName,
	}
}
