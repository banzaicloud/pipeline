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
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"go.uber.org/cadence/activity"
)

const DeleteControlPlaneActivityName = "eks-delete-control-plane"

// DeleteControlPlaneActivity responsible for deleting asg
type DeleteControlPlaneActivity struct {
	awsSessionFactory *AWSSessionFactory
}

type DeleteControlPlaneActivityInput struct {
	EKSActivityInput
}

//   DeleteControlPlaneActivityOutput holds the output data of the DeleteControlPlaneActivity
type DeleteControlPlaneActivityOutput struct {
}

//   DeleteControlPlaneActivity instantiates a new DeleteControlPlaneActivity
func NewDeleteControlPlaneActivity(awsSessionFactory *AWSSessionFactory) *DeleteControlPlaneActivity {
	return &DeleteControlPlaneActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *DeleteControlPlaneActivity) Execute(ctx context.Context, input DeleteControlPlaneActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
	)

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return err
	}

	logger.Info("deleting EKS control plane")

	eksSrv := eks.New(awsSession)
	deleteClusterInput := &eks.DeleteClusterInput{
		Name: aws.String(input.ClusterName),
	}
	_, err = eksSrv.DeleteCluster(deleteClusterInput)

	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == eks.ErrCodeResourceNotFoundException {
			return nil
		}
	}

	// wait until cluster exists
	startTime := time.Now()
	logger.Info("waiting for EKS control plane deletion")
	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(input.ClusterName),
	}
	err = a.waitUntilClusterExists(aws.BackgroundContext(), awsSession, describeClusterInput)
	if err != nil {
		return err
	}
	endTime := time.Now()
	logger.Info("EKS control plane deleted successfully in", endTime.Sub(startTime).String())

	return nil
}

func (a *DeleteControlPlaneActivity) waitUntilClusterExists(ctx aws.Context, awsSession *session.Session, input *eks.DescribeClusterInput, opts ...request.WaiterOption) error {
	eksSvc := eks.New(awsSession)

	w := request.Waiter{
		Name:        "WaitUntilClusterExists",
		MaxAttempts: 30,
		Delay:       request.ConstantWaiterDelay(30 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:    request.SuccessWaiterState,
				Matcher:  request.StatusWaiterMatch,
				Expected: 404,
			},
			{
				State:    request.RetryWaiterState,
				Matcher:  request.ErrorWaiterMatch,
				Expected: "ValidationError",
			},
		},
		Logger: eksSvc.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			var inCpy *eks.DescribeClusterInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := eksSvc.DescribeClusterRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	w.ApplyOptions(opts...)

	return w.WaitWithContext(ctx)
}
