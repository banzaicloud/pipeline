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

package pkeawsadapter

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/mocks"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsprovider/workflow"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
)

func TestNodePoolManagerDeleteNodePool(t *testing.T) {
	type inputType struct {
		c                         cluster.Cluster
		existingNodePool          pke.ExistingNodePool
		manager                   nodePoolManager
		shouldUpdateClusterStatus bool
	}

	testCases := []struct {
		caseName      string
		expectedError error
		input         inputType
		mockError     error
	}{
		{
			caseName:      "error",
			expectedError: errors.New("failed to start workflow: test error"),
			input: inputType{
				manager: nodePoolManager{
					workflowClient: &mocks.Client{},
				},
			},
			mockError: errors.New("test error"),
		},
		{
			caseName:      "success",
			expectedError: nil,
			input: inputType{
				c: cluster.Cluster{
					ID:             uint(1),
					Location:       "region",
					Name:           "cluster-name",
					OrganizationID: uint(2),
					SecretID: func() brn.ResourceName {
						secretID, err := brn.Parse("brn:2:secret:secret-id")
						require.NoError(t, err)

						return secretID
					}(),
				},
				existingNodePool: pke.ExistingNodePool{
					Name: "node-pool-name",
				},
				manager: nodePoolManager{
					workflowClient: &mocks.Client{},
				},
				shouldUpdateClusterStatus: true,
			},
			mockError: nil,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			testCase.input.manager.workflowClient.(*mocks.Client).On(
				"StartWorkflow",
				context.Background(),
				mock.Anything,
				workflow.DeleteNodePoolWorkflowName,
				workflow.DeleteNodePoolWorkflowInput{
					ClusterID:                 testCase.input.c.ID,
					ClusterName:               testCase.input.c.Name,
					NodePoolName:              testCase.input.existingNodePool.Name,
					OrganizationID:            testCase.input.c.OrganizationID,
					Region:                    testCase.input.c.Location,
					SecretID:                  testCase.input.c.SecretID.ResourceID,
					ShouldUpdateClusterStatus: testCase.input.shouldUpdateClusterStatus,
				},
			).Return(nil, testCase.mockError)

			actualError := testCase.input.manager.DeleteNodePool(
				context.Background(),
				testCase.input.c,
				testCase.input.existingNodePool,
				testCase.input.shouldUpdateClusterStatus,
			)

			if testCase.expectedError == nil {
				require.Nil(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}

			testCase.input.manager.workflowClient.(*mocks.Client).AssertExpectations(t)
		})
	}
}
