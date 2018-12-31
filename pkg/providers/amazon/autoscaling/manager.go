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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Manager is for quering auto scaling groups
type Manager struct {
	metricsEnabled bool
	logger         logrus.FieldLogger
	session        *session.Session

	asSvc  *autoscaling.AutoScaling
	cfSvc  *cloudformation.CloudFormation
	ec2Svc *ec2.EC2
}

// NewManager initialises and gives back a new Manager
func NewManager(session *session.Session, opts ...Option) *Manager {
	m := &Manager{
		session: session,

		asSvc:  autoscaling.New(session),
		cfSvc:  cloudformation.New(session),
		ec2Svc: ec2.New(session),
	}

	for _, o := range opts {
		o.apply(m)
	}

	if m.logger == nil {
		m.logger = logrus.New()
	}

	return m
}

// GetAutoscalingGroups gets auto scaling groups and gives back as initialised []Group
func (m *Manager) GetAutoscalingGroups() ([]*Group, error) {
	input := &autoscaling.DescribeAutoScalingGroupsInput{}

	svc := autoscaling.New(m.session)
	result, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		return nil, err
	}

	groups := make([]*Group, 0)
	for _, group := range result.AutoScalingGroups {
		groups = append(groups, NewGroup(m, group))
	}

	return groups, nil
}

// GetAutoscalingGroupByID gets and auto scaling group by it's ID and gives back as an initialised Group
func (m *Manager) GetAutoscalingGroupByID(id string) (*Group, error) {
	input := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(id),
		},
	}

	result, err := m.asSvc.DescribeAutoScalingGroups(input)
	if err != nil {
		return nil, err
	}

	if len(result.AutoScalingGroups) == 1 {
		return NewGroup(m, result.AutoScalingGroups[0]), nil
	}

	return nil, emperror.With(errors.New("ASG not found"), "id", id)
}

// GetAutoscalingGroupByStackName gets and auto scaling group by the name of the stack which created it and gives back as an initialised Group
func (m *Manager) GetAutoscalingGroupByStackName(stackName string) (*Group, error) {
	logResourceId := "NodeGroup"

	describeStackResourceInput := &cloudformation.DescribeStackResourceInput{
		LogicalResourceId: &logResourceId,
		StackName:         aws.String(stackName)}
	describeStacksOutput, err := m.cfSvc.DescribeStackResource(describeStackResourceInput)
	if err != nil {
		return nil, err
	}

	if describeStacksOutput.StackResourceDetail == nil || describeStacksOutput.StackResourceDetail.PhysicalResourceId == nil {
		return nil, awserr.New("ValidationError", fmt.Sprintf("Stack '%s' doest not exist", stackName), nil)
	}

	describeAutoScalingGroupsInput := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			describeStacksOutput.StackResourceDetail.PhysicalResourceId,
		},
	}
	describeAutoScalingGroupsOutput, err := m.asSvc.DescribeAutoScalingGroups(&describeAutoScalingGroupsInput)
	if err != nil {
		return nil, err
	}

	if len(describeAutoScalingGroupsOutput.AutoScalingGroups) != 1 {
		return nil, awserr.New("ASGNotFoundInResponse", "could not find ASG in response", nil)
	}

	return NewGroup(m, describeAutoScalingGroupsOutput.AutoScalingGroups[0]), nil
}
