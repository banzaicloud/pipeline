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

package ec2

import (
	"time"

	"emperror.dev/emperror"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const instancePipelineCreatedTag = "pipeline-created"

// SpotMetricsExporter describes
type SpotMetricsExporter struct {
	logger logrus.FieldLogger

	ec2SpotRequestDuration *prometheus.SummaryVec
	ec2SpotRequest         *prometheus.GaugeVec

	lastRun time.Time
}

// NewSpotMetricsExporter gives back an initialized SpotMetricsExporter
func NewSpotMetricsExporter(logger logrus.FieldLogger, namespace string) *SpotMetricsExporter {

	e := &SpotMetricsExporter{
		logger: logger,
	}

	e.ec2SpotRequestDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: namespace,
		Name:      "ec2_spot_request_duration_until_state_seconds",
		Help:      "Duration until an EC2 spot request got into it's current state in seconds",
	},
		[]string{"state", "region", "zone", "type", "instance_id"},
	)

	e.ec2SpotRequest = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "ec2_spot_requests_total",
		Help:      "Current number of EC2 spot request",
	},
		[]string{"state", "region", "zone", "type"},
	)

	prometheus.MustRegister(e.ec2SpotRequestDuration)
	prometheus.MustRegister(e.ec2SpotRequest)

	return e
}

// SetSpotRequestMetrics exposes spot requests information as Prometheus metrics
func (e *SpotMetricsExporter) SetSpotRequestMetrics(requests map[string]*SpotInstanceRequest) {
	var region, availabilityZone, instanceType, instanceId string

	// reset spot requests gauge
	e.ec2SpotRequest.Reset()

	for _, request := range requests {
		if request.LaunchSpecification == nil {
			continue
		}
		if request.LaunchSpecification.Placement != nil && request.LaunchSpecification.Placement.AvailabilityZone != nil {
			availabilityZone = *request.LaunchSpecification.Placement.AvailabilityZone
			if len(availabilityZone) > 1 {
				region = availabilityZone[:len(availabilityZone)-1]
			}
		}
		if request.LaunchSpecification.InstanceType != nil {
			instanceType = *request.LaunchSpecification.InstanceType
		}

		instanceId = ""
		if request.InstanceId != nil && request.IsPipelineRelated() {
			instanceId = *request.InstanceId
		}

		// increment gauge
		var statusCode string
		if request.Status != nil && request.Status.Code != nil {
			statusCode = *request.Status.Code
		}
		e.ec2SpotRequest.WithLabelValues(statusCode, region, availabilityZone, instanceType).Inc()

		// observe state duration
		if e.needsMeasure(request, e.lastRun) && request.CreateTime != nil {
			e.ec2SpotRequestDuration.WithLabelValues(statusCode, region, availabilityZone, instanceType, instanceId).Observe(request.Status.UpdateTime.Sub(*request.CreateTime).Seconds())
		}
	}

	e.lastRun = time.Now()
}

// GetSpotRequests gets spot requests from EC2 and sets related instance pipeline tags on them
func (e *SpotMetricsExporter) GetSpotRequests(client *ec2.EC2) (map[string]*SpotInstanceRequest, error) {
	input := &ec2.DescribeSpotInstanceRequestsInput{}
	result, err := client.DescribeSpotInstanceRequests(input)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get spot requests")
	}

	requests := make(map[string]*SpotInstanceRequest)
	for _, sr := range result.SpotInstanceRequests {
		if sr.SpotInstanceRequestId == nil {
			continue
		}
		if requests[*sr.SpotInstanceRequestId] == nil {
			requests[*sr.SpotInstanceRequestId] = e.addPipelineCreatedInstanceTag(client, &SpotInstanceRequest{SpotInstanceRequest: sr})
		}
	}

	return requests, nil
}

func (e *SpotMetricsExporter) addPipelineCreatedInstanceTag(client *ec2.EC2, request *SpotInstanceRequest) *SpotInstanceRequest {
	if request.InstanceId == nil {
		return request
	}

	i, err := DescribeInstanceById(client, *request.InstanceId)
	if err != nil {
		return request
	}

	if len(i.Tags) == 0 {
		return request
	}

	tags := request.Tags
	for _, tag := range i.Tags {
		if tag.Key != nil && tag.Value != nil && *tag.Key == instancePipelineCreatedTag && *tag.Value == "true" {
			tags = append(tags, tag)
			break
		}
	}

	return &SpotInstanceRequest{SpotInstanceRequest: request.SetTags(tags)}
}

func (e *SpotMetricsExporter) needsMeasure(request *SpotInstanceRequest, lastRun time.Time) bool {
	if request.Status == nil || request.Status.UpdateTime == nil || request.Status.UpdateTime.Before(lastRun) {
		return false
	}

	if request.GetState() == "open" {
		return true
	}

	switch request.GetStatusCode() {
	case "fulfilled", "request-canceled-and-instance-running", "marked-for-stop", "marked-for-termination", "instance-stopped-by-price", "instance-stopped-by-user", "instance-stopped-capacity-oversubscribed", "instance-stopped-no-capacity", "instance-terminated-by-price", "instance-terminated-by-schedule", "instance-terminated-by-service", "instance-terminated-no-capacity", "instance-terminated-capacity-oversubscribed", "instance-terminated-launch-group-constraint":
		return true
	}

	return false
}
