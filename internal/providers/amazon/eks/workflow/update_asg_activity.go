// Copyright © 2019 Banzai Cloud
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

package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/prometheus/common/log"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"

	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

const UpdateAsgActivityName = "eks-update-asg"

const awsNoUpdatesError = "No updates are to be performed."

// UpdateAsgActivity responsible for creating IAM roles
type UpdateAsgActivity struct {
	awsSessionFactory *AWSSessionFactory
	// body of the cloud formation template for setting up the VPC
	cloudFormationTemplate     string
	asgFulfillmentWaitAttempts int
	asgFulfillmentWaitInterval time.Duration
}

// UpdateAsgActivityInput holds data needed for setting up IAM roles
type UpdateAsgActivityInput struct {
	EKSActivityInput

	// name of the cloud formation template stack
	StackName        string
	ScaleEnabled     bool
	Name             string
	NodeSpotPrice    string
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeImage        string
	NodeInstanceType string
	Labels           map[string]string
}

// UpdateAsgActivityOutput holds the output data of the UpdateAsgActivityOutput
type UpdateAsgActivityOutput struct {
}

// UpdateAsgActivity instantiates a new UpdateAsgActivity
func NewUpdateAsgActivity(awsSessionFactory *AWSSessionFactory, cloudFormationTemplate string, asgFulfillmentWaitAttempts int, asgFulfillmentWaitInterval time.Duration) *UpdateAsgActivity {
	return &UpdateAsgActivity{
		awsSessionFactory:          awsSessionFactory,
		cloudFormationTemplate:     cloudFormationTemplate,
		asgFulfillmentWaitAttempts: asgFulfillmentWaitAttempts,
		asgFulfillmentWaitInterval: asgFulfillmentWaitInterval,
	}
}

func (a *UpdateAsgActivity) Execute(ctx context.Context, input UpdateAsgActivityInput) (*UpdateAsgActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"ami", input.NodeImage,
		"nodePool", input.Name,
	)

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	cloudformationClient := cloudformation.New(awsSession)

	if input.Autoscaling {
		autoscalingSrv := autoscaling.New(awsSession)
		// get current Desired count from ASG linked to nodeGroup stack if Autoscaling is enabled,
		// as we don't to override in this case only min/max counts
		asg, err := getAutoScalingGroup(cloudformationClient, autoscalingSrv, input.StackName)
		if err != nil {
			return nil, errors.WrapIff(err, "unable to find ASG for node pool %q", input.Name)
		}

		// override nodePool.Count with current DesiredCapacity in case of autoscale, as we don't want allow direct
		// setting of DesiredCapacity via API, however we have to limit it to be between new min, max values.
		if asg != nil {
			if asg.DesiredCapacity != nil {
				input.Count = int(*asg.DesiredCapacity)
			}
			if input.Count < input.NodeMinCount {
				input.Count = input.NodeMinCount
			} else if input.Count > input.NodeMaxCount {
				input.Count = input.NodeMaxCount
			}
			log.Infof("DesiredCapacity for %v will be: %v", aws.StringValue(asg.AutoScalingGroupARN), input.Count)
		}

	}

	logger.With("stackName", input.StackName).Info("updating stack")

	// update stack
	clusterAutoscalerEnabled := false
	terminationDetachEnabled := false

	if input.Autoscaling {
		clusterAutoscalerEnabled = true
	}

	// if ScaleOptions is enabled on cluster, ClusterAutoscaler is disabled on all node pools
	if input.ScaleEnabled {
		clusterAutoscalerEnabled = false
		terminationDetachEnabled = true
	}

	tags := getNodePoolStackTags(input.ClusterName)

	stackParams := []*cloudformation.Parameter{
		{
			ParameterKey:     aws.String("KeyName"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeImageId"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeInstanceType"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeSpotPrice"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
			ParameterValue: aws.String(fmt.Sprintf("%d", input.NodeMinCount)),
		},
		{
			ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
			ParameterValue: aws.String(fmt.Sprintf("%d", input.NodeMaxCount)),
		},
		{
			ParameterKey:   aws.String("NodeAutoScalingInitSize"),
			ParameterValue: aws.String(fmt.Sprintf("%d", input.Count)),
		},
		{
			ParameterKey:     aws.String("ClusterName"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeGroupName"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("ClusterControlPlaneSecurityGroup"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeSecurityGroup"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("VpcId"),
			UsePreviousValue: aws.Bool(true),
		}, {
			ParameterKey:     aws.String("Subnets"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeInstanceRoleId"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:   aws.String("ClusterAutoscalerEnabled"),
			ParameterValue: aws.String(fmt.Sprint(clusterAutoscalerEnabled)),
		},
		{
			ParameterKey:   aws.String("TerminationDetachEnabled"),
			ParameterValue: aws.String(fmt.Sprint(terminationDetachEnabled)),
		},
		{
			ParameterKey:     aws.String("BootstrapArguments"),
			UsePreviousValue: aws.Bool(true),
		},
	}

	clientRequestToken := generateRequestToken(input.AWSClientRequestTokenBase, UpdateAsgActivityName)

	// we don't reuse the creation time template, since it may have changed
	updateStackInput := &cloudformation.UpdateStackInput{
		ClientRequestToken: aws.String(clientRequestToken),
		StackName:          aws.String(input.StackName),
		Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		Parameters:         stackParams,
		Tags:               tags,
		TemplateBody:       aws.String(a.cloudFormationTemplate),
	}

	_, err = cloudformationClient.UpdateStack(updateStackInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ValidationError" && strings.HasPrefix(awsErr.Message(), awsNoUpdatesError) {
			// Get error details
			log.Warnf("nothing changed during update!")
			err = nil // nolint: ineffassign
		} else {
			var awsErr awserr.Error
			if errors.As(err, &awsErr) {
				if awsErr.Code() == request.WaiterResourceNotReadyErrorCode {
					err = pkgCloudformation.NewAwsStackFailure(err, input.StackName, clientRequestToken, cloudformationClient)
					err = errors.WrapIff(err, "waiting for %q CF stack create operation to complete failed", input.StackName)
					if pkgCloudformation.IsErrorFinal(err) {
						return nil, cadence.NewCustomError(ErrReasonStackFailed, err.Error())
					}
					return nil, errors.WrapIff(err, "waiting for %q CF stack create operation to complete failed", input.StackName)
				}
			}
		}
	}

	outParams := UpdateAsgActivityOutput{}
	return &outParams, nil
}
