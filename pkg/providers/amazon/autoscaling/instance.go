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
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"

	pkgEC2 "github.com/banzaicloud/pipeline/pkg/providers/amazon/ec2"
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
	var healthStatus, lifecycleState string

	if i.HealthStatus != nil {
		healthStatus = *i.HealthStatus
	}

	if i.LifecycleState != nil {
		lifecycleState = *i.LifecycleState
	}

	return healthStatus == "Healthy" && lifecycleState == "InService"
}

// Describe returns detailed information about the instance
func (i *Instance) Describe() (*ec2.Instance, error) {
	if i.InstanceId == nil {
		return nil, errors.New("instance id is nil")
	}

	return pkgEC2.DescribeInstanceById(i.manager.ec2Svc, *i.InstanceId)
}
