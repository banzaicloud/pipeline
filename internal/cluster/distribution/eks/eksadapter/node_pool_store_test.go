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

package eksadapter

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	//  SQLite driver used for integration test
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	pkggormtest "github.com/banzaicloud/pipeline/pkg/gorm/test"
)

func setUpDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	tables := []interface{}{
		&eksmodel.AmazonNodePoolsModel{},
		&eksmodel.EKSClusterModel{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating auth tables")

	err = db.AutoMigrate(tables...).Error
	require.NoError(t, err)

	return db
}

func TestNodePoolStoreCreateNodePool(t *testing.T) {
	type inputType struct {
		s              *nodePoolStore
		ctx            context.Context
		organizationID uint
		clusterID      uint
		clusterName    string
		createdBy      uint
		nodePool       eks.NewNodePool
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "cluster not found error -> error",
			expectedError: cluster.NotFoundError{
				OrganizationID: 1,
				ClusterID:      2,
				ClusterName:    "cluster-name",
			},
			input: inputType{
				s: &nodePoolStore{
					db: pkggormtest.NewFakeDatabase(t).
						CreateTablesFromEntities(t, &eksmodel.EKSClusterModel{}).DB,
				},
				ctx:            context.Background(),
				organizationID: 1,
				clusterID:      2,
				clusterName:    "cluster-name",
				createdBy:      3,
				nodePool:       eks.NewNodePool{},
			},
		},
		{
			caseDescription: "database fetch error -> error",
			expectedError:   errors.New("fetching cluster from database failed: test error"),
			input: inputType{
				s: &nodePoolStore{
					db: pkggormtest.NewFakeDatabase(t).
						CreateTablesFromEntities(
							t,
							&clustermodel.ClusterModel{},
							&eksmodel.EKSClusterModel{},
						).
						SaveEntities(
							t,
							&eksmodel.EKSClusterModel{
								Cluster: clustermodel.ClusterModel{
									CreatedBy:      3,
									ID:             2,
									Name:           "cluster-name",
									OrganizationID: 1,
								},
								ClusterID: 2,
							},
						).
						SetError(t, errors.New("test error")).DB,
				},
				ctx:            context.Background(),
				organizationID: 1,
				clusterID:      2,
				clusterName:    "cluster-name",
				createdBy:      3,
				nodePool:       eks.NewNodePool{},
			},
		},
		{
			caseDescription: "database save error -> error",
			expectedError:   errors.New("creating node pool in database failed: no such table: amazon_node_pools"),
			input: inputType{
				s: &nodePoolStore{
					db: pkggormtest.NewFakeDatabase(t).
						CreateTablesFromEntities(
							t,
							&clustermodel.ClusterModel{},
							&eksmodel.EKSClusterModel{},
						).
						SaveEntities(
							t,
							&eksmodel.EKSClusterModel{
								Cluster: clustermodel.ClusterModel{
									CreatedBy:      3,
									ID:             2,
									Name:           "cluster-name",
									OrganizationID: 1,
								},
								ClusterID: 2,
							},
						).DB,
				},
				ctx:            context.Background(),
				organizationID: 1,
				clusterID:      2,
				clusterName:    "cluster-name",
				createdBy:      3,
				nodePool:       eks.NewNodePool{},
			},
		},
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				s: &nodePoolStore{
					db: pkggormtest.NewFakeDatabase(t).
						CreateTablesFromEntities(
							t,
							&clustermodel.ClusterModel{},
							&eksmodel.EKSClusterModel{},
							&eksmodel.AmazonNodePoolsModel{},
						).
						SaveEntities(
							t,
							&eksmodel.EKSClusterModel{
								Cluster: clustermodel.ClusterModel{
									CreatedBy:      3,
									ID:             2,
									Name:           "cluster-name",
									OrganizationID: 1,
								},
								ClusterID: 2,
							},
						).DB,
				},
				ctx:            context.Background(),
				organizationID: 1,
				clusterID:      2,
				clusterName:    "cluster-name",
				createdBy:      3,
				nodePool:       eks.NewNodePool{},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualError := testCase.input.s.CreateNodePool(
				testCase.input.ctx,
				testCase.input.organizationID,
				testCase.input.clusterID,
				testCase.input.clusterName,
				testCase.input.createdBy,
				testCase.input.nodePool,
			)

			if testCase.expectedError == nil {
				require.NoError(t, actualError)

				err := testCase.input.s.db.First(&eksmodel.AmazonNodePoolsModel{}).Error
				require.NoError(t, err)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}

func TestNodePoolStoreDeleteNodePool(t *testing.T) {
	type inputType struct {
		clusterID      uint
		clusterName    string
		clusters       []eksmodel.EKSClusterModel
		nodePoolName   string
		organizationID uint
	}

	type outputType struct {
		expectedClusters []eksmodel.EKSClusterModel
		expectedError    error
	}

	type caseType struct {
		caseName string
		input    inputType
		output   outputType
	}

	testCases := []caseType{
		{
			caseName: "cluster not found error",
			input: inputType{
				clusterID:      1,
				clusterName:    "cluster-1",
				clusters:       []eksmodel.EKSClusterModel{},
				nodePoolName:   "1-pool-1",
				organizationID: 1,
			},
			output: outputType{
				expectedClusters: []eksmodel.EKSClusterModel{},
				expectedError:    errors.New("cluster not found"),
			},
		},
		{
			caseName: "success",
			input: inputType{
				clusterID:   1,
				clusterName: "cluster-3",
				clusters: []eksmodel.EKSClusterModel{
					{
						ID:        2,
						ClusterID: 1,
						NodePools: []*eksmodel.AmazonNodePoolsModel{
							{
								ID:            1,
								ClusterID:     2,
								Name:          "1-pool-1",
								StackID:       "",
								Status:        eks.NodePoolStatusCreating,
								StatusMessage: "",
							},
							{
								ID:            2,
								ClusterID:     2,
								Name:          "1-pool-2",
								StackID:       "",
								Status:        eks.NodePoolStatusCreating,
								StatusMessage: "",
							},
						},
					},
				},
				nodePoolName:   "1-pool-2",
				organizationID: 1,
			},
			output: outputType{
				expectedClusters: []eksmodel.EKSClusterModel{
					{
						ID:        2,
						ClusterID: 1,
						NodePools: []*eksmodel.AmazonNodePoolsModel{
							{
								ID:            1,
								ClusterID:     2,
								Name:          "1-pool-1",
								StackID:       "",
								Status:        eks.NodePoolStatusCreating,
								StatusMessage: "",
							},
						},
						SSHGenerated: true,
					},
				},
				expectedError: nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			database := setUpDatabase(t)
			for clusterIndex := range testCase.input.clusters {
				err := database.Save(&testCase.input.clusters[clusterIndex]).Error
				require.NoError(t, err)
			}

			nodePoolStore := NewNodePoolStore(database)

			actualError := nodePoolStore.DeleteNodePool(
				context.Background(),
				testCase.input.organizationID,
				testCase.input.clusterID,
				testCase.input.clusterName,
				testCase.input.nodePoolName,
			)

			var actualClusters []eksmodel.EKSClusterModel
			err := database.Preload("NodePools").Find(&actualClusters).Error
			require.NoError(t, err)
			for clusterIndex := range actualClusters {
				for nodePoolIndex := range actualClusters[clusterIndex].NodePools {
					actualClusters[clusterIndex].NodePools[nodePoolIndex].CreatedAt = time.Time{}
				}
			}

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedClusters, actualClusters)
		})
	}
}

func TestNodePoolStoreListNodePools(t *testing.T) {
	type inputType struct {
		cluster        eksmodel.EKSClusterModel
		clusterID      uint
		clusterName    string
		organizationID uint
	}

	type outputType struct {
		expectedError             error
		expectedExistingNodePools map[string]eks.ExistingNodePool
	}

	type caseType struct {
		caseName string
		input    inputType
		output   outputType
	}

	testCases := []caseType{
		{
			caseName: "cluster not found error",
			input: inputType{
				cluster:        eksmodel.EKSClusterModel{},
				clusterID:      1,
				clusterName:    "cluster-1",
				organizationID: 1,
			},
			output: outputType{
				expectedError:             errors.New("cluster not found"),
				expectedExistingNodePools: nil,
			},
		},
		{
			caseName: "success",
			input: inputType{
				cluster: eksmodel.EKSClusterModel{
					ClusterID: 2,
					NodePools: []*eksmodel.AmazonNodePoolsModel{
						{
							Name:          "2-pool-1",
							StackID:       "2-pool-1/stack-id",
							Status:        eks.NodePoolStatusCreating,
							StatusMessage: "",
						},
						{
							Name:          "2-pool-2",
							StackID:       "2-pool-2/stack-id",
							Status:        eks.NodePoolStatusError,
							StatusMessage: "AWS test error",
						},
					},
				},
				clusterID:      2,
				clusterName:    "cluster-2",
				organizationID: 1,
			},
			output: outputType{
				expectedError: nil,
				expectedExistingNodePools: map[string]eks.ExistingNodePool{
					"2-pool-1": {
						Name:          "2-pool-1",
						StackID:       "2-pool-1/stack-id",
						Status:        eks.NodePoolStatusCreating,
						StatusMessage: "",
					},
					"2-pool-2": {
						Name:          "2-pool-2",
						StackID:       "2-pool-2/stack-id",
						Status:        eks.NodePoolStatusError,
						StatusMessage: "AWS test error",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			database := setUpDatabase(t)
			err := database.Save(&testCase.input.cluster).Error
			require.NoError(t, err)

			nodePoolStore := NewNodePoolStore(database)

			actualExistingNodePools, actualError := nodePoolStore.ListNodePools(
				context.Background(),
				testCase.input.organizationID,
				testCase.input.clusterID,
				testCase.input.clusterName,
			)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedExistingNodePools, actualExistingNodePools)
		})
	}
}

func TestNodePoolStoreUpdateNodePoolStackID(t *testing.T) {
	type inputType struct {
		clusterID       uint
		clusterName     string
		clusters        []eksmodel.EKSClusterModel
		nodePoolName    string
		nodePoolStackID string
		organizationID  uint
	}

	type outputType struct {
		expectedClusters []eksmodel.EKSClusterModel
		expectedError    error
	}

	type caseType struct {
		caseName string
		input    inputType
		output   outputType
	}

	testCases := []caseType{
		{
			caseName: "cluster not found error",
			input: inputType{
				clusterID:       1,
				clusterName:     "cluster-1",
				clusters:        []eksmodel.EKSClusterModel{},
				nodePoolName:    "1-pool-1",
				nodePoolStackID: "1-pool-1/stack-id",
				organizationID:  1,
			},
			output: outputType{
				expectedClusters: []eksmodel.EKSClusterModel{},
				expectedError:    errors.New("cluster not found"),
			},
		},
		{
			caseName: "success",
			input: inputType{
				clusterID:   1,
				clusterName: "cluster-2",
				clusters: []eksmodel.EKSClusterModel{
					{
						ID:        2,
						ClusterID: 1,
						NodePools: []*eksmodel.AmazonNodePoolsModel{
							{
								ID:            1,
								ClusterID:     2,
								Name:          "1-pool-1",
								StackID:       "",
								Status:        eks.NodePoolStatusCreating,
								StatusMessage: "",
							},
							{
								ID:            2,
								ClusterID:     2,
								Name:          "1-pool-2",
								StackID:       "",
								Status:        eks.NodePoolStatusCreating,
								StatusMessage: "",
							},
						},
					},
				},
				nodePoolName:    "1-pool-2",
				nodePoolStackID: "1-pool-2/stack-id",
				organizationID:  1,
			},
			output: outputType{
				expectedClusters: []eksmodel.EKSClusterModel{
					{
						ID:        2,
						ClusterID: 1,
						NodePools: []*eksmodel.AmazonNodePoolsModel{
							{
								ID:            1,
								ClusterID:     2,
								Name:          "1-pool-1",
								StackID:       "",
								Status:        eks.NodePoolStatusCreating,
								StatusMessage: "",
							},
							{
								ID:            2,
								ClusterID:     2,
								Name:          "1-pool-2",
								StackID:       "1-pool-2/stack-id",
								Status:        eks.NodePoolStatusEmpty,
								StatusMessage: "",
							},
						},
						SSHGenerated: true,
					},
				},
				expectedError: nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			database := setUpDatabase(t)
			for clusterIndex := range testCase.input.clusters {
				err := database.Save(&testCase.input.clusters[clusterIndex]).Error
				require.NoError(t, err)
			}

			nodePoolStore := NewNodePoolStore(database)

			actualError := nodePoolStore.UpdateNodePoolStackID(
				context.Background(),
				testCase.input.organizationID,
				testCase.input.clusterID,
				testCase.input.clusterName,
				testCase.input.nodePoolName,
				testCase.input.nodePoolStackID,
			)

			var actualClusters []eksmodel.EKSClusterModel
			err := database.Preload("NodePools").Find(&actualClusters).Error
			require.NoError(t, err)
			for clusterIndex := range actualClusters {
				for nodePoolIndex := range actualClusters[clusterIndex].NodePools {
					actualClusters[clusterIndex].NodePools[nodePoolIndex].CreatedAt = time.Time{}
				}
			}

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedClusters, actualClusters)
		})
	}
}

func TestNodePoolStoreUpdateNodePoolStatus(t *testing.T) {
	type inputType struct {
		clusterID             uint
		clusterName           string
		clusters              []eksmodel.EKSClusterModel
		nodePoolName          string
		nodePoolStatus        eks.NodePoolStatus
		nodePoolStatusMessage string
		organizationID        uint
	}

	type outputType struct {
		expectedClusters []eksmodel.EKSClusterModel
		expectedError    error
	}

	type caseType struct {
		caseName string
		input    inputType
		output   outputType
	}

	testCases := []caseType{
		{
			caseName: "cluster not found error",
			input: inputType{
				clusterID:             1,
				clusterName:           "cluster-1",
				clusters:              []eksmodel.EKSClusterModel{},
				nodePoolName:          "1-pool-1",
				nodePoolStatus:        eks.NodePoolStatusUnknown,
				nodePoolStatusMessage: "",
				organizationID:        1,
			},
			output: outputType{
				expectedClusters: []eksmodel.EKSClusterModel{},
				expectedError:    errors.New("cluster not found"),
			},
		},
		{
			caseName: "success",
			input: inputType{
				clusterID:   1,
				clusterName: "cluster-1",
				clusters: []eksmodel.EKSClusterModel{
					{
						ID:        2,
						ClusterID: 1,
						NodePools: []*eksmodel.AmazonNodePoolsModel{
							{
								ID:            1,
								ClusterID:     2,
								Name:          "1-pool-1",
								StackID:       "",
								Status:        eks.NodePoolStatusCreating,
								StatusMessage: "",
							},
							{
								ID:            2,
								ClusterID:     2,
								Name:          "1-pool-2",
								StackID:       "",
								Status:        eks.NodePoolStatusCreating,
								StatusMessage: "",
							},
						},
					},
				},
				nodePoolName:          "1-pool-2",
				nodePoolStatus:        eks.NodePoolStatusError,
				nodePoolStatusMessage: "test AWS error",
				organizationID:        1,
			},
			output: outputType{
				expectedClusters: []eksmodel.EKSClusterModel{
					{
						ID:        2,
						ClusterID: 1,
						NodePools: []*eksmodel.AmazonNodePoolsModel{
							{
								ID:            1,
								ClusterID:     2,
								Name:          "1-pool-1",
								StackID:       "",
								Status:        eks.NodePoolStatusCreating,
								StatusMessage: "",
							},
							{
								ID:            2,
								ClusterID:     2,
								Name:          "1-pool-2",
								StackID:       "",
								Status:        eks.NodePoolStatusError,
								StatusMessage: "test AWS error",
							},
						},
						SSHGenerated: true,
					},
				},
				expectedError: nil,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			database := setUpDatabase(t)
			for clusterIndex := range testCase.input.clusters {
				err := database.Save(&testCase.input.clusters[clusterIndex]).Error
				require.NoError(t, err)
			}

			nodePoolStore := NewNodePoolStore(database)

			actualError := nodePoolStore.UpdateNodePoolStatus(
				context.Background(),
				testCase.input.organizationID,
				testCase.input.clusterID,
				testCase.input.clusterName,
				testCase.input.nodePoolName,
				testCase.input.nodePoolStatus,
				testCase.input.nodePoolStatusMessage,
			)

			var actualClusters []eksmodel.EKSClusterModel
			err := database.Preload("NodePools").Find(&actualClusters).Error
			require.NoError(t, err)
			for clusterIndex := range actualClusters {
				for nodePoolIndex := range actualClusters[clusterIndex].NodePools {
					actualClusters[clusterIndex].NodePools[nodePoolIndex].CreatedAt = time.Time{}
				}
			}

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedClusters, actualClusters)
		})
	}
}
