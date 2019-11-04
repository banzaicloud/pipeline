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

const CreateSubnetActivityName = "eks-create-subnet"

// CreateSubnetActivity responsible for setting up a Subnet for an EKS cluster
type CreateSubnetActivity struct {
	awsSessionFactory *AWSSessionFactory
}

// CreateSubnetActivityInput holds data needed for setting up
// a Subnet for EKS cluster
type CreateSubnetActivityInput struct {
	EKSActivityInput

	// the ID of the VPC to create the subnet into
	VpcID string

	// the ID of the Route Table to associate the Subnet with
	RouteTableID string

	// The AWS ID of the subnet
	SubnetID string

	// The CIDR of the subnet
	Cidr string

	// The availability zone of the subnet
	AvailabilityZone string

	// name of the cloud formation template stack
	StackName string

	// body of the cloud formation template for setting up the Subnet
	CloudFormationTemplate string
}

// CreateSubnetActivityOutput holds the output data of the CreateSubnetActivity
type CreateSubnetActivityOutput struct {
	SubnetID         string
	Cidr             string
	AvailabilityZone string
}

// NewCreateSubnetActivity instantiates a new CreateSubnetActivity
func NewCreateSubnetActivity(awsSessionFactory *AWSSessionFactory) *CreateSubnetActivity {
	return &CreateSubnetActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *CreateSubnetActivity) Execute(ctx context.Context, input CreateSubnetActivityInput) (*CreateSubnetActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"vpcID", input.VpcID,
		"routeTableID", input.RouteTableID,
		"subnetID", input.SubnetID,
		"subnetCidr", input.Cidr,
		"availabilityZone", input.AvailabilityZone,
		"secret", input.SecretID,
	)

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	if input.SubnetID == "" && input.Cidr != "" {
		logger.Debug("creating subnet")

		cloudformationClient := cloudformation.New(session)

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

		clientRequestToken := generateRequestToken(input.AWSClientRequestTokenBase, CreateSubnetActivityName)
		createStackInput := &cloudformation.CreateStackInput{
			ClientRequestToken: aws.String(input.AWSClientRequestTokenBase),
			DisableRollback:    aws.Bool(true),
			StackName:          aws.String(input.StackName),
			Parameters:         stackParams,
			Tags:               getSubnetStackTags(input.ClusterName),
			TemplateBody:       aws.String(input.CloudFormationTemplate),
			TimeoutInMinutes:   aws.Int64(10),
		}

		_, err := cloudformationClient.CreateStack(createStackInput)
		if err != nil {
			return nil, errors.WrapIf(err, "create stack failed")
		}

		describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(input.StackName)}
		err = cloudformationClient.WaitUntilStackCreateComplete(describeStacksInput)
		if err != nil {
			return nil, errors.WrapIff(pkgCloudformation.NewAwsStackFailure(err, input.StackName, clientRequestToken, cloudformationClient), "failed to create subnet with cidr %q", input.Cidr)
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

func getSubnetStackTags(clusterName string) []*cloudformation.Tag {
	return getStackTags(clusterName, "subnet")
}
