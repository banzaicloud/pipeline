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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
)

func TestListStoredNodePoolsActivityExecute(t *testing.T) {
	type caseInputType struct {
		input         ListStoredNodePoolsActivityInput
		inputActivity *ListStoredNodePoolsActivity
	}

	type caseOutputType struct {
		expectedError  error
		expectedOutput *ListStoredNodePoolsActivityOutput
	}

	nodePoolStoreListNodePoolsID := "NodePoolStore.ListNodePools"

	mockCalls := func(
		t *testing.T, caseInput caseInputType, caseMockData map[string][]interface{}, caseMockErrors map[string]error,
	) {
		if caseMockData == nil {
			caseMockData = map[string][]interface{}{} // NotE: using nil output values.
		}
		if caseMockErrors == nil {
			caseMockErrors = map[string]error{} // NotE: using nil mock errors.
		}

		mocks := []string{
			nodePoolStoreListNodePoolsID,
		}

		previousMockCalls := make(map[string]int, len(mocks))
		for _, mockID := range mocks {
			switch mockID {
			case nodePoolStoreListNodePoolsID:
				mock := caseInput.inputActivity.nodePoolStore.(*eks.MockNodePoolStore).On(
					"ListNodePools",
					mock.Anything,
					caseInput.input.OrganizationID,
					caseInput.input.ClusterID,
					caseInput.input.ClusterName,
				)

				err := caseMockErrors[mockID]
				if err == nil {
					mock.Return(append(caseMockData[mockID], nil)...).Once()
				} else {
					mock.Return(nil, caseMockErrors[mockID]).Once()
				}
			default:
				t.Errorf(
					"unexpected mock call, no mock method is available for this mock ID,"+
						" mock ID: '%s', ordered mock ID occurrences: '%+v'",
					mockID, mocks,
				)
				t.FailNow()

				return
			}

			if caseMockErrors[mockID] != nil {
				return
			}

			previousMockCalls[mockID] += 1
		}
	}

	testCases := []struct {
		caseDescription string
		caseInput       caseInputType
		caseOutput      caseOutputType
		caseMockData    map[string][]interface{}
		caseMockErrors  map[string]error
	}{
		{
			caseDescription: nodePoolStoreListNodePoolsID + " error",
			caseInput: caseInputType{
				input: ListStoredNodePoolsActivityInput{},
				inputActivity: &ListStoredNodePoolsActivity{
					nodePoolStore: &eks.MockNodePoolStore{},
				},
			},
			caseOutput: caseOutputType{
				expectedError:  errors.New("test error: " + nodePoolStoreListNodePoolsID),
				expectedOutput: nil,
			},
			caseMockData: map[string][]interface{}{
				nodePoolStoreListNodePoolsID: {
					nil,
				},
			},
			caseMockErrors: map[string]error{
				nodePoolStoreListNodePoolsID: errors.New("test error: " + nodePoolStoreListNodePoolsID),
			},
		},
		{
			caseDescription: "not found error",
			caseInput: caseInputType{
				input: ListStoredNodePoolsActivityInput{
					OptionalListedNodePoolNames: []string{
						"node-pool-1",
						"node-pool-2",
					},
				},
				inputActivity: &ListStoredNodePoolsActivity{
					nodePoolStore: &eks.MockNodePoolStore{},
				},
			},
			caseOutput: caseOutputType{
				expectedError:  errors.New("node pool node-pool-1 not found; node pool node-pool-2 not found"),
				expectedOutput: nil,
			},
			caseMockData: map[string][]interface{}{
				nodePoolStoreListNodePoolsID: {
					nil,
				},
			},
		},
		{
			caseDescription: "unfiltered success",
			caseInput: caseInputType{
				input: ListStoredNodePoolsActivityInput{
					OptionalListedNodePoolNames: nil,
				},
				inputActivity: &ListStoredNodePoolsActivity{
					nodePoolStore: &eks.MockNodePoolStore{},
				},
			},
			caseOutput: caseOutputType{
				expectedError: nil,
				expectedOutput: &ListStoredNodePoolsActivityOutput{
					NodePools: map[string]eks.ExistingNodePool{
						"pool1": {Name: "pool1"},
						"pool2": {Name: "pool2"},
						"pool3": {Name: "pool3"},
						"pool4": {Name: "pool4"},
						"pool5": {Name: "pool5"},
					},
				},
			},
			caseMockData: map[string][]interface{}{
				nodePoolStoreListNodePoolsID: {
					map[string]eks.ExistingNodePool{
						"pool1": {Name: "pool1"},
						"pool2": {Name: "pool2"},
						"pool3": {Name: "pool3"},
						"pool4": {Name: "pool4"},
						"pool5": {Name: "pool5"},
					},
				},
			},
		},
		{
			caseDescription: "filtered success",
			caseInput: caseInputType{
				input: ListStoredNodePoolsActivityInput{
					OptionalListedNodePoolNames: []string{
						"pool5",
						"pool3",
						"pool1",
					},
				},
				inputActivity: &ListStoredNodePoolsActivity{
					nodePoolStore: &eks.MockNodePoolStore{},
				},
			},
			caseOutput: caseOutputType{
				expectedError: nil,
				expectedOutput: &ListStoredNodePoolsActivityOutput{
					NodePools: map[string]eks.ExistingNodePool{
						"pool1": {Name: "pool1"},
						"pool3": {Name: "pool3"},
						"pool5": {Name: "pool5"},
					},
				},
			},
			caseMockData: map[string][]interface{}{
				nodePoolStoreListNodePoolsID: {
					map[string]eks.ExistingNodePool{
						"pool1": {Name: "pool1"},
						"pool2": {Name: "pool2"},
						"pool3": {Name: "pool3"},
						"pool4": {Name: "pool4"},
						"pool5": {Name: "pool5"},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			environment := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()
			environment.RegisterActivityWithOptions(
				testCase.caseInput.inputActivity.Execute,
				activity.RegisterOptions{
					Name: t.Name(),
				},
			)

			mockCalls(t, testCase.caseInput, testCase.caseMockData, testCase.caseMockErrors)

			actualValue, actualError := environment.ExecuteActivity(t.Name(), testCase.caseInput.input)
			var actualOutput *ListStoredNodePoolsActivityOutput
			if actualValue != nil &&
				actualValue.HasValue() {
				err := actualValue.Get(&actualOutput)
				require.NoError(t, err)
			}

			if testCase.caseOutput.expectedError == nil {
				require.Nil(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.caseOutput.expectedError.Error())
			}
			require.Equal(t, testCase.caseOutput.expectedOutput, actualOutput)
		})
	}
}

func TestNewListStoredNodePoolsActivity(t *testing.T) {
	testCases := []struct {
		caseName           string
		expectedActivity   *ListStoredNodePoolsActivity
		inputNodePoolStore eks.NodePoolStore
	}{
		{
			caseName:           "nil node pool store",
			expectedActivity:   &ListStoredNodePoolsActivity{},
			inputNodePoolStore: nil,
		},
		{
			caseName: "not nil node pool store",
			expectedActivity: &ListStoredNodePoolsActivity{
				nodePoolStore: &eks.MockNodePoolStore{},
			},
			inputNodePoolStore: &eks.MockNodePoolStore{},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualActivity := NewListStoredNodePoolsActivity(testCase.inputNodePoolStore)

			require.Equal(t, testCase.expectedActivity, actualActivity)
		})
	}
}
