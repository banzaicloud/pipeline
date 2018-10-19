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

package action

import (
	"time"

	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon/autoscaling"
)

// WaitForHealthyAutoscalingGroupsAction
type WaitForHealthyAutoscalingGroupsAction struct {
	context   *EksClusterCreateUpdateContext
	nodePools []*model.AmazonNodePoolsModel
	log       logrus.FieldLogger
	attempts  int
	interval  time.Duration
}

// NewWaitForHealthyAutoscalingGroupsAction creates a new WaitForHealthyAutoscalingGroupsAction
func NewWaitForHealthyAutoscalingGroupsAction(
	log logrus.FieldLogger,
	attempts int,
	interval time.Duration,
	creationContext *EksClusterCreateUpdateContext,
	nodePools ...*model.AmazonNodePoolsModel) *WaitForHealthyAutoscalingGroupsAction {
	return &WaitForHealthyAutoscalingGroupsAction{
		context:   creationContext,
		nodePools: nodePools,
		log:       log,
		attempts:  attempts,
		interval:  interval,
	}
}

// GetName returns the name of this WaitForHealthyAutoscalingGroupsAction
func (a *WaitForHealthyAutoscalingGroupsAction) GetName() string {
	return "WaitForHealthyAutoscalingGroupsAction"
}

// ExecuteAction executes the WaitForHealthyAutoscalingGroupsAction
func (a *WaitForHealthyAutoscalingGroupsAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	m := autoscaling.NewManager(a.context.Session)

	for i := 0; i <= a.attempts; i++ {
		nodePoolOk := 0
		for _, nodePool := range a.nodePools {
			asGroup, err := m.GetAutoscalingGroupByStackName(a.generateStackName(nodePool))
			if err != nil {
				return nil, err
			}
			a.log.WithField("asg-name", *asGroup.AutoScalingGroupName).Debug("checking ASG")

			ok, err := asGroup.IsHealthy()
			if err != nil {
				if autoscaling.IsErrorFinal(err) {
					return nil, emperror.WrapWith(err, nodePool.Name, "nodePoolName", nodePool.Name, "asgName", *asGroup.AutoScalingGroupName)
				}
				a.log.WithField("asg-name", *asGroup.AutoScalingGroupName).Debug(err)
			}
			if ok {
				nodePoolOk++
			}
		}

		if nodePoolOk == len(a.nodePools) {
			a.log.Debug("all ASGs are fine")
			return nil, nil
		}

		time.Sleep(a.interval)
	}

	return nil, errors.New("could not get ASGs into healthy state for a long time")
}

func (a *WaitForHealthyAutoscalingGroupsAction) generateStackName(nodePool *model.AmazonNodePoolsModel) string {
	return GenerateNodePoolStackName(a.context.ClusterName, nodePool.Name)
}
