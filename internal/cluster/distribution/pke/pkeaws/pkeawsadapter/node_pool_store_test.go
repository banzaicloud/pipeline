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

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsmodel"
)

func setUpDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	tables := []interface{}{
		&pkeawsmodel.AmazonNodePoolsModel{},
		&pkeawsmodel.PKEAWSClusterModel{},
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

func TestNodePoolStoreListNodePools(t *testing.T) {
	type inputType struct {
		cluster        pkeawsmodel.PKEAWSClusterModel
		clusterID      uint
		clusterName    string
		organizationID uint
	}

	type outputType struct {
		expectedError             error
		expectedExistingNodePools map[string]pke.ExistingNodePool
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
				cluster:        pkeawsmodel.PKEAWSClusterModel{},
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
				cluster: pkeawsmodel.PKEAWSClusterModel{
					ClusterID: 2,
					NodePools: []*pkeawsmodel.AmazonNodePoolsModel{
						{
							Name:          "2-pool-1",
							StackID:       "2-pool-1/stack-id",
							Status:        pke.NodePoolStatusCreating,
							StatusMessage: "",
						},
						{
							Name:          "2-pool-2",
							StackID:       "2-pool-2/stack-id",
							Status:        pke.NodePoolStatusError,
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
				expectedExistingNodePools: map[string]pke.ExistingNodePool{
					"2-pool-1": {
						Name:          "2-pool-1",
						StackID:       "2-pool-1/stack-id",
						Status:        pke.NodePoolStatusCreating,
						StatusMessage: "",
					},
					"2-pool-2": {
						Name:          "2-pool-2",
						StackID:       "2-pool-2/stack-id",
						Status:        pke.NodePoolStatusError,
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
		clusters        []pkeawsmodel.PKEAWSClusterModel
		nodePoolName    string
		nodePoolStackID string
		organizationID  uint
	}

	type outputType struct {
		expectedClusters []pkeawsmodel.PKEAWSClusterModel
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
				clusters:        []pkeawsmodel.PKEAWSClusterModel{},
				nodePoolName:    "1-pool-1",
				nodePoolStackID: "1-pool-1/stack-id",
				organizationID:  1,
			},
			output: outputType{
				expectedClusters: []pkeawsmodel.PKEAWSClusterModel{},
				expectedError:    errors.New("cluster not found"),
			},
		},
		{
			caseName: "success",
			input: inputType{
				clusterID:   1,
				clusterName: "cluster-2",
				clusters: []pkeawsmodel.PKEAWSClusterModel{
					{
						ID:        2,
						ClusterID: 1,
						NodePools: []*pkeawsmodel.AmazonNodePoolsModel{
							{
								ID:            1,
								ClusterID:     2,
								Name:          "1-pool-1",
								StackID:       "",
								Status:        pke.NodePoolStatusCreating,
								StatusMessage: "",
							},
							{
								ID:            2,
								ClusterID:     2,
								Name:          "1-pool-2",
								StackID:       "",
								Status:        pke.NodePoolStatusCreating,
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
				expectedClusters: []pkeawsmodel.PKEAWSClusterModel{
					{
						ID:        2,
						ClusterID: 1,
						NodePools: []*pkeawsmodel.AmazonNodePoolsModel{
							{
								ID:            1,
								ClusterID:     2,
								Name:          "1-pool-1",
								StackID:       "",
								Status:        pke.NodePoolStatusCreating,
								StatusMessage: "",
							},
							{
								ID:            2,
								ClusterID:     2,
								Name:          "1-pool-2",
								StackID:       "1-pool-2/stack-id",
								Status:        pke.NodePoolStatusEmpty,
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

			var actualClusters []pkeawsmodel.PKEAWSClusterModel
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
		clusters              []pkeawsmodel.PKEAWSClusterModel
		nodePoolName          string
		nodePoolStatus        pke.NodePoolStatus
		nodePoolStatusMessage string
		organizationID        uint
	}

	type outputType struct {
		expectedClusters []pkeawsmodel.PKEAWSClusterModel
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
				clusters:              []pkeawsmodel.PKEAWSClusterModel{},
				nodePoolName:          "1-pool-1",
				nodePoolStatus:        pke.NodePoolStatusUnknown,
				nodePoolStatusMessage: "",
				organizationID:        1,
			},
			output: outputType{
				expectedClusters: []pkeawsmodel.PKEAWSClusterModel{},
				expectedError:    errors.New("cluster not found"),
			},
		},
		{
			caseName: "success",
			input: inputType{
				clusterID:   1,
				clusterName: "cluster-1",
				clusters: []pkeawsmodel.PKEAWSClusterModel{
					{
						ID:        2,
						ClusterID: 1,
						NodePools: []*pkeawsmodel.AmazonNodePoolsModel{
							{
								ID:            1,
								ClusterID:     2,
								Name:          "1-pool-1",
								StackID:       "",
								Status:        pke.NodePoolStatusCreating,
								StatusMessage: "",
							},
							{
								ID:            2,
								ClusterID:     2,
								Name:          "1-pool-2",
								StackID:       "",
								Status:        pke.NodePoolStatusCreating,
								StatusMessage: "",
							},
						},
					},
				},
				nodePoolName:          "1-pool-2",
				nodePoolStatus:        pke.NodePoolStatusError,
				nodePoolStatusMessage: "test AWS error",
				organizationID:        1,
			},
			output: outputType{
				expectedClusters: []pkeawsmodel.PKEAWSClusterModel{
					{
						ID:        2,
						ClusterID: 1,
						NodePools: []*pkeawsmodel.AmazonNodePoolsModel{
							{
								ID:            1,
								ClusterID:     2,
								Name:          "1-pool-1",
								StackID:       "",
								Status:        pke.NodePoolStatusCreating,
								StatusMessage: "",
							},
							{
								ID:            2,
								ClusterID:     2,
								Name:          "1-pool-2",
								StackID:       "",
								Status:        pke.NodePoolStatusError,
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

			var actualClusters []pkeawsmodel.PKEAWSClusterModel
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
