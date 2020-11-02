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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
)

func TestNewSetNodePoolStatusActivity(t *testing.T) {
	testCases := []struct {
		caseDescription    string
		expectedActivity   *SetNodePoolStatusActivity
		inputNodePoolStore eks.NodePoolStore
	}{
		{
			caseDescription: "not nil node pool store success",
			expectedActivity: &SetNodePoolStatusActivity{
				nodePoolStore: &eks.MockNodePoolStore{},
			},
			inputNodePoolStore: &eks.MockNodePoolStore{},
		},
		{
			caseDescription:    "nil node pool store success",
			expectedActivity:   &SetNodePoolStatusActivity{},
			inputNodePoolStore: nil,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualActivity := NewSetNodePoolStatusActivity(testCase.inputNodePoolStore)

			require.Equal(t, testCase.expectedActivity, actualActivity)
		})
	}
}

func TestSetNodePoolStatusActivityExecute(t *testing.T) {
	type inputType struct {
		activity *SetNodePoolStatusActivity
		input    SetNodePoolStatusActivityInput
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				activity: NewSetNodePoolStatusActivity(&eks.MockNodePoolStore{}),
				input: SetNodePoolStatusActivityInput{
					ClusterID:             1,
					ClusterName:           "cluster-name",
					NodePoolName:          "node-pool-name",
					NodePoolStatus:        eks.NodePoolStatusCreating,
					NodePoolStatusMessage: "node-pool-status-message",
					OrganizationID:        2,
				},
			},
		},
		{
			caseDescription: "error",
			expectedError:   errors.New("test error"),
			input: inputType{
				activity: NewSetNodePoolStatusActivity(&eks.MockNodePoolStore{}),
				input: SetNodePoolStatusActivityInput{
					ClusterID:             1,
					ClusterName:           "cluster-name",
					NodePoolName:          "node-pool-name",
					NodePoolStatus:        eks.NodePoolStatusCreating,
					NodePoolStatusMessage: "node-pool-status-message",
					OrganizationID:        2,
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			mockNodePoolStore := testCase.input.activity.nodePoolStore.(*eks.MockNodePoolStore)
			mockNodePoolStore.On(
				"UpdateNodePoolStatus",
				mock.Anything,
				testCase.input.input.OrganizationID,
				testCase.input.input.ClusterID,
				testCase.input.input.ClusterName,
				testCase.input.input.NodePoolName,
				testCase.input.input.NodePoolStatus,
				testCase.input.input.NodePoolStatusMessage,
			).Return(testCase.expectedError)

			actualError := testCase.input.activity.Execute(nil, testCase.input.input)

			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}

func TestSetNodePoolErrorStatus(t *testing.T) {
	type inputType struct {
		clusterID      uint
		clusterName    string
		nodePoolName   string
		organizationID uint
		statusError    error
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				clusterID:      1,
				clusterName:    "cluster-name",
				nodePoolName:   "node-pool-name",
				organizationID: 2,
				statusError:    errors.New("test status error"),
			},
		},
		{
			caseDescription: "error",
			expectedError:   errors.New("test error"),
			input: inputType{
				clusterID:      1,
				clusterName:    "cluster-name",
				nodePoolName:   "node-pool-name",
				organizationID: 2,
				statusError:    nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input SetNodePoolStatusActivityInput) error {
					return testCase.expectedError
				},
				activity.RegisterOptions{Name: SetNodePoolStatusActivityName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualError = setNodePoolErrorStatus(
					ctx,
					testCase.input.organizationID,
					testCase.input.clusterID,
					testCase.input.clusterName,
					testCase.input.nodePoolName,
					testCase.input.statusError,
				)

				return nil
			})

			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}

func TestSetNodePoolStatus(t *testing.T) {
	type inputType struct {
		clusterID             uint
		clusterName           string
		nodePoolName          string
		nodePoolStatus        eks.NodePoolStatus
		nodePoolStatusMessage string
		organizationID        uint
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				clusterID:             1,
				clusterName:           "cluster-name",
				nodePoolName:          "node-pool-name",
				nodePoolStatus:        eks.NodePoolStatusCreating,
				nodePoolStatusMessage: "node-pool-status-message",
				organizationID:        2,
			},
		},
		{
			caseDescription: "error",
			expectedError:   errors.New("test error"),
			input: inputType{
				clusterID:             1,
				clusterName:           "cluster-name",
				nodePoolName:          "node-pool-name",
				nodePoolStatus:        eks.NodePoolStatusCreating,
				nodePoolStatusMessage: "node-pool-status-message",
				organizationID:        2,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input SetNodePoolStatusActivityInput) error {
					return testCase.expectedError
				},
				activity.RegisterOptions{Name: SetNodePoolStatusActivityName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualError = setNodePoolStatus(
					ctx,
					testCase.input.organizationID,
					testCase.input.clusterID,
					testCase.input.clusterName,
					testCase.input.nodePoolName,
					testCase.input.nodePoolStatus,
					testCase.input.nodePoolStatusMessage,
				)

				return nil
			})

			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}

func TestSetNodePoolStatusAsync(t *testing.T) {
	type inputType struct {
		clusterID             uint
		clusterName           string
		nodePoolName          string
		nodePoolStatus        eks.NodePoolStatus
		nodePoolStatusMessage string
		organizationID        uint
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				clusterID:             1,
				clusterName:           "cluster-name",
				nodePoolName:          "node-pool-name",
				nodePoolStatus:        eks.NodePoolStatusCreating,
				nodePoolStatusMessage: "node-pool-status-message",
				organizationID:        2,
			},
		},
		{
			caseDescription: "error",
			expectedError:   errors.New("test error"),
			input: inputType{
				clusterID:             1,
				clusterName:           "cluster-name",
				nodePoolName:          "node-pool-name",
				nodePoolStatus:        eks.NodePoolStatusCreating,
				nodePoolStatusMessage: "node-pool-status-message",
				organizationID:        2,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input SetNodePoolStatusActivityInput) error {
					return testCase.expectedError
				},
				activity.RegisterOptions{
					Name: SetNodePoolStatusActivityName,
				},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualFuture := setNodePoolStatusAsync(
					ctx,
					testCase.input.organizationID,
					testCase.input.clusterID,
					testCase.input.clusterName,
					testCase.input.nodePoolName,
					testCase.input.nodePoolStatus,
					testCase.input.nodePoolStatusMessage,
				)
				actualError = actualFuture.Get(ctx, nil)

				return nil
			})

			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, testCase.expectedError, actualError.Error())
			}
		})
	}
}
