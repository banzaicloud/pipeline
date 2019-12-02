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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"

	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

const DeleteStackActivityName = "eks-delete-stack"

// DeleteStackActivity responsible for deleting asg
type DeleteStackActivity struct {
	awsSessionFactory AWSFactory
}

type DeleteStackActivityInput struct {
	EKSActivityInput

	// name of the cloud formation template stack
	StackName string
}

//   DeleteStackActivityOutput holds the output data of the DeleteStackActivity
type DeleteStackActivityOutput struct {
}

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

	logger.Info("deleting stack")

	clientRequestToken := generateRequestToken(input.AWSClientRequestTokenBase, DeleteStackActivityName)
	deleteStackInput := &cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(clientRequestToken),
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

	describeStacksInput := &cloudformation.DescribeStacksInput{StackName: aws.String(input.StackName)}
	err = cloudformationClient.WaitUntilStackDeleteComplete(describeStacksInput)
	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			if awsErr.Code() == request.WaiterResourceNotReadyErrorCode {
				err = pkgCloudformation.NewAwsStackFailure(err, input.StackName, clientRequestToken, cloudformationClient)
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
