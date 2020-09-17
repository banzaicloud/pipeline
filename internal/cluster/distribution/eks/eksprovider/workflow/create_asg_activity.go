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

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
)

const CreateAsgActivityName = "eks-create-asg"

// CreateAsgActivity responsible for creating IAM roles
type CreateAsgActivity struct {
	awsSessionFactory AWSFactory
	// body of the cloud formation template for setting up the VPC
	cloudFormationTemplate string
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
	NodeVolumeSize   int
	NodeImage        string
	NodeInstanceType string
	Labels           map[string]string

	Subnets             []Subnet
	VpcID               string
	SecurityGroupID     string
	NodeSecurityGroupID string
	NodeInstanceRoleID  string
	Tags                map[string]string
}

// CreateAsgActivityOutput holds the output data of the CreateAsgActivityOutput
type CreateAsgActivityOutput struct {
}

// CreateAsgActivity instantiates a new CreateAsgActivity
func NewCreateAsgActivity(awsSessionFactory AWSFactory, cloudFormationTemplate string) *CreateAsgActivity {
	return &CreateAsgActivity{
		awsSessionFactory:      awsSessionFactory,
		cloudFormationTemplate: cloudFormationTemplate,
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

	tags := getNodePoolStackTags(input.ClusterName, input.Tags)
	var stackParams []*cloudformation.Parameter

	// do not update node labels via kubelet boostrap params as that induces node reboot or replacement
	// we only add node pool name here, all other labels will be added by NodePoolLabelSet operator
	nodeLabels := []string{
		fmt.Sprintf("%v=%v", cluster.NodePoolNameLabelKey, input.Name),
		fmt.Sprintf("%v=%v", cluster.NodePoolVersionLabelKey, eks.CalculateNodePoolVersion(input.NodeImage)),
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
			ParameterKey:   aws.String("NodeVolumeSize"),
			ParameterValue: aws.String(fmt.Sprintf("%d", input.NodeVolumeSize)),
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
		},
		{
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
	clientRequestToken := sdkAmazon.NewNormalizedClientRequestToken(input.AWSClientRequestTokenBase, CreateAsgActivityName)

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
	err = WaitUntilStackCreateCompleteWithContext(cloudformationClient, ctx, describeStacksInput)
	if err != nil {
		return nil, packageCFError(err, input.StackName, clientRequestToken, cloudformationClient, "waiting for CF stack create operation to complete failed")
	}

	// wait for ASG fulfillment
	err = WaitForASGToBeFulfilled(ctx, logger, awsSession, input.StackName, input.Name)
	if err != nil {
		return nil, errors.WrapIff(err, "node pool %q ASG not fulfilled", input.Name)
	}

	outParams := CreateAsgActivityOutput{}
	return &outParams, nil
}
