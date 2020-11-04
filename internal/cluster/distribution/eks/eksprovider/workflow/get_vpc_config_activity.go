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

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

const GetVpcConfigActivityName = "eks-get-vpc-cfg"

// GetVpcConfigActivity responsible for creating IAM roles
type GetVpcConfigActivity struct {
	awsSessionFactory awsworkflow.AWSFactory
}

// GetVpcConfigActivityInput holds data needed for setting up IAM roles
type GetVpcConfigActivityInput struct {
	EKSActivityInput

	// name of the cloud formation template stack
	StackName string
}

// GetVpcConfigActivityOutput holds the output data of the GetVpcConfigActivityOutput
type GetVpcConfigActivityOutput struct {
	VpcID               string
	SecurityGroupID     string
	NodeSecurityGroupID string
}

// GetVpcConfigActivity instantiates a new GetVpcConfigActivity
func NewGetVpcConfigActivity(awsSessionFactory awsworkflow.AWSFactory) *GetVpcConfigActivity {
	return &GetVpcConfigActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

// return with empty output fields in case VPC stack doesn't exists anymore
func (a *GetVpcConfigActivity) Execute(ctx context.Context, input GetVpcConfigActivityInput) (*GetVpcConfigActivityOutput, error) {
	output := GetVpcConfigActivityOutput{}

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	cloudformationClient := cloudformation.New(session)

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(input.StackName)}
	describeStacksOutput, err := cloudformationClient.DescribeStacksWithContext(ctx, describeStacksInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			stackDoesntExistsEmssage := fmt.Sprintf("Stack with id %v does not exist", input.StackName)
			if awsErr.Message() == stackDoesntExistsEmssage {
				return &output, nil
			}
		}
		return nil, errors.WrapIfWithDetails(err, "failed to describe stack", "stack", input.StackName)
	}

	for _, outputPrm := range describeStacksOutput.Stacks[0].Outputs {
		switch aws.StringValue(outputPrm.OutputKey) {
		case "VpcId":
			output.VpcID = aws.StringValue(outputPrm.OutputValue)
		case "SecurityGroups":
			output.SecurityGroupID = aws.StringValue(outputPrm.OutputValue)
		case "NodeSecurityGroup":
			output.NodeSecurityGroupID = aws.StringValue(outputPrm.OutputValue)
		}
	}

	return &output, nil
}

// Register registers the activity.
func (a GetVpcConfigActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: GetVpcConfigActivityName})
}

// getVPCConfig retrieves the VPC configuration for the specified VPC
// stack name.
//
// This is a convenience wrapper around the corresponding activity.
func getVPCConfig(
	ctx workflow.Context,
	eksActivityInput EKSActivityInput,
	stackName string,
) (GetVpcConfigActivityOutput, error) {
	var activityOutput GetVpcConfigActivityOutput
	err := getVPCConfigAsync(ctx, eksActivityInput, stackName).Get(ctx, &activityOutput)
	if err != nil {
		return GetVpcConfigActivityOutput{}, err
	}

	return activityOutput, nil
}

// getVPCConfigAsync returns a future object for retrieving the VPC
// configuration for the specified VPC stack name.
//
// This is a convenience wrapper around the corresponding activity.
func getVPCConfigAsync(
	ctx workflow.Context,
	eksActivityInput EKSActivityInput,
	stackName string,
) workflow.Future {
	return workflow.ExecuteActivity(ctx, GetVpcConfigActivityName, GetVpcConfigActivityInput{
		EKSActivityInput: eksActivityInput,
		StackName:        stackName,
	})
}
