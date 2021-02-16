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

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

func TestNewGetVPCConfigActivity(t *testing.T) {
	testCases := []struct {
		caseDescription        string
		expectedActivity       *GetVpcConfigActivity
		inputAWSSessionFactory awsworkflow.AWSFactory
	}{
		{
			caseDescription:        "nil AWS factory -> success",
			expectedActivity:       &GetVpcConfigActivity{awsSessionFactory: nil},
			inputAWSSessionFactory: nil,
		},
		{
			caseDescription:        "not nil AWS factory -> success",
			expectedActivity:       &GetVpcConfigActivity{awsSessionFactory: &awsworkflow.MockAWSFactory{}},
			inputAWSSessionFactory: &awsworkflow.MockAWSFactory{},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualActivity := NewGetVpcConfigActivity(testCase.inputAWSSessionFactory)

			require.Equal(t, testCase.expectedActivity, actualActivity)
		})
	}
}

func TestGetVPCConfig(t *testing.T) {
	type inputType struct {
		eksActivityInput EKSActivityInput
		stackName        string
	}

	type outputType struct {
		expectedError  error
		expectedOutput GetVpcConfigActivityOutput
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
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
				expectedOutput: GetVpcConfigActivityOutput{
					VpcID:               "vpc-id",
					SecurityGroupID:     "security-group-id",
					NodeSecurityGroupID: "node-security-group-id",
				},
			},
		},
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
				expectedOutput: GetVpcConfigActivityOutput{},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input GetVpcConfigActivityInput) (*GetVpcConfigActivityOutput, error) {
					return &testCase.output.expectedOutput, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: GetVpcConfigActivityName},
			)

			var actualError error
			var actualOutput GetVpcConfigActivityOutput
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualOutput, actualError = getVPCConfig(
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
			require.Equal(t, testCase.output.expectedOutput, actualOutput)
		})
	}
}

func TestGetVPCConfigAsync(t *testing.T) {
	type inputType struct {
		eksActivityInput EKSActivityInput
		stackName        string
	}

	type outputType struct {
		expectedError  error
		expectedOutput *GetVpcConfigActivityOutput
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
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
				expectedOutput: &GetVpcConfigActivityOutput{
					VpcID:               "vpc-id",
					SecurityGroupID:     "security-group-id",
					NodeSecurityGroupID: "node-security-group-id",
				},
			},
		},
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
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input GetVpcConfigActivityInput) (*GetVpcConfigActivityOutput, error) {
					return testCase.output.expectedOutput, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: GetVpcConfigActivityName},
			)

			var actualError error
			var actualOutput *GetVpcConfigActivityOutput
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualFuture := getVPCConfigAsync(
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
