// Copyright © 2020 Banzai Cloud
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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

type ApiMetrics struct {
	timer   prometheus.ObserverVec
	counter *prometheus.CounterVec
}

func NewApiMetrics(version string) *ApiMetrics {
	timerVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "pipeline",
		Name:      "integrated_service_api_request_time",
		Help:      "Integrated Service API call durations",
	}, []string{
		"version", "cluster", "service", "action",
	})
	errorVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "pipeline",
		Name:      "integrated_service_api_request_error",
		Help:      "Integrated Service API errors",
	}, []string{
		"version", "cluster", "service", "action", "error",
	})

	prometheus.MustRegister(timerVec, errorVec)

	return &ApiMetrics{
		timer: timerVec.MustCurryWith(prometheus.Labels{
			"version": version,
		}),
		counter: errorVec.MustCurryWith(prometheus.Labels{
			"version": version,
		}),
	}
}

func (a *ApiMetrics) RequestTimer(clusterID uint, service, action string) integratedservices.DurationObserver {
	return prometheus.NewTimer(a.timer.With(prometheus.Labels{
		"cluster": cast.ToString(clusterID),
		"service": service,
		"action":  action,
	}))
}

func (a *ApiMetrics) ErrorCounter(clusterID uint, service, action string) integratedservices.ErrorCounter {
	return &ErrorCounter{
		counter: a.counter.MustCurryWith(prometheus.Labels{
			"cluster": cast.ToString(clusterID),
			"service": service,
			"action":  action,
		}),
	}
}

type ErrorCounter struct {
	counter *prometheus.CounterVec
}

func (e *ErrorCounter) Increment(err string) {
	e.counter.With(prometheus.Labels{
		"error": err,
	}).Inc()
}
