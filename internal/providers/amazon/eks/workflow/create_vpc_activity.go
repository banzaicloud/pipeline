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

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"

	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

const CreateVpcActivityName = "eks-create-vpc"

// CreateVpcActivity responsible for setting up a VPC for an EKS cluster
type CreateVpcActivity struct {
	awsSessionFactory *AWSSessionFactory
}

// CreateVpcActivityInput holds data needed for setting up
// VPC for EKS cluster
type CreateVpcActivityInput struct {
	EKSActivityInput

	// body of the cloud formation template for setting up the VPC
	CloudFormationTemplate string

	// name of the cloud formation template stack
	StackName string

	// the ID of the VPC to be used instead of creating a new one
	VpcID string

	// the ID of the Route Table to be used with the existing VPC
	RouteTableID string

	// the CIDR to create new VPC with
	VpcCidr string
}

// CreateVpcActivityOutput holds the output data of the CreateVpcActivity
type CreateVpcActivityOutput struct {
	VpcID               string
	RouteTableID        string
	SecurityGroupID     string
	NodeSecurityGroupID string
}

// NewCreateVPCActivity instantiates a new CreateVpcActivity
func NewCreateVPCActivity(awsSessionFactory *AWSSessionFactory) *CreateVpcActivity {
	return &CreateVpcActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *CreateVpcActivity) Execute(ctx context.Context, input CreateVpcActivityInput) (*CreateVpcActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"vpcID", input.VpcID,
		"vpcCidr", input.VpcCidr,
		"routeTableID", input.RouteTableID,
		"secret", input.SecretID,
	)

	stackParams := []*cloudformation.Parameter{
		{
			ParameterKey:   aws.String("ClusterName"),
			ParameterValue: aws.String(input.ClusterName),
		},
	}

	if input.VpcID != "" {
		logger.Info("skip creating new VPC, as an existing VPC ID is provided")

		stackParams = append(stackParams,
			&cloudformation.Parameter{
				ParameterKey:   aws.String("VpcId"),
				ParameterValue: aws.String(input.VpcID),
			})

		if input.RouteTableID != "" {
			stackParams = append(stackParams,
				&cloudformation.Parameter{
					ParameterKey:   aws.String("RouteTableId"),
					ParameterValue: aws.String(input.RouteTableID),
				})
		}

	} else if input.VpcCidr != "" {
		stackParams = append(stackParams,
			&cloudformation.Parameter{
				ParameterKey:   aws.String("VpcBlock"),
				ParameterValue: aws.String(input.VpcCidr),
			})
	}

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}
	cloudformationClient := cloudformation.New(session)

	clientRequestToken := generateRequestToken(input.AWSClientRequestTokenBase, CreateVpcActivityName)
	createStackInput := &cloudformation.CreateStackInput{
		ClientRequestToken: aws.String(clientRequestToken),
		DisableRollback:    aws.Bool(true),
		StackName:          aws.String(input.StackName),
		Parameters:         stackParams,
		Tags:               getVPCStackTags(input.ClusterName),
		TemplateBody:       aws.String(input.CloudFormationTemplate),
		TimeoutInMinutes:   aws.Int64(10),
	}
	_, err = cloudformationClient.CreateStack(createStackInput)
	if err != nil {
		return nil, errors.WrapIf(err, "create stack failed")
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(input.StackName)}
	err = cloudformationClient.WaitUntilStackCreateComplete(describeStacksInput)
	if err != nil {
		return nil, pkgCloudformation.NewAwsStackFailure(err, input.StackName, clientRequestToken, cloudformationClient)
	}

	describeStacksOutput, err := cloudformationClient.DescribeStacks(describeStacksInput)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get stack output parameters")
	}

	outParams := CreateVpcActivityOutput{}
	for _, output := range describeStacksOutput.Stacks[0].Outputs {
		switch aws.StringValue(output.OutputKey) {
		case "VpcId":
			outParams.VpcID = aws.StringValue(output.OutputValue)
		case "RouteTableId":
			outParams.RouteTableID = aws.StringValue(output.OutputValue)
		case "SecurityGroups":
			outParams.SecurityGroupID = aws.StringValue(output.OutputValue)
		case "NodeSecurityGroup":
			outParams.NodeSecurityGroupID = aws.StringValue(output.OutputValue)
		}
	}

	return &outParams, nil
}

func getVPCStackTags(clusterName string) []*cloudformation.Tag {
	return getStackTags(clusterName, "vpc")
}
