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

package eks

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/brn"
)

func TestServiceListNodePools(t *testing.T) {
	exampleClusterID := uint(0)
	exampleOrganizationID := uint(1)
	exampleCluster := cluster.Cluster{
		ID:             exampleClusterID,
		UID:            "cluster UID",
		Name:           "cluster name",
		OrganizationID: exampleOrganizationID,
		Status:         "cluster status",
		StatusMessage:  "cluster status message",
		Cloud:          "cluster cloud",
		Distribution:   "cluster distribution",
		Location:       "cluster location",
		SecretID: brn.ResourceName{
			Scheme:         "cluster secret ID scheme",
			OrganizationID: exampleOrganizationID,
			ResourceType:   "cluster secret ID resource type",
			ResourceID:     "cluster secret ID resource ID",
		},
		ConfigSecretID: brn.ResourceName{
			Scheme:         "cluster config secret ID scheme",
			OrganizationID: exampleOrganizationID,
			ResourceType:   "cluster config secret ID resource type",
			ResourceID:     "cluster config secret ID resource ID",
		},
		Tags: map[string]string{
			"cluster-tag": "cluster tag value",
		},
	}
	exampleNodePoolNames := []string{
		"cluster-node-pool-name-2",
		"cluster-node-pool-name-3",
	}
	exampleNodePools := []NodePool{
		{
			Name: "cluster-node-pool-name-2",
			Labels: map[string]string{
				"label-1": "value-1",
				"label-2": "value-2",
			},
			Size: 4,
			Autoscaling: Autoscaling{
				Enabled: true,
				MinSize: 1,
				MaxSize: 2,
			},
			VolumeSize:   50,
			InstanceType: "instance-type",
			Image:        "image",
			SpotPrice:    "5",
		},
		{
			Name: "cluster-node-pool-name-3",
			Labels: map[string]string{
				"label-3": "value-3",
			},
			Size: 6,
			Autoscaling: Autoscaling{
				Enabled: false,
				MinSize: 0,
				MaxSize: 0,
			},
			InstanceType: "instance-type",
			Image:        "image",
			SpotPrice:    "7",
		},
	}

	type constructionArgumentType struct {
		genericClusters Store
		nodePools       NodePoolStore
		nodePoolManager NodePoolManager
	}
	type functionCallArgumentType struct {
		ctx       context.Context
		clusterID uint
	}
	testCases := []struct {
		caseName              string
		constructionArguments constructionArgumentType
		expectedNodePools     []NodePool
		expectedNotNilError   bool
		functionCallArguments functionCallArgumentType
		setupMocks            func(constructionArgumentType, functionCallArgumentType)
	}{
		{
			caseName: "ClusterNotFound",
			constructionArguments: constructionArgumentType{
				genericClusters: &MockStore{},
				nodePools:       &MockNodePoolStore{},
				nodePoolManager: &MockNodePoolManager{},
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:       context.Background(),
				clusterID: 1,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				genericClustersMock := constructionArguments.genericClusters.(*MockStore)
				genericClustersMock.On("GetCluster", functionCallArguments.ctx, functionCallArguments.clusterID).Return(cluster.Cluster{}, errors.New("ClusterNotFound"))
			},
		},
		{
			caseName: "NodePoolNamesError",
			constructionArguments: constructionArgumentType{
				genericClusters: &MockStore{},
				nodePools:       &MockNodePoolStore{},
				nodePoolManager: &MockNodePoolManager{},
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:       context.Background(),
				clusterID: 1,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				genericClustersMock := constructionArguments.genericClusters.(*MockStore)
				genericClustersMock.On("GetCluster", functionCallArguments.ctx, functionCallArguments.clusterID).Return(exampleCluster, nil)

				nodePoolStoreMock := constructionArguments.nodePools.(*MockNodePoolStore)
				nodePoolStoreMock.On("ListNodePoolNames", functionCallArguments.ctx, functionCallArguments.clusterID).Return([]string{}, errors.New("NodePoolNamesError"))
			},
		},
		{
			caseName: "NodePoolsError",
			constructionArguments: constructionArgumentType{
				genericClusters: &MockStore{},
				nodePools:       &MockNodePoolStore{},
				nodePoolManager: &MockNodePoolManager{},
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:       context.Background(),
				clusterID: 1,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				genericClustersMock := constructionArguments.genericClusters.(*MockStore)
				genericClustersMock.On("GetCluster", functionCallArguments.ctx, functionCallArguments.clusterID).Return(exampleCluster, nil)

				nodePoolStoreMock := constructionArguments.nodePools.(*MockNodePoolStore)
				nodePoolStoreMock.On("ListNodePoolNames", functionCallArguments.ctx, functionCallArguments.clusterID).Return(exampleNodePoolNames, nil)

				nodePoolManagerMock := constructionArguments.nodePoolManager.(*MockNodePoolManager)
				nodePoolManagerMock.On("ListNodePools", functionCallArguments.ctx, exampleCluster, exampleNodePoolNames).Return([]NodePool{}, errors.New("NodePoolsError"))
			},
		},
		{
			caseName: "ServiceListNodePoolsSuccess",
			constructionArguments: constructionArgumentType{
				genericClusters: &MockStore{},
				nodePools:       &MockNodePoolStore{},
				nodePoolManager: &MockNodePoolManager{},
			},
			expectedNodePools:   exampleNodePools,
			expectedNotNilError: false,
			functionCallArguments: functionCallArgumentType{
				ctx:       context.Background(),
				clusterID: 1,
			},
			setupMocks: func(constructionArguments constructionArgumentType, functionCallArguments functionCallArgumentType) {
				genericClustersMock := constructionArguments.genericClusters.(*MockStore)
				genericClustersMock.On("GetCluster", functionCallArguments.ctx, functionCallArguments.clusterID).Return(exampleCluster, nil)

				nodePoolStoreMock := constructionArguments.nodePools.(*MockNodePoolStore)
				nodePoolStoreMock.On("ListNodePoolNames", functionCallArguments.ctx, functionCallArguments.clusterID).Return(exampleNodePoolNames, nil)

				nodePoolManagerMock := constructionArguments.nodePoolManager.(*MockNodePoolManager)
				nodePoolManagerMock.On("ListNodePools", functionCallArguments.ctx, exampleCluster, exampleNodePoolNames).Return(exampleNodePools, nil)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			testCase.setupMocks(testCase.constructionArguments, testCase.functionCallArguments)

			s := service{
				genericClusters: testCase.constructionArguments.genericClusters,
				nodePools:       testCase.constructionArguments.nodePools,
				nodePoolManager: testCase.constructionArguments.nodePoolManager,
			}

			got, err := s.ListNodePools(testCase.functionCallArguments.ctx, testCase.functionCallArguments.clusterID)

			require.Truef(t, (err != nil) == testCase.expectedNotNilError,
				"error value doesn't match the expectation, is expected: %+v, actual error value: %+v", testCase.expectedNotNilError, err)
			require.Equal(t, testCase.expectedNodePools, got)
		})
	}
}
