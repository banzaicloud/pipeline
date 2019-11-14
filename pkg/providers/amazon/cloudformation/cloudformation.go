// Copyright © 2018 Banzai Cloud
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

package cloudformation

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

// GetExistingTaggedStackNames gives back existing CF stacks which have the given tags
func GetExistingTaggedStackNames(cfSvc *cloudformation.CloudFormation, tags map[string]string) ([]string, error) {
	names := make([]string, 0)

	err := cfSvc.DescribeStacksPages(&cloudformation.DescribeStacksInput{}, func(page *cloudformation.DescribeStacksOutput, lastPage bool) bool {
		for _, stack := range page.Stacks {
			stackTags := getFlattenedTags(stack.Tags)
			ok := true
			for k, v := range tags {
				if stackTags[k] != v {
					ok = false
					break
				}
			}
			if ok && stack.StackName != nil {
				names = append(names, *stack.StackName)
			}
		}
		return true
	})

	return names, err
}

func getFlattenedTags(tags []*cloudformation.Tag) map[string]string {

	t := make(map[string]string, 0)

	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			t[*tag.Key] = *tag.Value
		}
	}

	return t
}

type awsStackFailedError struct {
	awsStackError   error
	stackName       string
	failedEventsMsg []string
	isFinal         bool
}

func (e awsStackFailedError) Error() string {
	hdr := "stack " + e.stackName
	if len(e.failedEventsMsg) > 0 {
		return hdr + "\n" + strings.Join(e.failedEventsMsg, "\n")
	}

	return hdr + e.awsStackError.Error()
}

func (e awsStackFailedError) Cause() error {
	return e.awsStackError
}

// IsErrorFinal returns true if the error indicates that it
// originates from a stack that is in CREATE_FAILED, DELETE_FAILED, ROLLBACK_FAILED, UPDATE_ROLLBACK_FAILED state
func IsErrorFinal(err error) bool {
	var awsStackErr awsStackFailedError

	if errors.As(err, &awsStackErr) {
		return awsStackErr.isFinal
	}

	return false
}

func NewAwsStackFailure(awsStackError error, stackName, clientRequestToken string, cloudformationSrv *cloudformation.CloudFormation) error {
	if awsStackError == nil {
		return nil
	}

	stacksOutput, err := cloudformationSrv.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(stackName)})
	if err != nil {
		return errors.Combine(awsStackError, errors.WrapIf(err, "could not describe stack"))
	}

	isFinalErr := false

	switch aws.StringValue(stacksOutput.Stacks[0].StackStatus) {
	case cloudformation.StackStatusCreateFailed, cloudformation.StackStatusDeleteFailed, cloudformation.StackStatusRollbackFailed, cloudformation.StackStatusUpdateRollbackFailed:
		isFinalErr = true
	}

	failedStackEvents, err := collectFailedStackEvents(stackName, clientRequestToken, cloudformationSrv)
	if err != nil {
		return errors.Append(awsStackError, errors.WrapIf(err, "could not retrieve stack events with 'FAILED' state"))
	}

	var failedEventsMsg []string
	if len(failedStackEvents) > 0 {
		for _, event := range failedStackEvents {
			msg := fmt.Sprintf("%v %v %v", aws.StringValue(event.LogicalResourceId), aws.StringValue(event.ResourceStatus), aws.StringValue(event.ResourceStatusReason))
			failedEventsMsg = append(failedEventsMsg, msg)
		}
	}

	return awsStackFailedError{
		awsStackError:   awsStackError,
		stackName:       stackName,
		failedEventsMsg: failedEventsMsg,
		isFinal:         isFinalErr,
	}

}

func collectFailedStackEvents(stackName, clientRequestToken string, cloudformationSrv *cloudformation.CloudFormation) ([]*cloudformation.StackEvent, error) {
	var failedStackEvents []*cloudformation.StackEvent

	describeStackEventsInput := &cloudformation.DescribeStackEventsInput{StackName: aws.String(stackName)}
	err := cloudformationSrv.DescribeStackEventsPages(describeStackEventsInput,
		func(page *cloudformation.DescribeStackEventsOutput, lastPage bool) bool {

			for _, event := range page.StackEvents {
				if clientRequestToken != "" && aws.StringValue(event.ClientRequestToken) != clientRequestToken {
					continue
				}

				if strings.HasSuffix(aws.StringValue(event.ResourceStatus), "FAILED") {
					failedStackEvents = append(failedStackEvents, event)
				}
			}

			return true
		})
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to describe CloudFormation stack events", "stackName", aws.String(stackName))
	}

	return failedStackEvents, nil
}
