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
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	pkgcadence "github.com/banzaicloud/pipeline/pkg/cadence"
	sdkamazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
)

type DeleteNodePoolWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	environment *testsuite.TestWorkflowEnvironment
}

func TestDeleteNodePoolWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteNodePoolWorkflowTestSuite))
}

func (workflowTestSuite *DeleteNodePoolWorkflowTestSuite) SetupTest() {
	workflowTestSuite.environment = workflowTestSuite.NewTestWorkflowEnvironment()

	deleteNodePoolLabelSetActivity := clusterworkflow.NewDeleteNodePoolLabelSetActivity(nil, "")
	workflowTestSuite.environment.RegisterActivityWithOptions(deleteNodePoolLabelSetActivity.Execute, activity.RegisterOptions{Name: clusterworkflow.DeleteNodePoolLabelSetActivityName})

	deleteStackActivity := awsworkflow.NewDeleteStackActivity(nil)
	workflowTestSuite.environment.RegisterActivityWithOptions(
		deleteStackActivity.Execute, activity.RegisterOptions{Name: awsworkflow.DeleteStackActivityName})

	deleteStoredNodePoolActivity := NewDeleteStoredNodePoolActivity(nil)
	workflowTestSuite.environment.RegisterActivityWithOptions(deleteStoredNodePoolActivity.Execute, activity.RegisterOptions{Name: DeleteStoredNodePoolActivityName})

	listStoredNodePoolsActivity := NewListStoredNodePoolsActivity(nil)
	workflowTestSuite.environment.RegisterActivityWithOptions(listStoredNodePoolsActivity.Execute, activity.RegisterOptions{Name: ListStoredNodePoolsActivityName})

	setClusterStatusActivity := clusterworkflow.NewSetClusterStatusActivity(nil)
	workflowTestSuite.environment.RegisterActivityWithOptions(setClusterStatusActivity.Execute, activity.RegisterOptions{Name: clusterworkflow.SetClusterStatusActivityName})
}

func (workflowTestSuite *DeleteNodePoolWorkflowTestSuite) AfterTest(suiteName, testName string) {
	workflowTestSuite.environment.AssertExpectations(workflowTestSuite.T())
}

func (workflowTestSuite *DeleteNodePoolWorkflowTestSuite) TestDeleteNodePoolWorkflowExecute() {
	type inputType struct {
		input    DeleteNodePoolWorkflowInput
		workflow *DeleteNodePoolWorkflow
	}

	type intermediateDataType struct {
		StackID string
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

		errorAttemptCount := 31

		mocks := make([]string, 0, 6)
		mocks = append(mocks, ListStoredNodePoolsActivityName)
		if input.input.ShouldUpdateClusterStatus &&
			mockErrors[ListStoredNodePoolsActivityName] != nil {
			mocks = append(mocks, clusterworkflow.SetClusterStatusActivityName)
		}
		mocks = append(mocks, awsworkflow.DeleteStackActivityName)
		if input.input.ShouldUpdateClusterStatus &&
			mockErrors[awsworkflow.DeleteStackActivityName] != nil {
			mocks = append(mocks, clusterworkflow.SetClusterStatusActivityName)
		}
		mocks = append(mocks, DeleteStoredNodePoolActivityName)
		if input.input.ShouldUpdateClusterStatus &&
			mockErrors[DeleteStoredNodePoolActivityName] != nil {
			mocks = append(mocks, clusterworkflow.SetClusterStatusActivityName)
		}
		mocks = append(mocks, clusterworkflow.DeleteNodePoolLabelSetActivityName)
		if input.input.ShouldUpdateClusterStatus &&
			mockErrors[clusterworkflow.DeleteNodePoolLabelSetActivityName] != nil {
			mocks = append(mocks, clusterworkflow.SetClusterStatusActivityName)
		}
		if input.input.ShouldUpdateClusterStatus { // Note: final cluster status setting.
			mocks = append(mocks, clusterworkflow.SetClusterStatusActivityName)
		}
		if input.input.ShouldUpdateClusterStatus &&
			mockErrors[clusterworkflow.SetClusterStatusActivityName] != nil {
			mocks = append(mocks, clusterworkflow.SetClusterStatusActivityName)
		}

		previousMockCounts := make(map[string]int, len(mocks))
		for mockIndex, mockID := range mocks {
			switch mockID {
			case clusterworkflow.DeleteNodePoolLabelSetActivityName:
				activityInput := clusterworkflow.DeleteNodePoolLabelSetActivityInput{
					ClusterID:    input.input.ClusterID,
					NodePoolName: input.input.NodePoolName,
				}

				attempts := 1
				if mockErrors[mockID] != nil {
					attempts = errorAttemptCount
				}

				workflowTestSuite.environment.OnActivity(mockID, mock.Anything, activityInput).
					Return(mockErrors[mockID]).Times(attempts)

				if mockErrors[mockID] != nil &&
					!input.input.ShouldUpdateClusterStatus {
					return
				}
			case awsworkflow.DeleteStackActivityName:
				activityInput := awsworkflow.DeleteStackActivityInput{
					AWSCommonActivityInput: awsworkflow.AWSCommonActivityInput{
						OrganizationID: input.input.OrganizationID,
						SecretID:       input.input.SecretID,
						Region:         input.input.Region,
						ClusterName:    input.input.ClusterName,
						AWSClientRequestTokenBase: sdkamazon.NewNormalizedClientRequestToken(
							"default-test-workflow-id",
						),
					},
					StackID: intermediateData.StackID,
					StackName: pkeaws.GenerateNodePoolStackName(
						input.input.ClusterName, input.input.NodePoolName,
					),
				}

				attempts := 1
				if mockErrors[mockID] != nil {
					attempts = errorAttemptCount
				}

				workflowTestSuite.environment.OnActivity(mockID, mock.Anything, activityInput).
					Return(mockErrors[mockID]).Times(attempts)

				if mockErrors[mockID] != nil &&
					!input.input.ShouldUpdateClusterStatus {
					return
				}
			case DeleteStoredNodePoolActivityName:
				activityInput := DeleteStoredNodePoolActivityInput{
					ClusterID:      input.input.ClusterID,
					ClusterName:    input.input.ClusterName,
					NodePoolName:   input.input.NodePoolName,
					OrganizationID: input.input.OrganizationID,
				}

				attempts := 1
				if mockErrors[mockID] != nil {
					attempts = errorAttemptCount
				}

				workflowTestSuite.environment.OnActivity(mockID, mock.Anything, activityInput).
					Return(mockErrors[mockID]).Times(attempts)

				if mockErrors[mockID] != nil &&
					!input.input.ShouldUpdateClusterStatus {
					return
				}
			case ListStoredNodePoolsActivityName:
				activityInput := ListStoredNodePoolsActivityInput{
					ClusterID:                   input.input.ClusterID,
					ClusterName:                 input.input.ClusterName,
					OptionalListedNodePoolNames: []string{input.input.NodePoolName},
					OrganizationID:              input.input.OrganizationID,
				}

				mock := workflowTestSuite.environment.OnActivity(mockID, mock.Anything, activityInput)

				err := mockErrors[mockID]
				if err == nil {
					mock.Return(
						&ListStoredNodePoolsActivityOutput{
							NodePools: map[string]pke.ExistingNodePool{
								input.input.NodePoolName: {
									Name: input.input.NodePoolName,
								},
							},
						},
						nil,
					).Once()
				} else {
					mock.Return(nil, err).Times(errorAttemptCount)
				}

				if mockErrors[mockID] != nil &&
					!input.input.ShouldUpdateClusterStatus {
					return
				}
			case clusterworkflow.SetClusterStatusActivityName:
				previousMockID := mocks[mockIndex-1] // Note: last mocked call returned the error to propagate.
				previousMockError := mockErrors[previousMockID]

				if previousMockError != nil &&
					input.input.ShouldUpdateClusterStatus { // Note: activity error handling branch.
					activityInput := clusterworkflow.SetClusterStatusActivityInput{
						ClusterID:     input.input.ClusterID,
						Status:        cluster.Warning,
						StatusMessage: pkgcadence.UnwrapError(previousMockError).Error(),
					}

					workflowTestSuite.environment.OnActivity(mockID, mock.Anything, activityInput).Return(nil).Once()

					return
				} else if input.input.ShouldUpdateClusterStatus { // Note: workflow result cluster status setting.
					activityInput := clusterworkflow.SetClusterStatusActivityInput{
						ClusterID:     input.input.ClusterID,
						Status:        cluster.Running,
						StatusMessage: cluster.RunningMessage,
					}

					attempts := 1
					if mockErrors[mockID] != nil {
						attempts = errorAttemptCount
					}

					workflowTestSuite.environment.OnActivity(mockID, mock.Anything, activityInput).
						Return(mockErrors[mockID]).Times(attempts)

					if mockErrors[mockID] != nil {
						return
					}
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

			previousMockCounts[mockID] += 1
		}
	}

	testCases := []struct {
		caseName         string
		expectedError    error
		input            inputType
		intermediateData intermediateDataType
		mockErrors       map[string]error
	}{
		{
			caseName:      "ListStoredNodePoolsActivity error",
			expectedError: errors.New("test error: ListStoredNodePoolsActivity"),
			input: inputType{
				input:    DeleteNodePoolWorkflowInput{},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
			mockErrors: map[string]error{
				ListStoredNodePoolsActivityName: errors.New("test error: ListStoredNodePoolsActivity"),
			},
		},
		{
			caseName:      "ListStoredNodePoolsActivity with cluster status update error",
			expectedError: errors.New("test error: ListStoredNodePoolsActivity"),
			input: inputType{
				input: DeleteNodePoolWorkflowInput{
					ShouldUpdateClusterStatus: true,
				},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
			mockErrors: map[string]error{
				ListStoredNodePoolsActivityName: errors.New("test error: ListStoredNodePoolsActivity"),
			},
		},
		{
			caseName:      "DeleteStackActivity error",
			expectedError: errors.New("test error: DeleteStackActivity"),
			input: inputType{
				input:    DeleteNodePoolWorkflowInput{},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
			mockErrors: map[string]error{
				awsworkflow.DeleteStackActivityName: errors.New("test error: DeleteStackActivity"),
			},
		},
		{
			caseName:      "DeleteStackActivity with cluster status update error",
			expectedError: errors.New("test error: DeleteStackActivity"),
			input: inputType{
				input: DeleteNodePoolWorkflowInput{
					ShouldUpdateClusterStatus: true,
				},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
			mockErrors: map[string]error{
				awsworkflow.DeleteStackActivityName: errors.New("test error: DeleteStackActivity"),
			},
		},
		{
			caseName:      "DeleteStoredNodePool error",
			expectedError: errors.New("test error: DeleteStoredNodePool"),
			input: inputType{
				input:    DeleteNodePoolWorkflowInput{},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
			mockErrors: map[string]error{
				DeleteStoredNodePoolActivityName: errors.New("test error: DeleteStoredNodePool"),
			},
		},
		{
			caseName:      "DeleteStoredNodePool with cluster status update error",
			expectedError: errors.New("test error: DeleteStoredNodePool"),
			input: inputType{
				input: DeleteNodePoolWorkflowInput{
					ShouldUpdateClusterStatus: true,
				},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
			mockErrors: map[string]error{
				DeleteStoredNodePoolActivityName: errors.New("test error: DeleteStoredNodePool"),
			},
		},
		{
			caseName:      "DeleteNodePoolLabelSetActivityInput error",
			expectedError: errors.New("test error: DeleteNodePoolLabelSetActivityInput"),
			input: inputType{
				input:    DeleteNodePoolWorkflowInput{},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
			mockErrors: map[string]error{
				clusterworkflow.DeleteNodePoolLabelSetActivityName: errors.New(
					"test error: DeleteNodePoolLabelSetActivityInput",
				),
			},
		},
		{
			caseName:      "DeleteNodePoolLabelSetActivityInput with cluster status update error",
			expectedError: errors.New("test error: DeleteNodePoolLabelSetActivityInput"),
			input: inputType{
				input: DeleteNodePoolWorkflowInput{
					ShouldUpdateClusterStatus: true,
				},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
			mockErrors: map[string]error{
				clusterworkflow.DeleteNodePoolLabelSetActivityName: errors.New(
					"test error: DeleteNodePoolLabelSetActivityInput",
				),
			},
		},
		{
			caseName:      "SetClusterStatus with cluster status update error",
			expectedError: errors.New("test error: SetClusterStatus"),
			input: inputType{
				input: DeleteNodePoolWorkflowInput{
					ShouldUpdateClusterStatus: true,
				},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
			mockErrors: map[string]error{
				clusterworkflow.SetClusterStatusActivityName: errors.New("test error: SetClusterStatus"),
			},
		},
		{
			caseName:      "no cluster status update success",
			expectedError: nil,
			input: inputType{
				input:    DeleteNodePoolWorkflowInput{},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
		},
		{
			caseName:      "cluster status update success",
			expectedError: nil,
			input: inputType{
				input: DeleteNodePoolWorkflowInput{
					ShouldUpdateClusterStatus: true,
				},
				workflow: &DeleteNodePoolWorkflow{},
			},
			intermediateData: intermediateDataType{
				StackID: "stack-id",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		workflowTestSuite.SetupTest()

		workflowTestSuite.T().Run(testCase.caseName, func(t *testing.T) {
			workflow.RegisterWithOptions(testCase.input.workflow.Execute, workflow.RegisterOptions{Name: t.Name()})
			mockMethods(t, testCase.input, testCase.intermediateData, testCase.mockErrors)

			workflowTestSuite.environment.ExecuteWorkflow(t.Name(), testCase.input.input)
			workflowTestSuite.environment.CancelWorkflow()
			actualError := workflowTestSuite.environment.GetWorkflowError()

			require.True(t, workflowTestSuite.environment.IsWorkflowCompleted(), "the workflow has not completed")
			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})

		workflowTestSuite.AfterTest(workflowTestSuite.T().Name(), testCase.caseName)
	}
}

func (workflowTestSuite *DeleteNodePoolWorkflowTestSuite) TestDeleteNodePoolWorkflowRegister() {
	testCases := []struct {
		caseName      string
		inputWorkflow *DeleteNodePoolWorkflow
	}{
		{
			caseName:      "example",
			inputWorkflow: &DeleteNodePoolWorkflow{},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		workflowTestSuite.SetupTest()

		workflowTestSuite.T().Run(testCase.caseName, func(t *testing.T) {
			isHit := false
			workflowTestSuite.environment.OnWorkflow(DeleteNodePoolWorkflowName, mock.Anything, mock.Anything).
				Run(func(mock.Arguments) {
					isHit = true
				}).
				Return(nil)

			testCase.inputWorkflow.Register()
			workflowTestSuite.environment.ExecuteWorkflow(DeleteNodePoolWorkflowName, DeleteNodePoolWorkflowInput{})
			actualError := workflowTestSuite.environment.GetWorkflowError()

			require.True(t, workflowTestSuite.environment.IsWorkflowCompleted(), "the workflow has not completed")
			require.NoError(t, actualError)
			require.Equal(t, true, isHit)
		})

		workflowTestSuite.AfterTest(workflowTestSuite.T().Name(), testCase.caseName)
	}
}

func TestNewDeleteNodePoolWorkflow(t *testing.T) {
	testCases := []struct {
		caseName         string
		expectedWorkflow *DeleteNodePoolWorkflow
	}{
		{
			caseName:         "example",
			expectedWorkflow: &DeleteNodePoolWorkflow{},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualWorkflow := NewDeleteNodePoolWorkflow()

			require.Equal(t, testCase.expectedWorkflow, actualWorkflow)
		})
	}
}
