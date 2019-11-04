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

package pkeworkflow

import (
	"context"

	"emperror.dev/emperror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"

	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

const WaitCFCompletionActivityName = "pke-wait-cf-completion-activity"

type WaitCFCompletionActivity struct {
	awsClientFactory *AWSClientFactory
}

func NewWaitCFCompletionActivity(awsClientFactory *AWSClientFactory) *WaitCFCompletionActivity {
	return &WaitCFCompletionActivity{
		awsClientFactory: awsClientFactory,
	}
}

type WaitCFCompletionActivityInput struct {
	AWSActivityInput
	StackID string
}

func (a *WaitCFCompletionActivity) Execute(ctx context.Context, input WaitCFCompletionActivityInput) (map[string]string, error) {
	client, err := a.awsClientFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err != nil {
		return nil, err
	}

	cfClient := cloudformation.New(client)

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(input.StackID)}

	err = cfClient.WaitUntilStackCreateCompleteWithContext(ctx, describeStacksInput)
	if err != nil {
		return nil, emperror.Wrap(pkgCloudformation.NewAwsStackFailure(err, "", input.StackID, cfClient), "error waiting Cloud Formation template")
	}

	output, err := cfClient.DescribeStacks(describeStacksInput)
	if err != nil {
		return nil, emperror.Wrap(err, "error fetching Cloud Formation template")
	}

	outputMap := make(map[string]string)
	for _, p := range output.Stacks[0].Outputs {
		outputMap[*p.OutputKey] = *p.OutputValue
	}

	return outputMap, nil
}
