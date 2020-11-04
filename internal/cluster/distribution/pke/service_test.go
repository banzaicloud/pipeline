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

package pke

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

func TestServiceDeleteNodePool(t *testing.T) {
	type inputType struct {
		clusterID    uint
		nodePoolName string
		s            service
	}

	type intermediateDataType struct {
		isExisting bool
	}

	type outputType struct {
		expectedError     error
		expectedIsDeleted bool
	}

	mockMethods := func(
		t *testing.T,
		input inputType,
		intermediateData intermediateDataType,
		mockErrors map[string]error,
	) {
		if mockErrors == nil {
			mockErrors = map[string]error{} // Note: defaulting to nil errors.
		}

		mocks := []string{
			"Store.GetCluster",
			"NodePoolStore.ListNodePools",
			"Store.SetStatus",
			"NodePoolManager.DeleteNodePool",
		}

		c := cluster.Cluster{ID: input.clusterID}
		existingNodePool := ExistingNodePool{
			Name: input.nodePoolName,
		}

		previousMockCounts := make(map[string]int, len(mocks))
		for _, mockID := range mocks {
			switch mockID {
			case "NodePoolManager.DeleteNodePool":
				input.s.nodePoolManager.(*MockNodePoolManager).On(
					"DeleteNodePool", mock.Anything, c, existingNodePool, true,
				).Return(mockErrors[mockID]).Once()
			case "NodePoolStore.ListNodePools":
				mock := input.s.nodePoolStore.(*MockNodePoolStore).On(
					"ListNodePools", mock.Anything, mock.Anything, input.clusterID, mock.Anything,
				)

				err := mockErrors[mockID]
				if err == nil {
					nodePools := map[string]ExistingNodePool{}
					if intermediateData.isExisting {
						nodePools[input.nodePoolName] = existingNodePool
					}

					mock = mock.Return(nodePools, nil)
				} else {
					mock = mock.Return(nil, mockErrors[mockID])
				}

				mock.Once()
			case "Store.GetCluster":
				input.s.genericClusters.(*MockStore).On(
					"GetCluster", mock.Anything, input.clusterID,
				).Return(c, mockErrors[mockID]).Once()
			case "Store.SetStatus":
				input.s.genericClusters.(*MockStore).On(
					"SetStatus", mock.Anything, input.clusterID, cluster.Updating, "deleting node pool",
				).Return(mockErrors[mockID]).Once()
			default:
				t.Errorf(
					"unexpected mock call, no mock method is available for this mock ID,"+
						" mock ID: '%s', ordered mock ID occurrences: '%+v'",
					mockID, mocks,
				)
				t.FailNow()
				return
			}

			previousMockCounts[mockID] += 1
		}
	}

	testCases := []struct {
		caseName         string
		input            inputType
		intermediateData intermediateDataType
		mockErrors       map[string]error
		output           outputType
	}{
		{
			caseName: "GetCluster error",
			input: inputType{
				clusterID:    1,
				nodePoolName: "node-pool-name",
				s: service{
					genericClusters: &MockStore{},
					nodePoolManager: &MockNodePoolManager{},
					nodePoolStore:   &MockNodePoolStore{},
				},
			},
			intermediateData: intermediateDataType{},
			mockErrors: map[string]error{
				"Store.GetCluster": errors.New("test error: GetCluster"),
			},
			output: outputType{
				expectedError:     errors.New("test error: GetCluster"),
				expectedIsDeleted: false,
			},
		},
		{
			caseName: "ListNodePools error",
			input: inputType{
				clusterID:    1,
				nodePoolName: "node-pool-name",
				s: service{
					genericClusters: &MockStore{},
					nodePoolManager: &MockNodePoolManager{},
					nodePoolStore:   &MockNodePoolStore{},
				},
			},
			intermediateData: intermediateDataType{},
			mockErrors: map[string]error{
				"NodePoolStore.ListNodePools": errors.New("test error: ListNodePools"),
			},
			output: outputType{
				expectedError:     errors.New("test error: ListNodePools"),
				expectedIsDeleted: false,
			},
		},
		{
			caseName: "already deleted success",
			input: inputType{
				clusterID:    1,
				nodePoolName: "node-pool-name",
				s: service{
					genericClusters: &MockStore{},
					nodePoolManager: &MockNodePoolManager{},
					nodePoolStore:   &MockNodePoolStore{},
				},
			},
			intermediateData: intermediateDataType{
				isExisting: false,
			},
			mockErrors: map[string]error{},
			output: outputType{
				expectedError:     nil,
				expectedIsDeleted: true,
			},
		},
		{
			caseName: "SetStatus error",
			input: inputType{
				clusterID:    1,
				nodePoolName: "node-pool-name",
				s: service{
					genericClusters: &MockStore{},
					nodePoolManager: &MockNodePoolManager{},
					nodePoolStore:   &MockNodePoolStore{},
				},
			},
			intermediateData: intermediateDataType{
				isExisting: true,
			},
			mockErrors: map[string]error{
				"Store.SetStatus": errors.New("test error: SetStatus"),
			},
			output: outputType{
				expectedError:     errors.New("test error: SetStatus"),
				expectedIsDeleted: false,
			},
		},
		{
			caseName: "DeleteNodePool error",
			input: inputType{
				clusterID:    1,
				nodePoolName: "node-pool-name",
				s: service{
					genericClusters: &MockStore{},
					nodePoolManager: &MockNodePoolManager{},
					nodePoolStore:   &MockNodePoolStore{},
				},
			},
			intermediateData: intermediateDataType{
				isExisting: true,
			},
			mockErrors: map[string]error{
				"NodePoolManager.DeleteNodePool": errors.New("test error: DeleteNodePool"),
			},
			output: outputType{
				expectedError:     errors.New("test error: DeleteNodePool"),
				expectedIsDeleted: false,
			},
		},
		{
			caseName: "existing delete started success",
			input: inputType{
				clusterID:    1,
				nodePoolName: "node-pool-name",
				s: service{
					genericClusters: &MockStore{},
					nodePoolManager: &MockNodePoolManager{},
					nodePoolStore:   &MockNodePoolStore{},
				},
			},
			intermediateData: intermediateDataType{
				isExisting: true,
			},
			mockErrors: map[string]error{},
			output: outputType{
				expectedError:     nil,
				expectedIsDeleted: false,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			mockMethods(t, testCase.input, testCase.intermediateData, testCase.mockErrors)

			actualIsDeleted, actualError := testCase.input.s.DeleteNodePool(
				context.Background(),
				testCase.input.clusterID,
				testCase.input.nodePoolName,
			)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedIsDeleted, actualIsDeleted)
		})
	}
}
