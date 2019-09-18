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

func NewAwsStackFailure(awsStackError error, stackName string, cloudformationSrv *cloudformation.CloudFormation) error {
	if awsStackError == nil {
		return nil
	}

	failedStackEvents, err := collectFailedStackEvents(stackName, cloudformationSrv)
	if err != nil {
		return errors.Append(awsStackError, errors.WrapIf(err, "could not retrieve stack events with 'FAILED' state"))
	}

	if len(failedStackEvents) > 0 {
		var failedEventsMsg []string

		for _, event := range failedStackEvents {
			msg := fmt.Sprintf("%v %v %v", aws.StringValue(event.LogicalResourceId), aws.StringValue(event.ResourceStatus), aws.StringValue(event.ResourceStatusReason))
			failedEventsMsg = append(failedEventsMsg, msg)
		}

		return awsStackFailedError{
			awsStackError:   awsStackError,
			stackName:       stackName,
			failedEventsMsg: failedEventsMsg,
		}
	}

	return awsStackError
}

func collectFailedStackEvents(stackName string, cloudformationSrv *cloudformation.CloudFormation) ([]*cloudformation.StackEvent, error) {
	var failedStackEvents []*cloudformation.StackEvent

	describeStackEventsInput := &cloudformation.DescribeStackEventsInput{StackName: aws.String(stackName)}
	err := cloudformationSrv.DescribeStackEventsPages(describeStackEventsInput,
		func(page *cloudformation.DescribeStackEventsOutput, lastPage bool) bool {

			for _, event := range page.StackEvents {
				if strings.HasSuffix(*event.ResourceStatus, "FAILED") {
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
