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
	"strconv"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"

	zapadapter "logur.dev/adapter/zap"

	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon/autoscaling"
	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

const CreateAsgActivityName = "eks-create-asg"

// CreateAsgActivity responsible for creating IAM roles
type CreateAsgActivity struct {
	awsSessionFactory *AWSSessionFactory
	// body of the cloud formation template for setting up the VPC
	cloudFormationTemplate     string
	asgFulfillmentWaitAttempts int
	asgFulfillmentWaitInterval time.Duration
}

// CreateAsgActivityInput holds data needed for setting up IAM roles
type CreateAsgActivityInput struct {
	EKSActivityInput

	// name of the cloud formation template stack
	StackName string

	ScaleEnabled bool
	SSHKeyName   string

	Name             string
	NodeSpotPrice    string
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeImage        string
	NodeInstanceType string
	Labels           map[string]string

	Subnets             []Subnet
	VpcID               string
	SecurityGroupID     string
	NodeSecurityGroupID string
	NodeInstanceRoleID  string
}

// CreateAsgActivityOutput holds the output data of the CreateAsgActivityOutput
type CreateAsgActivityOutput struct {
}

// CreateAsgActivity instantiates a new CreateAsgActivity
func NewCreateAsgActivity(awsSessionFactory *AWSSessionFactory, cloudFormationTemplate string, asgFulfillmentWaitAttempts int, asgFulfillmentWaitInterval time.Duration) *CreateAsgActivity {
	return &CreateAsgActivity{
		awsSessionFactory:          awsSessionFactory,
		cloudFormationTemplate:     cloudFormationTemplate,
		asgFulfillmentWaitAttempts: asgFulfillmentWaitAttempts,
		asgFulfillmentWaitInterval: asgFulfillmentWaitInterval,
	}
}

func (a *CreateAsgActivity) Execute(ctx context.Context, input CreateAsgActivityInput) (*CreateAsgActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"vpcID", input.VpcID,
		"secret", input.SecretID,
		"ami", input.NodeImage,
		"nodePool", input.Name,
	)

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	cloudformationClient := cloudformation.New(awsSession)

	logger.With("stackName", input.StackName).Info("creating stack")

	spotPriceParam := ""
	if p, err := strconv.ParseFloat(input.NodeSpotPrice, 64); err == nil && p > 0.0 {
		spotPriceParam = input.NodeSpotPrice
	}

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
	var stackParams []*cloudformation.Parameter

	// do not update node labels via kubelet boostrap params as that induces node reboot or replacement
	// we only add node pool name here, all other labels will be added by NodePoolLabelSet operator
	nodeLabels := []string{
		fmt.Sprintf("%v=%v", common.LabelKey, input.Name),
	}

	var subnetIDs []string

	for _, subnet := range input.Subnets {
		subnetIDs = append(subnetIDs, subnet.SubnetID)
	}

	logger.With("subnets", subnetIDs).Info("node pool mapped to subnets")

	stackParams = []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("KeyName"),
			ParameterValue: aws.String(input.SSHKeyName),
		},
		{
			ParameterKey:   aws.String("NodeImageId"),
			ParameterValue: aws.String(input.NodeImage),
		},
		{
			ParameterKey:   aws.String("NodeInstanceType"),
			ParameterValue: aws.String(input.NodeInstanceType),
		},
		{
			ParameterKey:   aws.String("NodeSpotPrice"),
			ParameterValue: aws.String(spotPriceParam),
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
			ParameterKey:   aws.String("ClusterName"),
			ParameterValue: aws.String(input.ClusterName),
		},
		{
			ParameterKey:   aws.String("NodeGroupName"),
			ParameterValue: aws.String(input.Name),
		},
		{
			ParameterKey:   aws.String("ClusterControlPlaneSecurityGroup"),
			ParameterValue: aws.String(input.SecurityGroupID),
		},
		{
			ParameterKey:   aws.String("NodeSecurityGroup"),
			ParameterValue: aws.String(input.NodeSecurityGroupID),
		},
		{
			ParameterKey:   aws.String("VpcId"),
			ParameterValue: aws.String(input.VpcID),
		}, {
			ParameterKey:   aws.String("Subnets"),
			ParameterValue: aws.String(strings.Join(subnetIDs, ",")),
		},
		{
			ParameterKey:   aws.String("NodeInstanceRoleId"),
			ParameterValue: aws.String(input.NodeInstanceRoleID),
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
			ParameterKey:   aws.String("BootstrapArguments"),
			ParameterValue: aws.String(fmt.Sprintf("--kubelet-extra-args '--node-labels %v'", strings.Join(nodeLabels, ","))),
		},
	}
	clientRequestToken := generateRequestToken(input.AWSClientRequestTokenBase, CreateAsgActivityName)

	createStackInput := &cloudformation.CreateStackInput{
		ClientRequestToken: aws.String(clientRequestToken),
		DisableRollback:    aws.Bool(true),
		StackName:          aws.String(input.StackName),
		Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		Parameters:         stackParams,
		Tags:               tags,
		TemplateBody:       aws.String(a.cloudFormationTemplate),
		TimeoutInMinutes:   aws.Int64(10),
	}
	_, err = cloudformationClient.CreateStack(createStackInput)
	if err != nil {
		return nil, errors.WrapIff(err, "could not create '%s' CF stack", input.StackName)
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(input.StackName)}

	err = errors.WrapIff(cloudformationClient.WaitUntilStackCreateComplete(describeStacksInput),
		"waiting for %q CF stack create operation to complete failed", input.StackName)
	err = pkgCloudformation.NewAwsStackFailure(err, input.StackName, clientRequestToken, cloudformationClient)
	if err != nil {
		return nil, err
	}

	// wait for ASG fulfillment
	err = a.waitForASGToBeFulfilled(ctx, logger, awsSession, input.StackName, input.Name)
	if err != nil {
		return nil, errors.WrapIff(err, "node pool %q ASG not fulfilled", input.Name)
	}

	outParams := CreateAsgActivityOutput{}
	return &outParams, nil
}

// WaitForASGToBeFulfilled waits until an ASG has the desired amount of healthy nodes
func (a *CreateAsgActivity) waitForASGToBeFulfilled(
	ctx context.Context,
	logger *zap.SugaredLogger,
	awsSession *session.Session,
	stackName string,
	nodePoolName string) error {

	logger = logger.With("stackName", stackName)
	logger.Info("wait for ASG to be fulfilled")

	m := autoscaling.NewManager(awsSession, autoscaling.MetricsEnabled(true), autoscaling.Logger{
		Logger: zapadapter.New(logger.Desugar()),
	})

	ticker := time.NewTicker(a.asgFulfillmentWaitInterval)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-ticker.C:
			if i <= a.asgFulfillmentWaitAttempts {
				i++

				asGroup, err := m.GetAutoscalingGroupByStackName(stackName)
				if err != nil {
					if aerr, ok := err.(awserr.Error); ok {
						if aerr.Code() == "ValidationError" || aerr.Code() == "ASGNotFoundInResponse" {
							continue
						}
					}
					return errors.WrapIfWithDetails(err, "could not get ASG", "stackName", stackName)
				}

				ok, err := asGroup.IsHealthy()
				if err != nil {
					if autoscaling.IsErrorFinal(err) {
						return errors.WithDetails(err, "nodePoolName", nodePoolName, "stackName", aws.StringValue(asGroup.AutoScalingGroupName))
					}
					//log.Debug(err)
					continue
				}
				if ok {
					//log.Debug("ASG is healthy")
					return nil
				}
			} else {
				return errors.Errorf("waiting for ASG to be fulfilled timed out after %d x %s", a.asgFulfillmentWaitAttempts, a.asgFulfillmentWaitInterval)
			}
		case <-ctx.Done(): // wait for ASG fulfillment cancelled
			return nil
		}
	}

}
