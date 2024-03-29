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

package awsworkflow

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"

	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
)

const DeleteStackActivityName = "aws-common-delete-stack"

// DeleteStackActivity responsible for deleting asg
type DeleteStackActivity struct {
	awsSessionFactory AWSFactory
}

type DeleteStackActivityInput struct {
	AWSCommonActivityInput
	StackID string

	// name of the cloud formation template stack
	StackName string
}

// DeleteStackActivityOutput holds the output data of the DeleteStackActivity
type DeleteStackActivityOutput struct{}

// NewDeleteStackActivity instantiates a new DeleteStackActivity
func NewDeleteStackActivity(awsSessionFactory AWSFactory) *DeleteStackActivity {
	return &DeleteStackActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *DeleteStackActivity) Execute(ctx context.Context, input DeleteStackActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"stackName", input.StackName,
	)

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return err
	}

	cloudformationClient := cloudformation.New(awsSession)

	// Note: simplify this part when stack name can be thrown out.
	stackIdentifier := input.StackName
	if input.StackID != "" {
		stackIdentifier = input.StackID
	}

	describeStacksInput := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackIdentifier),
	}
	describeStacksOutput, err := cloudformationClient.DescribeStacks(describeStacksInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == cloudformation.ErrCodeStackInstanceNotFoundException ||
				(awsErr.Code() == "ValidationError" &&
					awsErr.Message() == fmt.Sprintf("Stack with id %s does not exist", stackIdentifier)) {
				// Note: no stack found for the corresponding stack name.
				return nil
			}
		}

		return err
	}

	if len(describeStacksOutput.Stacks) == 0 ||
		aws.StringValue(describeStacksOutput.Stacks[0].StackStatus) == cloudformation.StackStatusDeleteInProgress ||
		aws.StringValue(describeStacksOutput.Stacks[0].StackStatus) == cloudformation.StackStatusDeleteComplete {
		// Note: stack is already (being) deleted.
		return nil
	}

	logger.Info("deleting stack")

	requestToken := aws.String(sdkAmazon.NewNormalizedClientRequestToken(activity.GetInfo(ctx).WorkflowExecution.ID))
	deleteStackInput := &cloudformation.DeleteStackInput{
		ClientRequestToken: requestToken,
		StackName:          aws.String(input.StackName),
	}
	_, err = cloudformationClient.DeleteStack(deleteStackInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == cloudformation.ErrCodeStackInstanceNotFoundException {
				return nil
			}
		}
		return err
	}

	err = WaitUntilStackDeleteCompleteWithContext(cloudformationClient, ctx, describeStacksInput)
	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			if awsErr.Code() == request.WaiterResourceNotReadyErrorCode {
				err = pkgCloudformation.NewAwsStackFailure(err, input.StackName, *requestToken, cloudformationClient)
				err = errors.WrapIff(err, "waiting for %q CF stack create operation to complete failed", input.StackName)
				if pkgCloudformation.IsErrorFinal(err) {
					return cadence.NewCustomError(ErrReasonStackFailed, err.Error())
				}
				return errors.WrapIff(err, "waiting for %q CF stack create operation to complete failed", input.StackName)
			}
		}
	}

	return nil
}
