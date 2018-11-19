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

	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/prometheus/client_golang/prometheus"
)

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
		[]string{"state", "region", "zone", "type"},
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

func (e *SpotMetricsExporter) SetSpotRequestMetrics(requests map[string]*ec2.SpotInstanceRequest) {
	var region, availabilityZone, instanceType string

	// reset spot requests gauge
	e.ec2SpotRequest.Reset()

	for _, request := range requests {
		if request.LaunchSpecification.Placement.AvailabilityZone != nil {
			availabilityZone = *request.LaunchSpecification.Placement.AvailabilityZone
			if len(availabilityZone) > 1 {
				region = availabilityZone[:len(availabilityZone)-1]
			}
		}
		if request.LaunchSpecification.InstanceType != nil {
			instanceType = *request.LaunchSpecification.InstanceType
		}

		// increment gauge
		e.ec2SpotRequest.WithLabelValues(*request.Status.Code, region, availabilityZone, instanceType).Inc()

		// observe state duration
		if e.needsMeasure(request, e.lastRun) {
			e.ec2SpotRequestDuration.WithLabelValues(*request.Status.Code, region, availabilityZone, instanceType).Observe(request.Status.UpdateTime.Sub(*request.CreateTime).Seconds())
		}
	}

	e.lastRun = time.Now()
}

func (e *SpotMetricsExporter) GetSpotRequests(client *ec2.EC2) (map[string]*ec2.SpotInstanceRequest, error) {
	input := &ec2.DescribeSpotInstanceRequestsInput{}
	result, err := client.DescribeSpotInstanceRequests(input)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get spot requests")
	}

	requests := make(map[string]*ec2.SpotInstanceRequest)
	for _, sr := range result.SpotInstanceRequests {
		if requests[*sr.SpotInstanceRequestId] == nil {
			requests[*sr.SpotInstanceRequestId] = sr
		}
	}

	return requests, nil
}

func (e *SpotMetricsExporter) needsMeasure(request *ec2.SpotInstanceRequest, lastRun time.Time) bool {

	if request.Status.UpdateTime.Before(lastRun) {
		return false
	}

	if *request.State == "open" {
		return true
	}

	switch *request.Status.Code {
	case "fulfilled", "request-canceled-and-instance-running", "marked-for-stop", "marked-for-termination", "instance-stopped-by-price", "instance-stopped-by-user", "instance-stopped-capacity-oversubscribed", "instance-stopped-no-capacity", "instance-terminated-by-price", "instance-terminated-by-schedule", "instance-terminated-by-service", "instance-terminated-no-capacity", "instance-terminated-capacity-oversubscribed", "instance-terminated-launch-group-constraint":
		return true
	}

	return false
}
