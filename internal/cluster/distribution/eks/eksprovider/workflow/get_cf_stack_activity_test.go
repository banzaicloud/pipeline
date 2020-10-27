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
	"testing"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

func TestNewGetCFStackActivity(t *testing.T) {
	type inputType struct {
		awsFactory            awsworkflow.AWSFactory
		cloudFormationFactory awsworkflow.CloudFormationAPIFactory
	}

	type outputType struct {
		expectedActivity *GetCFStackActivity
	}

	type caseType struct {
		caseName string
		input    inputType
		output   outputType
	}

	testCases := []caseType{
		{
			caseName: "nil AWS factory, nil CloudFormation factory",
			input: inputType{
				awsFactory:            nil,
				cloudFormationFactory: nil,
			},
			output: outputType{
				expectedActivity: &GetCFStackActivity{
					awsFactory:            nil,
					cloudFormationFactory: nil,
				},
			},
		},
		{
			caseName: "nil AWS factory, not nil CloudFormation factory",
			input: inputType{
				awsFactory:            nil,
				cloudFormationFactory: awsworkflow.NewCloudFormationFactory(),
			},
			output: outputType{
				expectedActivity: &GetCFStackActivity{
					awsFactory:            nil,
					cloudFormationFactory: awsworkflow.NewCloudFormationFactory(),
				},
			},
		},
		{
			caseName: "not nil AWS factory, nil CloudFormation factory",
			input: inputType{
				awsFactory:            awsworkflow.NewAWSSessionFactory(nil),
				cloudFormationFactory: nil,
			},
			output: outputType{
				expectedActivity: &GetCFStackActivity{
					awsFactory:            awsworkflow.NewAWSSessionFactory(nil),
					cloudFormationFactory: nil,
				},
			},
		},
		{
			caseName: "not nil AWS factory, not nil CloudFormation factory",
			input: inputType{
				awsFactory:            awsworkflow.NewAWSSessionFactory(nil),
				cloudFormationFactory: awsworkflow.NewCloudFormationFactory(),
			},
			output: outputType{
				expectedActivity: &GetCFStackActivity{
					awsFactory:            awsworkflow.NewAWSSessionFactory(nil),
					cloudFormationFactory: awsworkflow.NewCloudFormationFactory(),
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualActivity := NewGetCFStackActivity(testCase.input.awsFactory, testCase.input.cloudFormationFactory)

			require.Equal(t, testCase.output.expectedActivity, actualActivity)
		})
	}
}

func TestGetCFStackActivityExecute(t *testing.T) {
	type inputType struct {
		activity *GetCFStackActivity
		input    GetCFStackActivityInput
	}

	type outputType struct {
		expectedError  error
		expectedOutput *GetCFStackActivityOutput
	}

	type caseType struct {
		caseName   string
		input      inputType
		output     outputType
		setupMocks func(input inputType, output outputType)
	}

	testCases := []caseType{
		{
			caseName: "nil activity",
			input: inputType{
				activity: nil,
				input: GetCFStackActivityInput{
					AWSCommonActivityInput: awsworkflow.AWSCommonActivityInput{},
					StackName:              "stack-name",
				},
			},
			output: outputType{
				expectedError:  errors.New("activity is nil"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {},
		},
		{
			caseName: "AWS factory error",
			input: inputType{
				activity: NewGetCFStackActivity(&awsworkflow.MockAWSFactory{}, &awsworkflow.MockCloudFormationAPIFactory{}),
				input: GetCFStackActivityInput{
					AWSCommonActivityInput: awsworkflow.AWSCommonActivityInput{},
					StackName:              "stack-name",
				},
			},
			output: outputType{
				expectedError:  errors.New("creating AWS client failed: test error"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSFactory, isOk := input.activity.awsFactory.(*awsworkflow.MockAWSFactory)
				require.True(t, isOk, "test AWS factory is not a mock")

				mockAWSFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(nil, errors.New("test error"))
			},
		},
		{
			caseName: "invalid stack name error",
			input: inputType{
				activity: NewGetCFStackActivity(&awsworkflow.MockAWSFactory{}, &awsworkflow.MockCloudFormationAPIFactory{}),
				input: GetCFStackActivityInput{
					AWSCommonActivityInput: awsworkflow.AWSCommonActivityInput{},
					StackName:              "stack-name",
				},
			},
			output: outputType{
				expectedError:  errors.New("describing cloudformation stack failed: test error"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSFactory, isOk := input.activity.awsFactory.(*awsworkflow.MockAWSFactory)
				require.True(t, isOk, "test AWS factory is not a mock")

				mockAWSSession := &session.Session{}
				mockAWSFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(mockAWSSession, nil)

				mockCFFactory, isOk := input.activity.cloudFormationFactory.(*awsworkflow.MockCloudFormationAPIFactory)
				require.True(t, isOk, "test CloudFormation factory is not a mock")

				mockCFClient := &awsworkflow.MockcloudFormationAPI{}
				mockCFFactory.On(
					"New",
					mockAWSSession,
				).Return(mockCFClient)

				mockCFClient.On(
					"DescribeStacks",
					&cloudformation.DescribeStacksInput{
						StackName: aws.String(input.input.StackName),
					},
				).Return(nil, errors.New("test error"))
			},
		},
		{
			caseName: "stack not found error",
			input: inputType{
				activity: NewGetCFStackActivity(&awsworkflow.MockAWSFactory{}, &awsworkflow.MockCloudFormationAPIFactory{}),
				input: GetCFStackActivityInput{
					AWSCommonActivityInput: awsworkflow.AWSCommonActivityInput{},
					StackName:              "stack-name",
				},
			},
			output: outputType{
				expectedError:  errors.New("missing cloudformation stack"),
				expectedOutput: nil,
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSFactory, isOk := input.activity.awsFactory.(*awsworkflow.MockAWSFactory)
				require.True(t, isOk, "test AWS factory is not a mock")

				mockAWSSession := &session.Session{}
				mockAWSFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(mockAWSSession, nil)

				mockCFFactory, isOk := input.activity.cloudFormationFactory.(*awsworkflow.MockCloudFormationAPIFactory)
				require.True(t, isOk, "test CloudFormation factory is not a mock")

				mockCFClient := &awsworkflow.MockcloudFormationAPI{}
				mockCFFactory.On(
					"New",
					mockAWSSession,
				).Return(mockCFClient)

				mockDescribeStacksOutput := &cloudformation.DescribeStacksOutput{
					Stacks: []*cloudformation.Stack{},
				}
				mockCFClient.On(
					"DescribeStacks",
					&cloudformation.DescribeStacksInput{
						StackName: aws.String(input.input.StackName),
					},
				).Return(mockDescribeStacksOutput, nil)
			},
		},
		{
			caseName: "success",
			input: inputType{
				activity: NewGetCFStackActivity(&awsworkflow.MockAWSFactory{}, &awsworkflow.MockCloudFormationAPIFactory{}),
				input: GetCFStackActivityInput{
					AWSCommonActivityInput: awsworkflow.AWSCommonActivityInput{},
					StackName:              "stack-name",
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &GetCFStackActivityOutput{
					Stack: &cloudformation.Stack{
						StackName: aws.String("stack-name"),
					},
				},
			},
			setupMocks: func(input inputType, output outputType) {
				mockAWSFactory, isOk := input.activity.awsFactory.(*awsworkflow.MockAWSFactory)
				require.True(t, isOk, "test AWS factory is not a mock")

				mockAWSSession := &session.Session{}
				mockAWSFactory.On(
					"New",
					input.input.OrganizationID,
					input.input.SecretID,
					input.input.Region,
				).Return(mockAWSSession, nil)

				mockCFFactory, isOk := input.activity.cloudFormationFactory.(*awsworkflow.MockCloudFormationAPIFactory)
				require.True(t, isOk, "test CloudFormation factory is not a mock")

				mockCFClient := &awsworkflow.MockcloudFormationAPI{}
				mockCFFactory.On(
					"New",
					mockAWSSession,
				).Return(mockCFClient)

				mockDescribeStacksOutput := &cloudformation.DescribeStacksOutput{
					Stacks: []*cloudformation.Stack{
						{
							StackName: aws.String(input.input.StackName),
						},
					},
				}
				mockCFClient.On(
					"DescribeStacks",
					&cloudformation.DescribeStacksInput{
						StackName: aws.String(input.input.StackName),
					},
				).Return(mockDescribeStacksOutput, nil)
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			workflowTestSuite := &testsuite.WorkflowTestSuite{}
			testActivityEnvironment := workflowTestSuite.NewTestActivityEnvironment()

			activity.RegisterWithOptions(
				testCase.input.activity.Execute,
				activity.RegisterOptions{Name: testCase.caseName},
			)
			testCase.setupMocks(testCase.input, testCase.output)

			actualValue, actualError := testActivityEnvironment.ExecuteActivity(
				testCase.caseName,
				testCase.input.input,
			)
			var actualOutput *GetCFStackActivityOutput
			if actualValue != nil &&
				actualValue.HasValue() {
				err := actualValue.Get(&actualOutput)
				require.NoError(t, err)
			}

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedOutput, actualOutput)
		})
	}
}
