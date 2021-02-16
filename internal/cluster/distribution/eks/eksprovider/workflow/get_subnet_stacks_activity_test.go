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
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

func TestGetClusterSubnetStackNames(t *testing.T) {
	type outputType struct {
		expectedError      error
		expectedStackNames []string
	}

	testCases := []struct {
		caseDescription       string
		inputEKSActivityInput EKSActivityInput
		output                outputType
	}{
		{
			caseDescription: "error",
			inputEKSActivityInput: EKSActivityInput{
				OrganizationID: 1,
				SecretID:       "secret-id",
				Region:         "region",
				ClusterName:    "cluster-name",
			},
			output: outputType{
				expectedError:      errors.New("test error"),
				expectedStackNames: nil,
			},
		},
		{
			caseDescription: "success",
			inputEKSActivityInput: EKSActivityInput{
				OrganizationID: 1,
				SecretID:       "secret-id",
				Region:         "region",
				ClusterName:    "cluster-name",
			},
			output: outputType{
				expectedError: nil,
				expectedStackNames: []string{
					"subnet-stack-name-1",
					"subnet-stack-name-2",
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
				func(ctx context.Context, input GetSubnetStacksActivityInput) (*GetSubnetStacksActivityOutput, error) {
					return &GetSubnetStacksActivityOutput{
						StackNames: testCase.output.expectedStackNames,
					}, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: GetSubnetStacksActivityName},
			)

			var actualError error
			var actualStackNames []string
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualStackNames, actualError = getClusterSubnetStackNames(ctx, testCase.inputEKSActivityInput)

				return nil
			})

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedStackNames, actualStackNames)
		})
	}
}

func TestGetClusterSubnetStackNamesAsync(t *testing.T) {
	type outputType struct {
		expectedError  error
		expectedOutput *GetSubnetStacksActivityOutput
	}

	testCases := []struct {
		caseDescription       string
		inputEKSActivityInput EKSActivityInput
		output                outputType
	}{
		{
			caseDescription: "error",
			inputEKSActivityInput: EKSActivityInput{
				OrganizationID: 1,
				SecretID:       "secret-id",
				Region:         "region",
				ClusterName:    "cluster-name",
			},
			output: outputType{
				expectedError:  errors.New("test error"),
				expectedOutput: nil,
			},
		},
		{
			caseDescription: "success",
			inputEKSActivityInput: EKSActivityInput{
				OrganizationID: 1,
				SecretID:       "secret-id",
				Region:         "region",
				ClusterName:    "cluster-name",
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &GetSubnetStacksActivityOutput{
					StackNames: []string{
						"subnet-stack-name-1",
						"subnet-stack-name-2",
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
				func(ctx context.Context, input GetSubnetStacksActivityInput) (*GetSubnetStacksActivityOutput, error) {
					return testCase.output.expectedOutput, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: GetSubnetStacksActivityName},
			)

			var actualError error
			var actualOutput *GetSubnetStacksActivityOutput
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualFuture := getClusterSubnetStackNamesAsync(ctx, testCase.inputEKSActivityInput)
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

func TestNewGetSubnetStacksActivity(t *testing.T) {
	type inputType struct {
		awsSessionFactory *awsworkflow.AWSSessionFactory
	}

	testCases := []struct {
		caseDescription  string
		expectedActivity *GetSubnetStacksActivity
		input            inputType
	}{
		{
			caseDescription:  "nil values -> success",
			expectedActivity: &GetSubnetStacksActivity{},
			input:            inputType{},
		},
		{
			caseDescription: "not nil values -> success",
			expectedActivity: &GetSubnetStacksActivity{
				awsSessionFactory: awsworkflow.NewAWSSessionFactory(nil),
			},
			input: inputType{
				awsSessionFactory: awsworkflow.NewAWSSessionFactory(nil),
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualActivity := NewGetSubnetStacksActivity(testCase.input.awsSessionFactory)

			require.Equal(t, testCase.expectedActivity, actualActivity)
		})
	}
}
