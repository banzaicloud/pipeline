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
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
)

func TestCreateNodePool(t *testing.T) {
	type inputType struct {
		clusterID                 uint
		userID                    uint
		nodePool                  eks.NewNodePool
		shouldStoreNodePool       bool
		shouldUpdateClusterStatus bool
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
				clusterID:                 1,
				userID:                    2,
				nodePool:                  eks.NewNodePool{},
				shouldStoreNodePool:       true,
				shouldUpdateClusterStatus: true,
			},
		},
		{
			caseDescription: "error",
			expectedError:   errors.New("test error"),
			input: inputType{
				clusterID:                 1,
				userID:                    2,
				nodePool:                  eks.NewNodePool{},
				shouldStoreNodePool:       true,
				shouldUpdateClusterStatus: true,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterWorkflowWithOptions(
				func(ctx workflow.Context, input CreateNodePoolWorkflowInput) error {
					return testCase.expectedError
				},
				workflow.RegisterOptions{Name: CreateNodePoolWorkflowName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
					ExecutionStartToCloseTimeout: 10 * time.Second,
					TaskStartToCloseTimeout:      3 * time.Second,
					WaitForCancellation:          true,
				})

				actualError = createNodePool(
					ctx,
					testCase.input.clusterID,
					testCase.input.userID,
					testCase.input.nodePool,
					testCase.input.shouldStoreNodePool,
					testCase.input.shouldUpdateClusterStatus,
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

func TestCreateNodePoolAsync(t *testing.T) {
	type inputType struct {
		clusterID                 uint
		userID                    uint
		nodePool                  eks.NewNodePool
		shouldStoreNodePool       bool
		shouldUpdateClusterStatus bool
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
				clusterID:                 1,
				userID:                    2,
				nodePool:                  eks.NewNodePool{},
				shouldStoreNodePool:       true,
				shouldUpdateClusterStatus: true,
			},
		},
		{
			caseDescription: "error",
			expectedError:   errors.New("test error"),
			input: inputType{
				clusterID:                 1,
				userID:                    2,
				nodePool:                  eks.NewNodePool{},
				shouldStoreNodePool:       true,
				shouldUpdateClusterStatus: true,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterWorkflowWithOptions(
				func(ctx workflow.Context, input CreateNodePoolWorkflowInput) error {
					return testCase.expectedError
				},
				workflow.RegisterOptions{Name: CreateNodePoolWorkflowName},
			)

			var actualError error
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
					ExecutionStartToCloseTimeout: 10 * time.Second,
					TaskStartToCloseTimeout:      3 * time.Second,
					WaitForCancellation:          true,
				})

				actualFuture := createNodePoolAsync(
					ctx,
					testCase.input.clusterID,
					testCase.input.userID,
					testCase.input.nodePool,
					testCase.input.shouldStoreNodePool,
					testCase.input.shouldUpdateClusterStatus,
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

func TestCreateNodePoolWorkflowExecute(t *testing.T) {
	type inputType struct {
		workflow *CreateNodePoolWorkflow
		input    CreateNodePoolWorkflowInput
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "list stored EKS clusters error -> error",
			expectedError:   errors.New("list stored EKS clusters error"),
			input: inputType{
				workflow: NewCreateNodePoolWorkflow(),
				input: CreateNodePoolWorkflowInput{
					ClusterID:                 1,
					NodePool:                  eks.NewNodePool{},
					ShouldStoreNodePool:       true,
					ShouldUpdateClusterStatus: true,
					UserID:                    2,
				},
			},
		},
		{
			caseDescription: "create stored node pool error -> error",
			expectedError:   errors.New("create stored node pool error"),
			input: inputType{
				workflow: NewCreateNodePoolWorkflow(),
				input: CreateNodePoolWorkflowInput{
					ClusterID:                 1,
					NodePool:                  eks.NewNodePool{},
					ShouldStoreNodePool:       true,
					ShouldUpdateClusterStatus: true,
					UserID:                    2,
				},
			},
		},
		{
			caseDescription: "get AMI size error -> error",
			expectedError:   errors.New("get AMI size error"),
			input: inputType{
				workflow: NewCreateNodePoolWorkflow(),
				input: CreateNodePoolWorkflowInput{
					ClusterID:                 1,
					NodePool:                  eks.NewNodePool{},
					ShouldStoreNodePool:       true,
					ShouldUpdateClusterStatus: true,
					UserID:                    2,
				},
			},
		},
		{
			caseDescription: "select volume size error -> error",
			expectedError:   errors.New("select volume size error"),
			input: inputType{
				workflow: NewCreateNodePoolWorkflow(),
				input: CreateNodePoolWorkflowInput{
					ClusterID:                 1,
					NodePool:                  eks.NewNodePool{},
					ShouldStoreNodePool:       true,
					ShouldUpdateClusterStatus: true,
					UserID:                    2,
				},
			},
		},
		{
			caseDescription: "create node pool label set error -> error",
			expectedError:   errors.New("create node pool label set error"),
			input: inputType{
				workflow: NewCreateNodePoolWorkflow(),
				input: CreateNodePoolWorkflowInput{
					ClusterID:                 1,
					NodePool:                  eks.NewNodePool{},
					ShouldStoreNodePool:       true,
					ShouldUpdateClusterStatus: true,
					UserID:                    2,
				},
			},
		},
		{
			caseDescription: "get VPC config error -> error",
			expectedError:   errors.New("get VPC config error"),
			input: inputType{
				workflow: NewCreateNodePoolWorkflow(),
				input: CreateNodePoolWorkflowInput{
					ClusterID:                 1,
					NodePool:                  eks.NewNodePool{},
					ShouldStoreNodePool:       true,
					ShouldUpdateClusterStatus: true,
					UserID:                    2,
				},
			},
		},
		{
			caseDescription: "create ASG error -> error",
			expectedError:   errors.New("create ASG error"),
			input: inputType{
				workflow: NewCreateNodePoolWorkflow(),
				input: CreateNodePoolWorkflowInput{
					ClusterID:                 1,
					NodePool:                  eks.NewNodePool{},
					ShouldStoreNodePool:       true,
					ShouldUpdateClusterStatus: true,
					UserID:                    2,
				},
			},
		},
		{
			caseDescription: "create ASG error -> error",
			expectedError:   errors.New("create ASG error"),
			input: inputType{
				workflow: NewCreateNodePoolWorkflow(),
				input: CreateNodePoolWorkflowInput{
					ClusterID:                 1,
					NodePool:                  eks.NewNodePool{},
					ShouldStoreNodePool:       true,
					ShouldUpdateClusterStatus: true,
					UserID:                    2,
				},
			},
		},
		{
			caseDescription: "set cluster status error -> error",
			expectedError:   errors.New("set cluster status error"),
			input: inputType{
				workflow: NewCreateNodePoolWorkflow(),
				input: CreateNodePoolWorkflowInput{
					ClusterID:                 1,
					NodePool:                  eks.NewNodePool{},
					ShouldStoreNodePool:       true,
					ShouldUpdateClusterStatus: true,
					UserID:                    2,
				},
			},
		},
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				workflow: NewCreateNodePoolWorkflow(),
				input: CreateNodePoolWorkflowInput{
					ClusterID:                 1,
					NodePool:                  eks.NewNodePool{},
					ShouldStoreNodePool:       true,
					ShouldUpdateClusterStatus: true,
					UserID:                    2,
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterWorkflowWithOptions(
				testCase.input.workflow.Execute,
				workflow.RegisterOptions{
					Name: t.Name(),
				},
			)
			environment.RegisterActivityWithOptions(
				func(
					ctx context.Context,
					input ListStoredEKSClustersActivityInput,
				) (*ListStoredEKSClustersActivityOutput, error) {
					if testCase.expectedError != nil &&
						strings.HasPrefix(testCase.expectedError.Error(), "list stored EKS clusters error") {
						return nil, testCase.expectedError
					}

					return &ListStoredEKSClustersActivityOutput{
						EKSClusters: map[uint]eksmodel.EKSClusterModel{
							testCase.input.input.ClusterID: {
								Cluster: clustermodel.ClusterModel{
									ID:       testCase.input.input.ClusterID,
									SecretID: "secret-id",
								},
								ClusterID: testCase.input.input.ClusterID,
								Subnets: []*eksmodel.EKSSubnetModel{
									{
										SubnetId: aws.String(""),
									},
								},
							},
						},
					}, nil
				},
				activity.RegisterOptions{
					Name: ListStoredEKSClustersActivityName,
				},
			)
			if testCase.input.input.ShouldStoreNodePool {
				environment.RegisterActivityWithOptions(
					func(ctx context.Context, input CreateStoredNodePoolActivityInput) error {
						if testCase.expectedError != nil &&
							strings.HasPrefix(testCase.expectedError.Error(), "create stored node pool error") {
							return testCase.expectedError
						}

						return nil
					},
					activity.RegisterOptions{
						Name: CreateStoredNodePoolActivityName,
					},
				)
			}
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input GetAMISizeActivityInput) (*GetAMISizeActivityOutput, error) {
					if testCase.expectedError != nil &&
						strings.HasPrefix(testCase.expectedError.Error(), "get AMI size error") {
						return nil, testCase.expectedError
					}

					return &GetAMISizeActivityOutput{AMISize: 1}, nil
				},
				activity.RegisterOptions{
					Name: GetAMISizeActivityName,
				},
			)
			environment.RegisterActivityWithOptions(
				func(
					ctx context.Context,
					input SelectVolumeSizeActivityInput,
				) (*SelectVolumeSizeActivityOutput, error) {
					if testCase.expectedError != nil &&
						strings.HasPrefix(testCase.expectedError.Error(), "select volume size error") {
						return nil, testCase.expectedError
					}

					return &SelectVolumeSizeActivityOutput{VolumeSize: 1}, nil
				},
				activity.RegisterOptions{
					Name: SelectVolumeSizeActivityName,
				},
			)
			environment.RegisterActivityWithOptions(
				func(
					ctx context.Context,
					input clusterworkflow.CreateNodePoolLabelSetActivityInput,
				) error {
					if testCase.expectedError != nil &&
						strings.HasPrefix(testCase.expectedError.Error(), "create node pool label set error") {
						return testCase.expectedError
					}

					return nil
				},
				activity.RegisterOptions{
					Name: clusterworkflow.CreateNodePoolLabelSetActivityName,
				},
			)
			environment.RegisterActivityWithOptions(
				func(
					ctx context.Context,
					input GetVpcConfigActivityInput,
				) (*GetVpcConfigActivityOutput, error) {
					if testCase.expectedError != nil &&
						strings.HasPrefix(testCase.expectedError.Error(), "get VPC config error") {
						return nil, testCase.expectedError
					}

					return &GetVpcConfigActivityOutput{}, nil
				},
				activity.RegisterOptions{
					Name: GetVpcConfigActivityName,
				},
			)
			environment.RegisterActivityWithOptions(
				func(
					ctx context.Context,
					input CreateAsgActivityInput,
				) (*CreateAsgActivityOutput, error) {
					if testCase.expectedError != nil &&
						strings.HasPrefix(testCase.expectedError.Error(), "create ASG error") {
						return nil, testCase.expectedError
					}

					return &CreateAsgActivityOutput{}, nil
				},
				activity.RegisterOptions{
					Name: CreateAsgActivityName,
				},
			)
			if testCase.input.input.ShouldUpdateClusterStatus {
				environment.RegisterActivityWithOptions(
					func(
						ctx context.Context,
						input SetClusterStatusActivityInput,
					) error {
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
			environment.RegisterActivityWithOptions(
				func(
					ctx context.Context,
					input SetNodePoolStatusActivityInput,
				) error {
					return nil
				},
				activity.RegisterOptions{
					Name: SetNodePoolStatusActivityName,
				},
			)

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

func TestNewCreateNodePoolWorkflow(t *testing.T) {
	require.Equal(t, &CreateNodePoolWorkflow{}, NewCreateNodePoolWorkflow())
}
