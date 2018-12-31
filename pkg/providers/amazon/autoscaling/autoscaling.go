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
	"github.com/aws/aws-sdk-go/service/autoscaling"
	awsEC2 "github.com/aws/aws-sdk-go/service/ec2"

	"github.com/banzaicloud/pipeline/pkg/providers/amazon/ec2"
)

// Group extends autoscaling.Group
type Group struct {
	*autoscaling.Group
	manager *Manager
}

// NewGroup initialises and gives back a Group
func NewGroup(manager *Manager, group *autoscaling.Group) *Group {
	return &Group{
		Group:   group,
		manager: manager,
	}
}

// IsHealthy checks whether an ASG is in a healthy state
// which means it has as many healthy instances as desired
func (group *Group) IsHealthy() (bool, error) {
	ok := 0

	instances := group.getInstances()
	for _, instance := range instances {
		if instance.IsHealthyAndInService() {
			if group.manager.StopMetricTimer(instance) {
				group.manager.RegisterSpotFulfillmentDuration(instance, group)
			}
			ok++
		}
		if instance.LifecycleState != nil && *instance.LifecycleState == "Pending" {
			group.manager.StartMetricTimer(instance)
		}
	}

	if group.DesiredCapacity != nil && int(*group.DesiredCapacity) > 0 && ok == int(*group.DesiredCapacity) {
		return true, nil
	}

	spotRequests, err := group.getSpotRequests()
	if err != nil {
		return false, err
	}

	if len(spotRequests) == 0 {
		return false, NewAutoscalingGroupNotHealthyError(int(*group.DesiredCapacity), ok)
	}

	for _, spotRequest := range spotRequests {
		if spotRequest.IsActive() && !spotRequest.IsPending() && !spotRequest.IsFulfilled() {
			return false, ec2.NewSpotRequestFailedError(spotRequest.GetStatusCode())
		}
	}

	return false, NewAutoscalingGroupNotHealthyError(int(*group.DesiredCapacity), ok)
}

func (group *Group) getInstances() []*Instance {
	instances := make([]*Instance, 0)

	for _, inst := range group.Instances {
		instances = append(instances, NewInstance(group.manager, inst))
	}

	return instances
}

func (group *Group) getSpotRequests() ([]*ec2.SpotInstanceRequest, error) {
	input := &awsEC2.DescribeSpotInstanceRequestsInput{}
	result, err := group.manager.ec2Svc.DescribeSpotInstanceRequests(input)
	if err != nil {
		return nil, err
	}

	lc, err := group.getLaunchConfiguration()
	if err != nil {
		return nil, err
	}

	if lc.SpotPrice != nil && *lc.SpotPrice == "" {
		return nil, nil
	}

	requests := make([]*ec2.SpotInstanceRequest, 0)
	for _, res := range result.SpotInstanceRequests {
		if res.LaunchSpecification == nil || res.LaunchSpecification.IamInstanceProfile == nil || res.LaunchSpecification.IamInstanceProfile.Name == nil || lc.IamInstanceProfile == nil {
			continue
		}
		// !!! We must use unique instance profile for every ASG for this to work !!!
		if *res.LaunchSpecification.IamInstanceProfile.Name == *lc.IamInstanceProfile {
			requests = append(requests, ec2.NewSpotInstanceRequest(res))
		}
	}

	return requests, nil
}

func (group *Group) getLaunchConfiguration() (*autoscaling.LaunchConfiguration, error) {
	input := &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{
			group.LaunchConfigurationName,
		},
	}

	result, err := group.manager.asSvc.DescribeLaunchConfigurations(input)
	if err != nil {
		return nil, err
	}

	if len(result.LaunchConfigurations) > 0 {
		return result.LaunchConfigurations[0], nil
	}

	return nil, nil
}
