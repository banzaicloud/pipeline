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
	"strings"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

func TestGetCFStack(t *testing.T) {
	type inputType struct {
		eksActivityInput EKSActivityInput
		stackName        string
	}

	type outputType struct {
		expectedError error
		expectedStack *cloudformation.Stack
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "error",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName: "stack-name",
			},
			output: outputType{
				expectedError: errors.New("test error"),
				expectedStack: nil,
			},
		},
		{
			caseDescription: "success",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName: "stack-name",
			},
			output: outputType{
				expectedError: nil,
				expectedStack: &cloudformation.Stack{
					Parameters: []*cloudformation.Parameter{
						{
							ParameterKey:   aws.String("parameter-name"),
							ParameterValue: aws.String("parameter-value"),
						},
					},
					StackId:     aws.String("stack-id"),
					StackName:   aws.String("stack-name"),
					StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input GetCFStackActivityInput) (*GetCFStackActivityOutput, error) {
					return &GetCFStackActivityOutput{
						Stack: testCase.output.expectedStack,
					}, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: GetCFStackActivityName},
			)

			var actualError error
			var actualStack *cloudformation.Stack
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualStack, actualError = getCFStack(
					ctx,
					testCase.input.eksActivityInput,
					testCase.input.stackName,
				)

				return nil
			})

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedStack, actualStack)
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
					EKSActivityInput: EKSActivityInput{},
					StackName:        "stack-name",
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
					EKSActivityInput: EKSActivityInput{},
					StackName:        "stack-name",
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
					EKSActivityInput: EKSActivityInput{},
					StackName:        "stack-name",
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
					EKSActivityInput: EKSActivityInput{},
					StackName:        "stack-name",
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
					EKSActivityInput: EKSActivityInput{},
					StackName:        "stack-name",
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

			testActivityEnvironment.RegisterActivityWithOptions(
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

func TestGetCFStackAsync(t *testing.T) {
	type inputType struct {
		eksActivityInput EKSActivityInput
		stackName        string
	}

	type outputType struct {
		expectedError  error
		expectedOutput *GetCFStackActivityOutput
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "error",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName: "stack-name",
			},
			output: outputType{
				expectedError:  errors.New("test error"),
				expectedOutput: nil,
			},
		},
		{
			caseDescription: "success",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName: "stack-name",
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &GetCFStackActivityOutput{
					Stack: &cloudformation.Stack{
						Parameters: []*cloudformation.Parameter{
							{
								ParameterKey:   aws.String("parameter-name"),
								ParameterValue: aws.String("parameter-value"),
							},
						},
						StackId:     aws.String("stack-id"),
						StackName:   aws.String("stack-name"),
						StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input GetCFStackActivityInput) (*GetCFStackActivityOutput, error) {
					return testCase.output.expectedOutput, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: GetCFStackActivityName},
			)

			var actualError error
			var actualOutput *GetCFStackActivityOutput
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualFuture := getCFStackAsync(
					ctx,
					testCase.input.eksActivityInput,
					testCase.input.stackName,
				)
				actualError = actualFuture.Get(ctx, &actualOutput)

				return nil
			})

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedOutput, actualOutput)
		})
	}
}

func TestGetCFStackOutputs(t *testing.T) {
	type inputType struct {
		eksActivityInput    EKSActivityInput
		stackName           string
		typedOutputsPointer interface{}
	}

	type expectedOutputsType struct {
		Bool   bool
		Float  float64
		Int    int
		String string
		Uint   uint
	}

	type outputType struct {
		expectedError   error
		expectedOutputs *expectedOutputsType
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "nil typed outputs pointer error -> error",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName:           "stack-name",
				typedOutputsPointer: nil,
			},
			output: outputType{
				expectedError:   errors.New("typed outputs pointer is nil"),
				expectedOutputs: nil,
			},
		},
		{
			caseDescription: "get CF stack error -> error",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName:           "stack-name",
				typedOutputsPointer: &expectedOutputsType{},
			},
			output: outputType{
				expectedError:   errors.New("get CF stack error"),
				expectedOutputs: &expectedOutputsType{},
			},
		},
		{
			caseDescription: "parse CF stack outputs error -> error",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName:           "stack-name",
				typedOutputsPointer: &expectedOutputsType{},
			},
			output: outputType{
				expectedError: errors.New(
					"parsing stack outputs failed" +
						": missing expected key Bool" +
						"; missing expected key Float" +
						"; missing expected key Int" +
						"; missing expected key String" +
						"; missing expected key Uint",
				),
				expectedOutputs: &expectedOutputsType{},
			},
		},
		{
			caseDescription: "success",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName:           "stack-name",
				typedOutputsPointer: &expectedOutputsType{},
			},
			output: outputType{
				expectedError: nil,
				expectedOutputs: &expectedOutputsType{
					Bool:   true,
					Float:  3.14,
					Int:    -5,
					String: "value",
					Uint:   2,
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input GetCFStackActivityInput) (*GetCFStackActivityOutput, error) {
					if testCase.output.expectedError != nil &&
						strings.HasPrefix(testCase.output.expectedError.Error(), "parsing stack outputs failed: ") {
						return &GetCFStackActivityOutput{
							Stack: &cloudformation.Stack{
								Outputs: []*cloudformation.Output{
									{
										OutputKey:   aws.String("bool"),
										OutputValue: aws.String("not-a-bool"),
									},
								},
								StackId:     aws.String("stack-id"),
								StackName:   aws.String(testCase.input.stackName),
								StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
							},
						}, nil
					}

					return &GetCFStackActivityOutput{
						Stack: &cloudformation.Stack{
							Outputs: []*cloudformation.Output{
								{
									OutputKey:   aws.String("Bool"),
									OutputValue: aws.String("true"),
								},
								{
									OutputKey:   aws.String("Float"),
									OutputValue: aws.String("3.14"),
								},
								{
									OutputKey:   aws.String("Int"),
									OutputValue: aws.String("-5"),
								},
								{
									OutputKey:   aws.String("String"),
									OutputValue: aws.String("value"),
								},
								{
									OutputKey:   aws.String("Uint"),
									OutputValue: aws.String("2"),
								},
							},
							StackId:     aws.String("stack-id"),
							StackName:   aws.String(testCase.input.stackName),
							StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
						},
					}, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: GetCFStackActivityName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualError = getCFStackOutputs(
					ctx,
					testCase.input.eksActivityInput,
					testCase.input.stackName,
					testCase.input.typedOutputsPointer,
				)

				return nil
			})

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			if testCase.output.expectedOutputs == nil {
				require.Nil(t, testCase.input.typedOutputsPointer)
			} else {
				require.Equal(t, testCase.output.expectedOutputs, testCase.input.typedOutputsPointer)
			}
		})
	}
}

func TestGetCFStackParameters(t *testing.T) {
	type inputType struct {
		eksActivityInput       EKSActivityInput
		stackName              string
		typedParametersPointer interface{}
	}

	type expectedParametersType struct {
		Bool   bool
		Float  float64
		Int    int
		String string
		Uint   uint
	}

	type outputType struct {
		expectedError      error
		expectedParameters *expectedParametersType
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "nil typed parameters pointer error -> error",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName:              "stack-name",
				typedParametersPointer: nil,
			},
			output: outputType{
				expectedError:      errors.New("typed parameters pointer is nil"),
				expectedParameters: nil,
			},
		},
		{
			caseDescription: "get CF stack error -> error",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName:              "stack-name",
				typedParametersPointer: &expectedParametersType{},
			},
			output: outputType{
				expectedError:      errors.New("get CF stack error"),
				expectedParameters: &expectedParametersType{},
			},
		},
		{
			caseDescription: "parse CF stack parameters error -> error",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName:              "stack-name",
				typedParametersPointer: &expectedParametersType{},
			},
			output: outputType{
				expectedError: errors.New(
					"parsing stack parameters failed" +
						": missing expected key Bool" +
						"; missing expected key Float" +
						"; missing expected key Int" +
						"; missing expected key String" +
						"; missing expected key Uint",
				),
				expectedParameters: &expectedParametersType{},
			},
		},
		{
			caseDescription: "success",
			input: inputType{
				eksActivityInput: EKSActivityInput{
					OrganizationID: 1,
					SecretID:       "brn:1:secret:id",
					Region:         "region",
					ClusterName:    "cluster-name",
				},
				stackName:              "stack-name",
				typedParametersPointer: &expectedParametersType{},
			},
			output: outputType{
				expectedError: nil,
				expectedParameters: &expectedParametersType{
					Bool:   true,
					Float:  3.14,
					Int:    -5,
					String: "value",
					Uint:   2,
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input GetCFStackActivityInput) (*GetCFStackActivityOutput, error) {
					if testCase.output.expectedError != nil &&
						strings.HasPrefix(testCase.output.expectedError.Error(), "parsing stack parameters failed: ") {
						return &GetCFStackActivityOutput{
							Stack: &cloudformation.Stack{
								Parameters: []*cloudformation.Parameter{
									{
										ParameterKey:   aws.String("bool"),
										ParameterValue: aws.String("not-a-bool"),
									},
								},
								StackId:     aws.String("stack-id"),
								StackName:   aws.String(testCase.input.stackName),
								StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
							},
						}, nil
					}

					return &GetCFStackActivityOutput{
						Stack: &cloudformation.Stack{
							Parameters: []*cloudformation.Parameter{
								{
									ParameterKey:   aws.String("Bool"),
									ParameterValue: aws.String("true"),
								},
								{
									ParameterKey:   aws.String("Float"),
									ParameterValue: aws.String("3.14"),
								},
								{
									ParameterKey:   aws.String("Int"),
									ParameterValue: aws.String("-5"),
								},
								{
									ParameterKey:   aws.String("String"),
									ParameterValue: aws.String("value"),
								},
								{
									ParameterKey:   aws.String("Uint"),
									ParameterValue: aws.String("2"),
								},
							},
							StackId:     aws.String("stack-id"),
							StackName:   aws.String(testCase.input.stackName),
							StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
						},
					}, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: GetCFStackActivityName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualError = getCFStackParameters(
					ctx,
					testCase.input.eksActivityInput,
					testCase.input.stackName,
					testCase.input.typedParametersPointer,
				)

				return nil
			})

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			if testCase.output.expectedParameters == nil {
				require.Nil(t, testCase.input.typedParametersPointer)
			} else {
				require.Equal(t, testCase.output.expectedParameters, testCase.input.typedParametersPointer)
			}
		})
	}
}

func TestNewGetCFStackActivity(t *testing.T) {
	type inputType struct {
		awsFactory            awsworkflow.AWSFactory
		cloudFormationFactory awsworkflow.CloudFormationAPIFactory
	}

	type outputType struct {
		expectedActivity *GetCFStackActivity
	}

	type caseType struct {
		caseDescription string
		input           inputType
		output          outputType
	}

	testCases := []caseType{
		{
			caseDescription: "nil AWS factory, nil CloudFormation factory -> success",
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
			caseDescription: "nil AWS factory, not nil CloudFormation factory -> success",
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
			caseDescription: "not nil AWS factory, nil CloudFormation factory -> success",
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
			caseDescription: "not nil AWS factory, not nil CloudFormation factory -> success",
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

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualActivity := NewGetCFStackActivity(testCase.input.awsFactory, testCase.input.cloudFormationFactory)

			require.Equal(t, testCase.output.expectedActivity, actualActivity)
		})
	}
}
