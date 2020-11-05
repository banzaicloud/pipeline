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

package pkeawsworkflow

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/pkg/backoff"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

const (
	HealthCheckActivityName = "nlb-health-check"
	backoffDelay            = 3 * time.Second
	backoffMaxRetries       = 3
)

type HealthCheckActivity struct {
	awsSessionFactory AWSSessionFactory
	elbv2Factory      ELBV2APIFactory
}

// NewHealthCheckActivity returns a new HealthCheckActivity.
func NewHealthCheckActivity(
	awsSessionFactory AWSSessionFactory,
	elbv2Factory ELBV2APIFactory,
) *HealthCheckActivity {
	return &HealthCheckActivity{
		awsSessionFactory: awsSessionFactory,
		elbv2Factory:      elbv2Factory,
	}
}

type HealthCheckActivityInput struct {
	OrganizationID uint
	SecretID       string
	Region         string
	ClusterID      uint
	ClusterName    string
}

// HealthCheckActivityOutput holds the output data
type HealthCheckActivityOutput struct {
	nlbStatus []*elbv2.TargetHealthDescription
}

// Register registers the activity in the worker.
func (a *HealthCheckActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: HealthCheckActivityName})
}

func (a *HealthCheckActivity) Execute(ctx context.Context, input HealthCheckActivityInput) (*HealthCheckActivityOutput, error) {
	if a == nil {
		return nil, errors.New("activity is nil")
	}
	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create AWS session")
	}

	elbv2Svc := a.elbv2Factory.New(session)
	nlbName := "pke-" + input.ClusterName + "-nlb"

	backoffConfig := backoff.ConstantBackoffConfig{
		Delay:      backoffDelay,
		MaxRetries: backoffMaxRetries,
	}
	backoffPolicy := backoff.NewConstantBackoffPolicy(backoffConfig)

	var loadBalancerOutput *elbv2.DescribeLoadBalancersOutput
	describeLoadBalancers := func() error {
		var err error
		loadBalancerOutput, err = elbv2Svc.DescribeLoadBalancers(
			&elbv2.DescribeLoadBalancersInput{
				Names: []*string{&nlbName},
			})
		if err != nil {
			return errors.WrapIf(err, "failed to describe load balancers")
		}

		return nil
	}

	if err := backoff.Retry(describeLoadBalancers, backoffPolicy); err != nil {
		return nil, err
	}

	var loadBalancerArn *string
	if len(loadBalancerOutput.LoadBalancers) > 0 {
		loadBalancerArn = loadBalancerOutput.LoadBalancers[0].LoadBalancerArn
	} else {
		return nil, errors.New("load balancers output doesn't contain load balancer")
	}

	var targetGroupOutput *elbv2.DescribeTargetGroupsOutput
	describeTargetGroups := func() error {
		var err error
		targetGroupOutput, err = elbv2Svc.DescribeTargetGroups(
			&elbv2.DescribeTargetGroupsInput{
				LoadBalancerArn: loadBalancerArn,
			})
		if err != nil {
			return errors.WrapIf(err, "failed to describe target groups")
		}

		return nil
	}

	if err := backoff.Retry(describeTargetGroups, backoffPolicy); err != nil {
		return nil, err
	}

	var targetGroupArn *string
	if len(targetGroupOutput.TargetGroups) > 0 {
		targetGroupArn = targetGroupOutput.TargetGroups[0].TargetGroupArn
	} else {
		return nil, errors.New("target groups output doesn't conatain target group")
	}

	var targetHealthOutput *elbv2.DescribeTargetHealthOutput
	describeTargetHealth := func() error {
		var err error
		targetHealthOutput, err = elbv2Svc.DescribeTargetHealth(
			&elbv2.DescribeTargetHealthInput{
				TargetGroupArn: targetGroupArn,
			})
		if err != nil {
			return errors.WrapIf(err, "failed to describe target health")
		}

		return nil
	}

	if err := backoff.Retry(describeTargetHealth, backoffPolicy); err != nil {
		return nil, err
	}

	for _, tgHealthDesc := range targetHealthOutput.TargetHealthDescriptions {
		if tgHealthDesc.TargetHealth == nil || tgHealthDesc.TargetHealth.State == nil {
			return nil, errors.NewWithDetails(
				"unable to getting state of the target health description",
				"targetHealthDescriptions", targetHealthOutput.TargetHealthDescriptions,
			)
		}
		if *tgHealthDesc.TargetHealth.State != elbv2.TargetHealthStateEnumHealthy {
			return nil, errors.NewWithDetails(
				"not all targets of the network load balancer are healthy",
				"targetHealthDescriptions", targetHealthOutput.TargetHealthDescriptions,
			)
		}
	}

	return &HealthCheckActivityOutput{nlbStatus: targetHealthOutput.TargetHealthDescriptions}, nil
}
