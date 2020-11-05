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

package workflow

import (
	"context"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	sdkcloudformation "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon/cloudformation"
)

const (
	// GetCFStackActivityName defines the Cadence activity name for retrieving
	// CloudFormation stacks.
	GetCFStackActivityName = "eks-get-cf-stack-activity"
)

// GetCFStackActivity defines the high level component dependencies of
// retrieving a CloudFormation stack.
type GetCFStackActivity struct {
	awsFactory            awsworkflow.AWSFactory
	cloudFormationFactory awsworkflow.CloudFormationAPIFactory
}

// GetCFStackActivityInput defines the required parameters for retrieving a
// CloudFormation stack.
type GetCFStackActivityInput struct {
	EKSActivityInput
	StackName string
}

// GetCFStackActivityOutput encapsulates the returned stack information.
type GetCFStackActivityOutput struct {
	Stack *cloudformation.Stack
}

// NewGetCFStackActivity instantiates an activity for retrieving CloudFormation
// stacks.
func NewGetCFStackActivity(
	awsFactory awsworkflow.AWSFactory, cloudFormationFactory awsworkflow.CloudFormationAPIFactory,
) *GetCFStackActivity {
	return &GetCFStackActivity{
		awsFactory:            awsFactory,
		cloudFormationFactory: cloudFormationFactory,
	}
}

// Execute executes the activity.
func (a *GetCFStackActivity) Execute(ctx context.Context, input GetCFStackActivityInput) (output *GetCFStackActivityOutput, err error) {
	if a == nil {
		return nil, errors.New("activity is nil")
	}

	awsClient, err := a.awsFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err != nil {
		return nil, errors.WrapIf(err, "creating AWS client failed")
	}

	cloudFormationClient := a.cloudFormationFactory.New(awsClient)

	describeStacksInput := &cloudformation.DescribeStacksInput{
		StackName: aws.String(input.StackName),
	}

	describeStacksOutput, err := cloudFormationClient.DescribeStacks(describeStacksInput)
	if err != nil {
		return nil, errors.WrapIf(err, "describing cloudformation stack failed")
	} else if len(describeStacksOutput.Stacks) == 0 {
		return nil, errors.NewWithDetails("missing cloudformation stack", "stackName", input.StackName)
	}

	return &GetCFStackActivityOutput{
		Stack: describeStacksOutput.Stacks[0],
	}, nil
}

// Register registers the activity.
func (a GetCFStackActivity) Register() {
	activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: GetCFStackActivityName})
}

// getCFStack retrieves the CloudFormation stack corresponding to the specified
// stack name. Stack name might be a stack ID as well. Deleted stacks can only
// be queried by stack ID.
//
// This is a convenience wrapper around the corresponding activity.
func getCFStack(
	ctx workflow.Context,
	eksActivityInput EKSActivityInput,
	stackName string,
) (*cloudformation.Stack, error) {
	var activityOutput GetCFStackActivityOutput
	err := getCFStackAsync(ctx, eksActivityInput, stackName).Get(ctx, &activityOutput)
	if err != nil {
		return nil, err
	}

	return activityOutput.Stack, nil
}

// getCFStackAsync returns a future object for retrieving the CloudFormation
// stack corresponding to the specified stack name. Stack name might be a stack
// ID as well. Deleted stacks can only be queried by stack ID.
//
// This is a convenience wrapper around the corresponding activity.
func getCFStackAsync(
	ctx workflow.Context,
	eksActivityInput EKSActivityInput,
	stackName string,
) workflow.Future {
	return workflow.ExecuteActivity(ctx, GetCFStackActivityName, GetCFStackActivityInput{
		EKSActivityInput: eksActivityInput,
		StackName:        stackName,
	})
}

// getCFStackOutputs retrieves the strongly typed CloudFormation stack outputs
// corresponding to the specified stack name. Stack name might be a stack ID as
// well. Deleted stacks can only be queried by stack ID.
//
// This is a convenience wrapper around the corresponding activity.
func getCFStackOutputs(
	ctx workflow.Context,
	eksActivityInput EKSActivityInput,
	stackName string,
	typedOutputsPointer interface{},
) error {
	if typedOutputsPointer == nil {
		return errors.New("typed outputs pointer is nil")
	}

	stack, err := getCFStack(ctx, eksActivityInput, stackName)
	if err != nil {
		return err
	}

	err = sdkcloudformation.ParseStackOutputs(stack.Outputs, typedOutputsPointer)
	if err != nil {
		return errors.WrapWithDetails(err, "parsing stack outputs failed", "outputs", stack.Outputs)
	}

	return nil
}

// getCFStackParameters retrieves the strongly typed CloudFormation stack
// parameters corresponding to the specified stack name. Stack name might be a
// stack ID as well. Deleted stacks can only be queried by stack ID.
//
// This is a convenience wrapper around the corresponding activity.
func getCFStackParameters(
	ctx workflow.Context,
	eksActivityInput EKSActivityInput,
	stackName string,
	typedParametersPointer interface{},
) error {
	if typedParametersPointer == nil {
		return errors.New("typed parameters pointer is nil")
	}

	stack, err := getCFStack(ctx, eksActivityInput, stackName)
	if err != nil {
		return err
	}

	err = sdkcloudformation.ParseStackParameters(stack.Parameters, typedParametersPointer)
	if err != nil {
		return errors.WrapWithDetails(err, "parsing stack parameters failed", "parameters", stack.Parameters)
	}

	return nil
}
