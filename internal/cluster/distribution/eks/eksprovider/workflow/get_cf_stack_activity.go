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

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

const (
	GetCFStackActivityName = "eks-get-cf-stack-activity"
)

type GetCFStackActivity struct {
	awsFactory            awsworkflow.AWSFactory
	cloudFormationFactory awsworkflow.CloudFormationAPIFactory
}

type GetCFStackActivityInput struct {
	awsworkflow.AWSCommonActivityInput
	StackName string
}

type GetCFStackActivityOutput struct {
	Stack *cloudformation.Stack
}

func NewGetCFStackActivity(
	awsFactory awsworkflow.AWSFactory, cloudFormationFactory awsworkflow.CloudFormationAPIFactory,
) *GetCFStackActivity {
	return &GetCFStackActivity{
		awsFactory:            awsFactory,
		cloudFormationFactory: cloudFormationFactory,
	}
}

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
