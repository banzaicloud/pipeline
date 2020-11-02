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
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"
)

func TestNewSelectVolumeSizeActivity(t *testing.T) {
	testCases := []struct {
		caseDescription            string
		expectedActivity           *SelectVolumeSizeActivity
		inputDefaultNodeVolumeSize int
	}{
		{
			caseDescription: "not zero default node volume size success",
			expectedActivity: &SelectVolumeSizeActivity{
				defaultVolumeSize: 1,
			},
			inputDefaultNodeVolumeSize: 1,
		},
		{
			caseDescription:            "zero default node volume size success",
			expectedActivity:           &SelectVolumeSizeActivity{},
			inputDefaultNodeVolumeSize: 0,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualActivity := NewSelectVolumeSizeActivity(testCase.inputDefaultNodeVolumeSize)

			require.Equal(t, testCase.expectedActivity, actualActivity)
		})
	}
}

func TestSelectVolumeSizeActivityExecute(t *testing.T) {
	type inputType struct {
		activity *SelectVolumeSizeActivity
		input    SelectVolumeSizeActivityInput
	}

	type outputType struct {
		expectedError  error
		expectedOutput *SelectVolumeSizeActivityOutput
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "too small explicit volume size error",
			input: inputType{
				activity: NewSelectVolumeSizeActivity(0),
				input: SelectVolumeSizeActivityInput{
					AMISize:            2,
					OptionalVolumeSize: 1,
				},
			},
			output: outputType{
				expectedError: errors.New(
					"selected volume size of 1 GB (source: explicitly set) is less than the AMI size of 2 GB",
				),
				expectedOutput: nil,
			},
		},
		{
			caseDescription: "too small default volume size error",
			input: inputType{
				activity: NewSelectVolumeSizeActivity(1),
				input: SelectVolumeSizeActivityInput{
					AMISize:            2,
					OptionalVolumeSize: 0,
				},
			},
			output: outputType{
				expectedError: errors.New(
					"selected volume size of 1 GB (source: default configured) is less than the AMI size of 2 GB",
				),
				expectedOutput: nil,
			},
		},
		{
			caseDescription: "explicitly set success",
			input: inputType{
				activity: NewSelectVolumeSizeActivity(0),
				input: SelectVolumeSizeActivityInput{
					AMISize:            1,
					OptionalVolumeSize: 1,
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &SelectVolumeSizeActivityOutput{
					VolumeSize: 1,
				},
			},
		},
		{
			caseDescription: "default configured success",
			input: inputType{
				activity: NewSelectVolumeSizeActivity(1),
				input: SelectVolumeSizeActivityInput{
					AMISize:            1,
					OptionalVolumeSize: 0,
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &SelectVolumeSizeActivityOutput{
					VolumeSize: 1,
				},
			},
		},
		{
			caseDescription: "AMI size success",
			input: inputType{
				activity: NewSelectVolumeSizeActivity(0),
				input: SelectVolumeSizeActivityInput{
					AMISize:            60,
					OptionalVolumeSize: 0,
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &SelectVolumeSizeActivityOutput{
					VolumeSize: 60,
				},
			},
		},
		{
			caseDescription: "fallback value success",
			input: inputType{
				activity: NewSelectVolumeSizeActivity(0),
				input: SelectVolumeSizeActivityInput{
					AMISize:            1,
					OptionalVolumeSize: 0,
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &SelectVolumeSizeActivityOutput{
					VolumeSize: 50,
				},
			},
		},
		{
			caseDescription: "explicitly set precedence success",
			input: inputType{
				activity: NewSelectVolumeSizeActivity(49),
				input: SelectVolumeSizeActivityInput{
					AMISize:            48,
					OptionalVolumeSize: 48,
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &SelectVolumeSizeActivityOutput{
					VolumeSize: 48,
				},
			},
		},
		{
			caseDescription: "default configured precedence success",
			input: inputType{
				activity: NewSelectVolumeSizeActivity(49),
				input: SelectVolumeSizeActivityInput{
					AMISize:            48,
					OptionalVolumeSize: 0,
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &SelectVolumeSizeActivityOutput{
					VolumeSize: 49,
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualOutput, actualError := testCase.input.activity.Execute(nil, testCase.input.input)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedOutput, actualOutput)
		})
	}
}

func TestSelectVolumeSize(t *testing.T) {
	type inputType struct {
		amiSize            int
		optionalVolumeSize int
	}

	type outputType struct {
		expectedError      error
		expectedVolumeSize int
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "success",
			input: inputType{
				amiSize:            1,
				optionalVolumeSize: 2,
			},
			output: outputType{
				expectedError:      nil,
				expectedVolumeSize: 2,
			},
		},
		{
			caseDescription: "error",
			input: inputType{
				amiSize:            2,
				optionalVolumeSize: 1,
			},
			output: outputType{
				expectedError: errors.New(
					"selected volume size of 1 GB (source: explicitly set) is less than the AMI size of 2 GB",
				),
				expectedVolumeSize: 0,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input SelectVolumeSizeActivityInput) (*SelectVolumeSizeActivityOutput, error) {
					if testCase.output.expectedError != nil {
						return nil, testCase.output.expectedError
					}

					return &SelectVolumeSizeActivityOutput{
						VolumeSize: testCase.output.expectedVolumeSize,
					}, nil
				},
				activity.RegisterOptions{
					Name: SelectVolumeSizeActivityName,
				},
			)

			var actualError error
			var actualVolumeSize int
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualVolumeSize, actualError = selectVolumeSize(
					ctx,
					testCase.input.amiSize,
					testCase.input.optionalVolumeSize,
				)

				return nil
			})

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedVolumeSize, actualVolumeSize)
		})
	}
}

func TestSelectVolumeSizeAsync(t *testing.T) {
	type inputType struct {
		amiSize            int
		optionalVolumeSize int
	}

	type outputType struct {
		expectedError  error
		expectedOutput *SelectVolumeSizeActivityOutput
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "success",
			input: inputType{
				amiSize:            1,
				optionalVolumeSize: 2,
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &SelectVolumeSizeActivityOutput{
					VolumeSize: 2,
				},
			},
		},
		{
			caseDescription: "error",
			input: inputType{
				amiSize:            2,
				optionalVolumeSize: 1,
			},
			output: outputType{
				expectedError: errors.New(
					"selected volume size of 1 GB (source: explicitly set) is less than the AMI size of 2 GB",
				),
				expectedOutput: nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(
					ctx context.Context,
					input SelectVolumeSizeActivityInput,
				) (*SelectVolumeSizeActivityOutput, error) {
					return testCase.output.expectedOutput, testCase.output.expectedError
				},
				activity.RegisterOptions{
					Name: SelectVolumeSizeActivityName,
				},
			)

			var actualError error
			var actualOutput *SelectVolumeSizeActivityOutput
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualFuture := selectVolumeSizeAsync(
					ctx,
					testCase.input.amiSize,
					testCase.input.optionalVolumeSize,
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
