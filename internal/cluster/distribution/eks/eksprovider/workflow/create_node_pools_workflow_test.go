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

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
)

func TestCreateNodePoolsWorkflowExecute(t *testing.T) {
	t.Parallel()

	type inputType struct {
		workflow *CreateNodePoolsWorkflow
		input    CreateNodePoolsWorkflowInput
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "create node pool error -> error",
			expectedError:   errors.New("creating node pool failed, nodePool: node-pool-2-error: create node pool error"),
			input: inputType{
				workflow: NewCreateNodePoolsWorkflow(),
				input: CreateNodePoolsWorkflowInput{
					ClusterID:     1,
					CreatorUserID: 2,
					NodePools: map[string]eks.NewNodePool{
						"node-pool-1": {
							Name:     "node-pool-1",
							SubnetID: "subnet-id-1",
						},
						"node-pool-2-error": {
							Name:     "node-pool-2-error",
							SubnetID: "subnet-id-2",
						},
						"node-pool-3": {
							Name:     "node-pool-3",
							SubnetID: "subnet-id-3",
						},
					},
					NodePoolSubnetIDs: map[string][]string{
						"node-pool-1": {"subnet-id-1"},
						"node-pool-2": {"subnet-id-2"},
						"node-pool-3": {"subnet-id-3"},
					},
					ShouldCreateNodePoolLabelSet: true,
					ShouldStoreNodePool:          true,
					ShouldUpdateClusterStatus:    true,
				},
			},
		},
		{
			caseDescription: "set cluster status error -> error",
			expectedError:   errors.New("set cluster status error"),
			input: inputType{
				workflow: NewCreateNodePoolsWorkflow(),
				input: CreateNodePoolsWorkflowInput{
					ClusterID:     1,
					CreatorUserID: 2,
					NodePools: map[string]eks.NewNodePool{
						"node-pool-1": {
							Name:     "node-pool-1",
							SubnetID: "subnet-id-1",
						},
						"node-pool-2": {
							Name:     "node-pool-2",
							SubnetID: "subnet-id-2",
						},
						"node-pool-3": {
							Name:     "node-pool-3",
							SubnetID: "subnet-id-3",
						},
					},
					NodePoolSubnetIDs: map[string][]string{
						"node-pool-1": {"subnet-id-1"},
						"node-pool-2": {"subnet-id-2"},
						"node-pool-3": {"subnet-id-3"},
					},
					ShouldCreateNodePoolLabelSet: true,
					ShouldStoreNodePool:          true,
					ShouldUpdateClusterStatus:    true,
				},
			},
		},
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				workflow: NewCreateNodePoolsWorkflow(),
				input: CreateNodePoolsWorkflowInput{
					ClusterID:     1,
					CreatorUserID: 2,
					NodePools: map[string]eks.NewNodePool{
						"node-pool-1": {
							Name:     "node-pool-1",
							SubnetID: "subnet-id-1",
						},
						"node-pool-2": {
							Name:     "node-pool-2",
							SubnetID: "subnet-id-2",
						},
						"node-pool-3": {
							Name:     "node-pool-3",
							SubnetID: "subnet-id-3",
						},
					},
					NodePoolSubnetIDs: map[string][]string{
						"node-pool-1": {"subnet-id-1"},
						"node-pool-2": {"subnet-id-2"},
						"node-pool-3": {"subnet-id-3"},
					},
					ShouldCreateNodePoolLabelSet: true,
					ShouldStoreNodePool:          true,
					ShouldUpdateClusterStatus:    true,
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			t.Parallel()

			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterWorkflowWithOptions(
				testCase.input.workflow.Execute,
				workflow.RegisterOptions{
					Name: t.Name(),
				},
			)

			environment.RegisterWorkflowWithOptions(
				func(ctx workflow.Context, input CreateNodePoolWorkflowInput) error {
					if testCase.expectedError != nil &&
						strings.HasSuffix(testCase.expectedError.Error(), "create node pool error") &&
						strings.Contains(testCase.expectedError.Error(), input.NodePool.Name) {
						return errors.Errorf("create node pool error")
					}

					return nil
				},
				workflow.RegisterOptions{
					Name: CreateNodePoolWorkflowName,
				},
			)

			if testCase.input.input.ShouldUpdateClusterStatus {
				environment.RegisterActivityWithOptions(
					func(ctx context.Context, input SetClusterStatusActivityInput) error {
						if testCase.expectedError != nil &&
							strings.HasPrefix(testCase.expectedError.Error(), "set cluster status error") {
							return testCase.expectedError
						}

						return nil
					},
					activity.RegisterOptions{
						Name: SetClusterStatusActivityName,
					},
				)
			}

			environment.ExecuteWorkflow(t.Name(), testCase.input.input)
			actualError := environment.GetWorkflowError()

			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}

func TestNewCreateNodePoolsWorkflow(t *testing.T) {
	t.Parallel()

	require.Equal(t, &CreateNodePoolsWorkflow{}, NewCreateNodePoolsWorkflow())
}
