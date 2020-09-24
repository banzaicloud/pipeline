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

package pkeworkflow

import (
	"context"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"

	cloudformation2 "github.com/banzaicloud/pipeline/internal/cloudformation"
	eksWorkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
)

const CreateSubnetActivityName = "pke-create-subnet"

const SubnetCloudFormationTemplate = "subnet.cf.yaml"

func NewCreateSubnetActivity(awsClientFactory *AWSClientFactory) *CreateSubnetActivity {
	return &CreateSubnetActivity{
		awsClientFactory: awsClientFactory,
	}
}

// CreateSubnetActivity responsible for setting up a Subnet for an EKS cluster
type CreateSubnetActivity struct {
	awsClientFactory *AWSClientFactory
}

// CreateSubnetActivityInput holds data needed for setting up
// a Subnet for EKS cluster
type CreateSubnetActivityInput struct {
	AWSActivityInput
	ClusterID   uint
	ClusterName string
	// the ID of the VPC to create the subnet into
	VpcID string

	// the ID of the Route Table to associate the Subnet with
	RouteTableID string

	// The CIDR of the subnet
	Cidr string

	// The availability zone of the subnet
	AvailabilityZone string
}

// CreateSubnetActivityOutput holds the output data of the CreateSubnetActivity
type CreateSubnetActivityOutput struct {
	SubnetID         string
	Cidr             string
	AvailabilityZone string
}

func (a *CreateSubnetActivity) Execute(ctx context.Context, input CreateSubnetActivityInput) (*CreateSubnetActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"vpcID", input.VpcID,
		"routeTableID", input.RouteTableID,
		"subnetCidr", input.Cidr,
		"availabilityZone", input.AvailabilityZone,
		"secret", input.SecretID,
	)

	client, err := a.awsClientFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	template, err := cloudformation2.GetCloudFormationTemplate(PKECloudFormationTemplateBasePath, SubnetCloudFormationTemplate)
	if err != nil {
		return nil, errors.WrapIf(err, "loading CF template")
	}

	stackName := aws.String("pke-subnet-" + input.ClusterName + "-" + input.AvailabilityZone)

	if input.Cidr != "" {
		logger.Debug("creating subnet")

		cloudformationClient := cloudformation.New(client)

		stackParams := []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("VpcId"),
				ParameterValue: aws.String(input.VpcID),
			},
			{
				ParameterKey:   aws.String("RouteTableId"),
				ParameterValue: aws.String(input.RouteTableID),
			},
			{
				ParameterKey:   aws.String("SubnetBlock"),
				ParameterValue: aws.String(input.Cidr),
			},
			{
				ParameterKey:   aws.String("AvailabilityZoneName"),
				ParameterValue: aws.String(input.AvailabilityZone),
			},
		}
		createStackInput := &cloudformation.CreateStackInput{
			DisableRollback:  aws.Bool(true),
			StackName:        stackName,
			Parameters:       stackParams,
			Tags:             getSubnetStackTags(input.ClusterName),
			TemplateBody:     aws.String(template),
			TimeoutInMinutes: aws.Int64(10),
		}

		_, err := cloudformationClient.CreateStack(createStackInput)
		if err != nil {
			return nil, errors.WrapIf(err, "create stack failed")
		}

		describeStacksInput := &cloudformation.DescribeStacksInput{StackName: stackName}
		err = eksWorkflow.WaitUntilStackCreateCompleteWithContext(cloudformationClient, ctx, describeStacksInput)

		if err != nil {
			return nil, err
		}

		describeStacksOutput, err := cloudformationClient.DescribeStacks(describeStacksInput)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to get subnet ID from stack output parameters")
		}

		var subnetId string
		for _, output := range describeStacksOutput.Stacks[0].Outputs {
			switch aws.StringValue(output.OutputKey) {
			case "SubnetId":
				subnetId = aws.StringValue(output.OutputValue)
			}
		}

		logger.Debug("subnet successfully created")

		return &CreateSubnetActivityOutput{
			SubnetID:         subnetId,
			Cidr:             input.Cidr,
			AvailabilityZone: input.AvailabilityZone,
		}, nil
	}

	return nil, nil
}
