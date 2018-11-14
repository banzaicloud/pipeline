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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Instance extends autoscaling.Instance
type Instance struct {
	*autoscaling.Instance
	manager *Manager
}

// NewInstance initialises and gives back a new Instance
func NewInstance(manager *Manager, instance *autoscaling.Instance) *Instance {
	return &Instance{
		Instance: instance,
		manager:  manager,
	}
}

// IsHealthyAndInService is true if the instance is healthy and in InService state
func (i *Instance) IsHealthyAndInService() bool {
	return *i.HealthStatus == "Healthy" && *i.LifecycleState == "InService"
}

// Describe returns detailed information about the instance
func (i *Instance) Describe() (*ec2.Instance, error) {
	result, err := i.manager.ec2Svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			i.InstanceId,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(result.Reservations) == 1 {
		reservation := result.Reservations[0]
		if len(reservation.Instances) == 1 {
			return reservation.Instances[0], nil
		}
	}

	return nil, awserr.New("404", "instance not found", nil)
}
