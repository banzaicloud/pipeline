// Copyright © 2018 Banzai Cloud
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

package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type spotMetricsErrorHandler struct {
	logger                         logrus.FieldLogger
	spotMetricsCollectorErrorCount prometheus.Counter
}

// NewSpotMetricsErrorHandler returns a spot metrics error handler
func NewSpotMetricsErrorHandler(logger logrus.FieldLogger) *spotMetricsErrorHandler {
	h := &spotMetricsErrorHandler{
		logger: logger,
		spotMetricsCollectorErrorCount: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricsNamesapce,
			Name:      "spot_metrics_collector_errors_total",
			Help:      "Total number of errors happened during collecting spot metrics",
		},
		),
	}

	prometheus.MustRegister(h.spotMetricsCollectorErrorCount)

	return h
}

// Handle logs and counts an error
func (h *spotMetricsErrorHandler) Handle(err error) {
	h.logger.Error(err)
	h.spotMetricsCollectorErrorCount.Inc()
}
