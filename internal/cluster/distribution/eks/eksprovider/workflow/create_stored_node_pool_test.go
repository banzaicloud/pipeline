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

func TestNewCreateStoredNodePoolActivity(t *testing.T) {
	testCases := []struct {
		caseDescription    string
		expectedActivity   *CreateStoredNodePoolActivity
		inputNodePoolStore eks.NodePoolStore
	}{
		{
			caseDescription: "not nil node pool store -> success",
			expectedActivity: &CreateStoredNodePoolActivity{
				nodePoolStore: &eks.MockNodePoolStore{},
			},
			inputNodePoolStore: &eks.MockNodePoolStore{},
		},
		{
			caseDescription:    "nil node pool store -> success",
			expectedActivity:   &CreateStoredNodePoolActivity{},
			inputNodePoolStore: nil,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualActivity := NewCreateStoredNodePoolActivity(testCase.inputNodePoolStore)

			require.Equal(t, testCase.expectedActivity, actualActivity)
		})
	}
}

func TestCreateStoredNodePoolActivityExecute(t *testing.T) {
	type inputType struct {
		activity *CreateStoredNodePoolActivity
		input    CreateStoredNodePoolActivityInput
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
				activity: NewCreateStoredNodePoolActivity(&eks.MockNodePoolStore{}),
				input: CreateStoredNodePoolActivityInput{
					ClusterID:      1,
					ClusterName:    "cluster-name",
					NodePool:       eks.NewNodePool{},
					OrganizationID: 2,
					UserID:         3,
				},
			},
		},
		{
			caseDescription: "error",
			expectedError:   errors.New("test error"),
			input: inputType{
				activity: NewCreateStoredNodePoolActivity(&eks.MockNodePoolStore{}),
				input: CreateStoredNodePoolActivityInput{
					ClusterID:      1,
					ClusterName:    "cluster-name",
					NodePool:       eks.NewNodePool{},
					OrganizationID: 2,
					UserID:         3,
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			mockNodePoolStore := testCase.input.activity.nodePoolStore.(*eks.MockNodePoolStore)
			mockNodePoolStore.On(
				"CreateNodePool",
				mock.Anything,
				testCase.input.input.OrganizationID,
				testCase.input.input.ClusterID,
				testCase.input.input.ClusterName,
				testCase.input.input.UserID,
				testCase.input.input.NodePool,
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

func TestCreateStoredNodePool(t *testing.T) {
	type inputType struct {
		clusterID      uint
		clusterName    string
		nodePool       eks.NewNodePool
		organizationID uint
		userID         uint
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
				nodePool:       eks.NewNodePool{},
				organizationID: 2,
				userID:         3,
			},
		},
		{
			caseDescription: "error",
			expectedError:   errors.New("test error"),
			input: inputType{
				clusterID:      1,
				clusterName:    "cluster-name",
				nodePool:       eks.NewNodePool{},
				organizationID: 2,
				userID:         3,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input CreateStoredNodePoolActivityInput) error {
					return testCase.expectedError
				},
				activity.RegisterOptions{Name: CreateStoredNodePoolActivityName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualError = createStoredNodePool(
					ctx,
					testCase.input.organizationID,
					testCase.input.clusterID,
					testCase.input.clusterName,
					testCase.input.userID,
					testCase.input.nodePool,
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

func TestCreateStoredNodePoolAsync(t *testing.T) {
	type inputType struct {
		clusterID      uint
		clusterName    string
		nodePool       eks.NewNodePool
		organizationID uint
		userID         uint
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
				nodePool:       eks.NewNodePool{},
				organizationID: 2,
				userID:         3,
			},
		},
		{
			caseDescription: "error",
			expectedError:   errors.New("test error"),
			input: inputType{
				clusterID:      1,
				clusterName:    "cluster-name",
				nodePool:       eks.NewNodePool{},
				organizationID: 2,
				userID:         3,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input CreateStoredNodePoolActivityInput) error {
					return testCase.expectedError
				},
				activity.RegisterOptions{Name: CreateStoredNodePoolActivityName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualFuture := createStoredNodePoolAsync(
					ctx,
					testCase.input.organizationID,
					testCase.input.clusterID,
					testCase.input.clusterName,
					testCase.input.userID,
					testCase.input.nodePool,
				)
				actualError = actualFuture.Get(ctx, nil)

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
