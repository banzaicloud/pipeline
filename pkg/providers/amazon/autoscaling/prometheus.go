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

package autoscaling

import (
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
	"github.com/prometheus/client_golang/prometheus"
)

// nolint: gochecknoglobals
var (
	timers map[string]*prometheus.Timer

	ec2InstanceStartupDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "pipeline",
		Name:      "ec2_instance_startup_duration_seconds",
		Help:      "EC2 instance startup duration in seconds",
	},
		[]string{"provider", "region", "zone", "type", "price_type"},
	)
	ec2SpotInstanceFulfillmentDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "pipeline",
		Name:      "ec2_spot_instance_fulfillment_duration_seconds",
		Help:      "EC2 spot instance fulfillment duration in seconds",
	},
		[]string{"provider", "region", "zone", "type"},
	)
)

func init() {
	timers = make(map[string]*prometheus.Timer)
	prometheus.MustRegister(ec2InstanceStartupDuration)
	prometheus.MustRegister(ec2SpotInstanceFulfillmentDuration)
}

// StartMetricTimer starts a prometheus metric timer for a node instance
func (m *Manager) StartMetricTimer(instance *Instance) *prometheus.Timer {
	var region, availabilityZone, instanceType, priceType string

	if !m.metricsEnabled {
		return nil
	}

	if instance.InstanceId == nil {
		return nil
	}

	key := *instance.InstanceId
	if timers[key] != nil {
		return timers[key]
	}

	m.logger.WithField("instance-id", *instance.InstanceId).Debug("start metric timer")

	instanceDetails, err := instance.Describe()
	if err == nil {
		if instanceDetails.InstanceType != nil {
			instanceType = *instanceDetails.InstanceType
		}
		priceType = "ondemand"
		if instanceDetails.InstanceLifecycle != nil && *instanceDetails.InstanceLifecycle == "spot" {
			priceType = "spot"
		}
	}

	if instance.AvailabilityZone != nil {
		availabilityZone = *instance.AvailabilityZone
		if len(availabilityZone) > 1 {
			region = availabilityZone[:len(availabilityZone)-1]
		}
	}

	timers[key] = prometheus.NewTimer(ec2InstanceStartupDuration.WithLabelValues(amazon.Provider, region, availabilityZone, instanceType, priceType))

	return timers[key]
}

// StopMetricTimer stops an existing timer of a node instance
func (m *Manager) StopMetricTimer(instance *Instance) bool {
	if instance.InstanceId == nil {
		return false
	}

	key := *instance.InstanceId

	if timers[key] == nil {
		return false
	}

	m.logger.WithField("instance-id", key).Debug("stop metric timer")
	timers[key].ObserveDuration()
	timers[key] = nil

	return true
}

// RegisterSpotFulfillmentDuration checks whether a node instance has a fulfilled spot request related to it
// and sets the fulfillment duration into a Prometheus metric
func (m *Manager) RegisterSpotFulfillmentDuration(instance *Instance, group *Group) {
	var region, availabilityZone, instanceType string

	if !m.metricsEnabled {
		return
	}

	if instance.InstanceId == nil {
		return
	}

	instanceDetails, err := instance.Describe()
	if err == nil && instanceDetails.InstanceType != nil {
		instanceType = *instanceDetails.InstanceType
	}
	if instance.AvailabilityZone != nil {
		availabilityZone = *instance.AvailabilityZone
		if len(availabilityZone) > 1 {
			region = availabilityZone[:len(availabilityZone)-1]
		}
	}

	spotRequests, err := group.getSpotRequests()
	if err == nil {
		for _, sr := range spotRequests {
			if sr.InstanceId != nil && sr.CreateTime != nil && *sr.InstanceId == *instance.InstanceId && sr.IsFulfilled() {
				m.logger.WithFields(logrus.Fields{
					"instance-id": *instance.InstanceId,
					"seconds":     sr.Status.UpdateTime.Sub(*sr.CreateTime).Seconds(),
				}).Debug("register fulfillment duration")
				ec2SpotInstanceFulfillmentDuration.WithLabelValues(amazon.Provider, region, availabilityZone, instanceType).Observe(sr.Status.UpdateTime.Sub(*sr.CreateTime).Seconds())
				break
			}
		}
	}
}
