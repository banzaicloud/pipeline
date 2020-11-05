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

package eksworkflow

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

const WaitCloudFormationStackUpdateActivityName = "eks-wait-cloudformation-stack-update"

// WaitCloudFormationStackUpdateActivity updates an existing node group.
type WaitCloudFormationStackUpdateActivity struct {
	sessionFactory AWSSessionFactory
}

// WaitCloudFormationStackUpdateActivityInput holds the parameters for the node group update.
type WaitCloudFormationStackUpdateActivityInput struct {
	SecretID  string
	Region    string
	StackName string
}

// NewWaitCloudFormationStackUpdateActivity creates a new WaitCloudFormationStackUpdateActivity instance.
func NewWaitCloudFormationStackUpdateActivity(sessionFactory AWSSessionFactory) WaitCloudFormationStackUpdateActivity {
	return WaitCloudFormationStackUpdateActivity{
		sessionFactory: sessionFactory,
	}
}

// Register registers the activity in the worker.
func (a WaitCloudFormationStackUpdateActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: WaitCloudFormationStackUpdateActivityName})
}

// Execute is the main body of the activity.
func (a WaitCloudFormationStackUpdateActivity) Execute(ctx context.Context, input WaitCloudFormationStackUpdateActivityInput) error {
	sess, err := a.sessionFactory.NewSession(input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil { // internal error?
		return err
	}

	cloudformationClient := cloudformation.New(sess)

	count := 0
	if activity.HasHeartbeatDetails(ctx) {
		_ = activity.GetHeartbeatDetails(ctx, &count)
	}

	w := request.Waiter{
		Name:        "WaitUntilStackUpdateComplete",
		MaxAttempts: 120 - count,
		Delay:       request.ConstantWaiterDelay(30 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAllWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_COMPLETE",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_ROLLBACK_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_ROLLBACK_COMPLETE",
			},
			{
				State:    request.FailureWaiterState,
				Matcher:  request.ErrorWaiterMatch,
				Expected: "ValidationError",
			},
		},
		Logger: cloudformationClient.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			count++
			activity.RecordHeartbeat(ctx, count)

			req, _ := cloudformationClient.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
				StackName: aws.String(input.StackName),
			})
			req.SetContext(ctx)
			req.ApplyOptions(opts...)

			return req, nil
		},
	}

	err = w.WaitWithContext(ctx)
	if err != nil {
		return packageCFError(
			err,
			input.StackName,
			activity.GetInfo(ctx).WorkflowExecution.ID, cloudformationClient,
			"waiting for CF stack create operation to complete failed",
		)
	}

	return nil
}
