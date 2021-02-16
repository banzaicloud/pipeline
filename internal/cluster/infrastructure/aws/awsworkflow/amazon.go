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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/src/secret"
)

// ErrReasonStackFailed cadence custom error reason that denotes a stack operation that resulted a stack failure
const ErrReasonStackFailed = "CLOUDFORMATION_STACK_FAILED"

// AWSCommonActivityInput holds common input data for all activities
type AWSCommonActivityInput struct {
	OrganizationID uint
	SecretID       string

	Region string

	ClusterName string
}

type SecretStore interface {
	Get(orgnaizationID uint, secretID string) (*secret.SecretItemResponse, error)
	GetByName(orgnaizationID uint, secretID string) (*secret.SecretItemResponse, error)
	Store(organizationID uint, request *secret.CreateSecretRequest) (string, error)
	Delete(organizationID uint, secretID string) error
	Update(organizationID uint, secretID string, request *secret.CreateSecretRequest) error
}

func WaitUntilStackDeleteCompleteWithContext(cf *cloudformation.CloudFormation, ctx aws.Context, input *cloudformation.DescribeStacksInput, opts ...request.WaiterOption) error {
	count := 0
	w := request.Waiter{
		Name:        "WaitUntilStackDeleteComplete",
		MaxAttempts: 120,
		Delay:       request.ConstantWaiterDelay(30 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathAllWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "DELETE_COMPLETE",
			},
			{
				State:    request.SuccessWaiterState,
				Matcher:  request.ErrorWaiterMatch,
				Expected: "ValidationError",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "DELETE_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "CREATE_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "ROLLBACK_FAILED",
			},
			{
				State:   request.FailureWaiterState,
				Matcher: request.PathAnyWaiterMatch, Argument: "Stacks[].StackStatus",
				Expected: "UPDATE_ROLLBACK_IN_PROGRESS",
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
		},
		Logger: cf.Config.Logger,
		NewRequest: func(opts []request.Option) (*request.Request, error) {
			count++
			activity.RecordHeartbeat(ctx, count)

			var inCpy *cloudformation.DescribeStacksInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := cf.DescribeStacksRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}
	w.ApplyOptions(opts...)

	return w.WaitWithContext(ctx)
}
