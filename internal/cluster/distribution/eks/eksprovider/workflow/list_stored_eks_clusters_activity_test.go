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
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	pkggormtest "github.com/banzaicloud/pipeline/pkg/gorm/test"
)

func newTestEKSCluster(
	genericClusterID uint,
	sshGenerated bool,
	eksCluster eksmodel.EKSClusterModel,
) eksmodel.EKSClusterModel {
	zeroTime := time.Time{}
	if eksCluster.Cluster.CreatedAt == zeroTime {
		eksCluster.Cluster.CreatedAt = time.Now()
	}

	if eksCluster.Cluster.ID == 0 {
		eksCluster.Cluster.ID = genericClusterID
	}

	if eksCluster.Cluster.Location == "" {
		eksCluster.Cluster.Location = "unknown"
	}

	if eksCluster.Cluster.UID == "" {
		eksCluster.Cluster.UID = "default0-test-gene-ric0-clusterUUID0"
	}

	if eksCluster.Cluster.UpdatedAt == zeroTime {
		eksCluster.Cluster.UpdatedAt = time.Now()
	}

	if eksCluster.ClusterID == 0 {
		eksCluster.ClusterID = genericClusterID
	}

	if eksCluster.NodePools == nil {
		eksCluster.NodePools = []*eksmodel.AmazonNodePoolsModel{}
	}

	if eksCluster.Subnets == nil {
		eksCluster.Subnets = []*eksmodel.EKSSubnetModel{}
	}

	if eksCluster.SSHGenerated != sshGenerated {
		eksCluster.SSHGenerated = sshGenerated
	}

	return eksCluster
}

func newTestEKSClusterDatabase(t *testing.T) *pkggormtest.FakeDatabase {
	return pkggormtest.NewFakeDatabase(t).CreateTablesFromEntities(
		t,
		&clustermodel.ClusterModel{},
		&eksmodel.AmazonNodePoolsModel{},
		&eksmodel.EKSClusterModel{},
		&eksmodel.EKSSubnetModel{},
	)
}

func TestNewListStoredEKSClustersActivity(t *testing.T) {
	testCases := []struct {
		caseDescription  string
		expectedActivity *ListStoredEKSClustersActivity
		inputDB          *gorm.DB
	}{
		{
			caseDescription: "not nil database success",
			expectedActivity: &ListStoredEKSClustersActivity{
				db: &gorm.DB{
					Error: errors.NewPlain("test error"),
				},
			},
			inputDB: &gorm.DB{
				Error: errors.NewPlain("test error"),
			},
		},
		{
			caseDescription:  "nil database success",
			expectedActivity: &ListStoredEKSClustersActivity{},
			inputDB:          nil,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualActivity := NewListStoredEKSClustersActivity(testCase.inputDB)

			require.Equal(t, testCase.expectedActivity, actualActivity)
		})
	}
}

func TestListStoredEKSClustersActivityExecute(t *testing.T) {
	type inputType struct {
		activity *ListStoredEKSClustersActivity
		input    ListStoredEKSClustersActivityInput
	}

	type outputType struct {
		expectedError  error
		expectedOutput *ListStoredEKSClustersActivityOutput
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "database error -> error",
			input: inputType{
				activity: NewListStoredEKSClustersActivity(
					newTestEKSClusterDatabase(t).SetError(t, errors.New("test error")).DB,
				),
			},
			output: outputType{
				expectedError:  errors.New("listing stored eks clusters failed: test error"),
				expectedOutput: nil,
			},
		},
		{
			caseDescription: "nil optional listed generic cluster IDs -> all cluster listed success",
			input: inputType{
				activity: NewListStoredEKSClustersActivity(
					newTestEKSClusterDatabase(t).
						SaveEntities(
							t,
							&eksmodel.EKSClusterModel{
								ID: 1,
								Cluster: clustermodel.ClusterModel{
									ID: 2,
								},
							},
							&eksmodel.EKSClusterModel{
								ID: 2,
								Cluster: clustermodel.ClusterModel{
									ID: 3,
								},
							},
						).DB,
				),
				input: ListStoredEKSClustersActivityInput{
					OptionalListedGenericClusterIDs: nil,
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &ListStoredEKSClustersActivityOutput{
					map[uint]eksmodel.EKSClusterModel{
						2: newTestEKSCluster(2, true, eksmodel.EKSClusterModel{
							ID: 1,
						}),
						3: newTestEKSCluster(3, true, eksmodel.EKSClusterModel{
							ID: 2,
						}),
					},
				},
			},
		},
		{
			caseDescription: "not empty optional listed generic cluster IDs -> filtered listed clusters success",
			input: inputType{
				activity: NewListStoredEKSClustersActivity(
					newTestEKSClusterDatabase(t).
						SaveEntities(
							t,
							&eksmodel.EKSClusterModel{
								ID: 1,
								Cluster: clustermodel.ClusterModel{
									ID: 2,
								},
							},
							&eksmodel.EKSClusterModel{
								ID: 2,
								Cluster: clustermodel.ClusterModel{
									ID: 3,
								},
							},
							&eksmodel.EKSClusterModel{
								ID: 3,
								Cluster: clustermodel.ClusterModel{
									ID: 5,
								},
							},
							&eksmodel.EKSClusterModel{
								ID: 4,
								Cluster: clustermodel.ClusterModel{
									ID: 6,
								},
							},
						).DB,
				),
				input: ListStoredEKSClustersActivityInput{
					OptionalListedGenericClusterIDs: []uint{3, 6},
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &ListStoredEKSClustersActivityOutput{
					map[uint]eksmodel.EKSClusterModel{
						3: newTestEKSCluster(3, true, eksmodel.EKSClusterModel{
							ID: 2,
						}),
						6: newTestEKSCluster(6, true, eksmodel.EKSClusterModel{
							ID: 4,
						}),
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualOutput, actualError := testCase.input.activity.Execute(
				context.Background(),
				testCase.input.input,
			)

			if actualOutput != nil {
				// Note: faking auto-generated database values for equality check.
				for genericClusterID, eksCluster := range actualOutput.EKSClusters {
					require.InEpsilon(t, time.Now().Unix(), eksCluster.Cluster.CreatedAt.Unix(), 3.0)
					require.NotEmpty(t, eksCluster.Cluster.UID)
					require.InEpsilon(t, time.Now().Unix(), eksCluster.Cluster.UpdatedAt.Unix(), 3.0)

					eksCluster.Cluster.CreatedAt =
						testCase.output.expectedOutput.EKSClusters[genericClusterID].Cluster.CreatedAt
					eksCluster.Cluster.UID = testCase.output.expectedOutput.EKSClusters[genericClusterID].Cluster.UID
					eksCluster.Cluster.UpdatedAt =
						testCase.output.expectedOutput.EKSClusters[genericClusterID].Cluster.UpdatedAt

					actualOutput.EKSClusters[genericClusterID] = eksCluster
				}
			}

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedOutput, actualOutput)
		})
	}
}

func TestListStoredEKSClusters(t *testing.T) {
	type inputType struct {
		optionalListedGenericClusterIDs []uint
	}

	type outputType struct {
		expectedError       error
		expectedEKSClusters map[uint]eksmodel.EKSClusterModel
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "success",
			input: inputType{
				optionalListedGenericClusterIDs: []uint{
					1,
				},
			},
			output: outputType{
				expectedError: nil,
				expectedEKSClusters: map[uint]eksmodel.EKSClusterModel{
					1: {},
				},
			},
		},
		{
			caseDescription: "error",
			input: inputType{
				optionalListedGenericClusterIDs: []uint{
					1,
				},
			},
			output: outputType{
				expectedError:       errors.New("test error"),
				expectedEKSClusters: nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			environment := suite.NewTestWorkflowEnvironment()
			environment.RegisterActivityWithOptions(
				func(ctx context.Context, input ListStoredEKSClustersActivityInput) (*ListStoredEKSClustersActivityOutput, error) {
					return &ListStoredEKSClustersActivityOutput{
						EKSClusters: testCase.output.expectedEKSClusters,
					}, testCase.output.expectedError
				},
				activity.RegisterOptions{Name: ListStoredEKSClustersActivityName},
			)

			var actualError error
			var actualEKSClusters map[uint]eksmodel.EKSClusterModel
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualEKSClusters, actualError = listStoredEKSClusters(
					ctx,
					testCase.input.optionalListedGenericClusterIDs...,
				)

				return nil
			})

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedEKSClusters, actualEKSClusters)
		})
	}
}

func TestListStoredEKSClustersAsync(t *testing.T) {
	type inputType struct {
		optionalListedGenericClusterIDs []uint
	}

	type outputType struct {
		expectedError  error
		expectedOutput *ListStoredEKSClustersActivityOutput
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "success",
			input: inputType{
				optionalListedGenericClusterIDs: []uint{
					1,
				},
			},
			output: outputType{
				expectedError: nil,
				expectedOutput: &ListStoredEKSClustersActivityOutput{
					EKSClusters: map[uint]eksmodel.EKSClusterModel{
						1: {},
					},
				},
			},
		},
		{
			caseDescription: "error",
			input: inputType{
				optionalListedGenericClusterIDs: []uint{
					1,
				},
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
				func(
					ctx context.Context,
					input ListStoredEKSClustersActivityInput,
				) (*ListStoredEKSClustersActivityOutput, error) {
					return testCase.output.expectedOutput, testCase.output.expectedError
				},
				activity.RegisterOptions{
					Name: ListStoredEKSClustersActivityName,
				},
			)

			actualError := (error)(nil)
			actualOutput := (*ListStoredEKSClustersActivityOutput)(nil)
			environment.ExecuteWorkflow(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					ScheduleToCloseTimeout: 10 * time.Second,
					ScheduleToStartTimeout: 3 * time.Second,
					StartToCloseTimeout:    7 * time.Second,
					WaitForCancellation:    true,
				})

				actualFuture := listStoredEKSClustersAsync(ctx, testCase.input.optionalListedGenericClusterIDs...)
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
