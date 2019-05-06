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

package prometheus

import (
	"github.com/banzaicloud/pipeline/internal/cluster/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusClusterStatusChangeDurationMetric struct {
	*prometheus.SummaryVec
}

func MakePrometheusClusterStatusChangeDurationMetric() PrometheusClusterStatusChangeDurationMetric {
	return PrometheusClusterStatusChangeDurationMetric{
		prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace: "pipeline",
				Name:      "cluster_status_change_duration",
				Help:      "Cluster status change duration in seconds",
			},
			[]string{"provider", "location", "status", "orgName", "clusterName"},
		),
	}
}

func (m PrometheusClusterStatusChangeDurationMetric) StartTimer(values metrics.ClusterStatusChangeDurationMetricValues) metrics.DurationMetricTimer {
	return prometheusDurationMetricTimer{
		timer: prometheus.NewTimer(m.WithLabelValues(values.ProviderName, values.LocationName, values.Status, values.OrganizationName, values.ClusterName)),
	}
}

type prometheusDurationMetricTimer struct {
	timer *prometheus.Timer
}

func (t prometheusDurationMetricTimer) RecordDuration() {
	t.timer.ObserveDuration()
}
