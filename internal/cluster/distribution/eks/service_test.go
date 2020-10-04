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
	"fmt"
	"testing"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/brn"
)

func TestNewNodePoolFromCFStackDescriptionError(t *testing.T) {
	type inputType struct {
		err              error
		existingNodePool ExistingNodePool
	}

	type outputType struct {
		expectedNodePool NodePool
	}

	testCases := []struct {
		caseName string
		input    inputType
		output   outputType
	}{
		{
			caseName: "old node pool, no stored information description failure success",
			input: inputType{
				err: fmt.Errorf("test error"),
				existingNodePool: ExistingNodePool{
					Name:          "node-pool-name",
					StackID:       "",
					Status:        NodePoolStatusEmpty,
					StatusMessage: "",
				},
			},
			output: outputType{
				expectedNodePool: NodePool{
					Name:          "node-pool-name",
					Status:        NodePoolStatusDeleting,
					StatusMessage: "",
				},
			},
		},
		{
			caseName: "pre-stack node pool description failure success",
			input: inputType{
				err: fmt.Errorf("test error"),
				existingNodePool: ExistingNodePool{
					Name:          "node-pool-name-2",
					StackID:       "",
					Status:        NodePoolStatusCreating,
					StatusMessage: "status message",
				},
			},
			output: outputType{
				expectedNodePool: NodePool{
					Name:          "node-pool-name-2",
					Status:        NodePoolStatusCreating,
					StatusMessage: "status message",
				},
			},
		},
		{
			caseName: "unknown description failure success",
			input: inputType{
				err: fmt.Errorf("test error"),
				existingNodePool: ExistingNodePool{
					Name:          "node-pool-name-3",
					StackID:       "node-pool-name-3/stack-id",
					Status:        NodePoolStatusCreating,
					StatusMessage: "status message",
				},
			},
			output: outputType{
				expectedNodePool: NodePool{
					Name:          "node-pool-name-3",
					Status:        NodePoolStatusUnknown,
					StatusMessage: "retrieving node pool information failed: test error",
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualNodePool := NewNodePoolFromCFStackDescriptionError(
				testCase.input.err, testCase.input.existingNodePool,
			)

			require.Equal(t, testCase.output.expectedNodePool, actualNodePool)
		})
	}
}

func TestNewNodePoolFromCFStack(t *testing.T) {
	type inputType struct {
		labels map[string]string
		name   string
		stack  *cloudformation.Stack
	}

	type outputType struct {
		expectedNodePool NodePool
	}

	testCases := []struct {
		caseName string
		input    inputType
		output   outputType
	}{
		{
			caseName: "parse failed success",
			input: inputType{
				labels: nil,
				name:   "node-pool",
				stack: &cloudformation.Stack{
					Parameters: []*cloudformation.Parameter{
						{
							ParameterKey:   aws.String("ClusterAutoscalerEnabled"),
							ParameterValue: aws.String("not-a-bool"),
						},
						{
							ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
							ParameterValue: aws.String("2"),
						},
						{
							ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
							ParameterValue: aws.String("1"),
						},
						{
							ParameterKey:   aws.String("NodeAutoScalingInitSize"),
							ParameterValue: aws.String("1"),
						},
						{
							ParameterKey:   aws.String("NodeImageId"),
							ParameterValue: aws.String("ami-0123456789"),
						},
						{
							ParameterKey:   aws.String("NodeInstanceType"),
							ParameterValue: aws.String("t2.small"),
						},
						{
							ParameterKey:   aws.String("NodeSpotPrice"),
							ParameterValue: aws.String("0.02"),
						},
						{
							ParameterKey:   aws.String("NodeVolumeSize"),
							ParameterValue: aws.String("20"),
						},
						{
							ParameterKey:   aws.String("Subnets"),
							ParameterValue: aws.String("subnet-0123456789"),
						},
					},
				},
			},
			output: outputType{
				expectedNodePool: NodePool{
					Name:          "node-pool",
					Status:        NodePoolStatusError,
					StatusMessage: "parsing cloudformation stack parameter failed: strconv.ParseBool: parsing \"not-a-bool\": invalid syntax",
				},
			},
		},
		{
			caseName: "parsed success",
			input: inputType{
				labels: map[string]string{
					"key": "value",
				},
				name: "node-pool",
				stack: &cloudformation.Stack{
					Parameters: []*cloudformation.Parameter{
						{
							ParameterKey:   aws.String("ClusterAutoscalerEnabled"),
							ParameterValue: aws.String("true"),
						},
						{
							ParameterKey:   aws.String("NodeAutoScalingGroupMaxSize"),
							ParameterValue: aws.String("2"),
						},
						{
							ParameterKey:   aws.String("NodeAutoScalingGroupMinSize"),
							ParameterValue: aws.String("1"),
						},
						{
							ParameterKey:   aws.String("NodeAutoScalingInitSize"),
							ParameterValue: aws.String("1"),
						},
						{
							ParameterKey:   aws.String("NodeImageId"),
							ParameterValue: aws.String("ami-0123456789"),
						},
						{
							ParameterKey:   aws.String("NodeInstanceType"),
							ParameterValue: aws.String("t2.small"),
						},
						{
							ParameterKey:   aws.String("NodeSpotPrice"),
							ParameterValue: aws.String("0.02"),
						},
						{
							ParameterKey:   aws.String("NodeVolumeSize"),
							ParameterValue: aws.String("20"),
						},
						{
							ParameterKey:   aws.String("Subnets"),
							ParameterValue: aws.String("subnet-0123456789"),
						},
					},
					StackStatus:       aws.String(cloudformation.StackStatusCreateComplete),
					StackStatusReason: aws.String("this is a test"),
				},
			},
			output: outputType{
				expectedNodePool: NodePool{
					Name: "node-pool",
					Labels: map[string]string{
						"key": "value",
					},
					Size: 1,
					Autoscaling: Autoscaling{
						Enabled: true,
						MinSize: 1,
						MaxSize: 2,
					},
					VolumeSize:    20,
					InstanceType:  "t2.small",
					Image:         "ami-0123456789",
					SpotPrice:     "0.02",
					SubnetID:      "subnet-0123456789",
					Status:        NodePoolStatusReady,
					StatusMessage: "this is a test",
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualNodePool := NewNodePoolFromCFStack(testCase.input.name, testCase.input.labels, testCase.input.stack)

			require.Equal(t, testCase.output.expectedNodePool, actualNodePool)
		})
	}
}

func TestNewNodePoolWithNoValues(t *testing.T) {
	type inputType struct {
		name          string
		status        NodePoolStatus
		statusMessage string
	}

	type outputType struct {
		expectedNodePool NodePool
	}

	testCases := []struct {
		caseName string
		input    inputType
		output   outputType
	}{
		{
			caseName: "arbitrary message success",
			input: inputType{
				name:          "node-pool",
				status:        NodePoolStatusError,
				statusMessage: "status message",
			},
			output: outputType{
				expectedNodePool: NodePool{
					Name:          "node-pool",
					Status:        NodePoolStatusError,
					StatusMessage: "status message",
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseName, func(t *testing.T) {
			actualNodePool := NewNodePoolWithNoValues(
				testCase.input.name, testCase.input.status, testCase.input.statusMessage,
			)

			require.Equal(t, testCase.output.expectedNodePool, actualNodePool)
		})
	}
}

func TestServiceListNodePools(t *testing.T) {
	exampleClusterID := uint(1)
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
	exampleExistingNodePools := map[string]ExistingNodePool{
		"cluster-node-pool-name-2": {
			Name:          "cluster-node-pool-name-2",
			StackID:       "cluster-node-pool-name-2/stack-id",
			Status:        NodePoolStatusReady,
			StatusMessage: "",
		},
		"cluster-node-pool-name-3": {
			Name:          "cluster-node-pool-name-3",
			StackID:       "cluster-node-pool-name-3/stack-id",
			Status:        NodePoolStatusReady,
			StatusMessage: "",
		},
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
			caseName: "ExistingNodePoolsError",
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
				nodePoolStoreMock.On("ListNodePools",
					functionCallArguments.ctx,
					exampleCluster.OrganizationID,
					exampleCluster.ID,
					exampleCluster.Name,
				).Return(map[string]ExistingNodePool{}, errors.New("ExistingNodePoolsError"))
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
				nodePoolStoreMock.On("ListNodePools",
					functionCallArguments.ctx,
					exampleCluster.OrganizationID,
					exampleCluster.ID,
					exampleCluster.Name,
				).Return(exampleExistingNodePools, nil)

				nodePoolManagerMock := constructionArguments.nodePoolManager.(*MockNodePoolManager)
				nodePoolManagerMock.On("ListNodePools", functionCallArguments.ctx, exampleCluster, exampleExistingNodePools).Return(nil, errors.New("NodePoolsError"))
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
				nodePoolStoreMock.On("ListNodePools",
					functionCallArguments.ctx,
					exampleCluster.OrganizationID,
					exampleCluster.ID,
					exampleCluster.Name,
				).Return(exampleExistingNodePools, nil)

				nodePoolManagerMock := constructionArguments.nodePoolManager.(*MockNodePoolManager)
				nodePoolManagerMock.On("ListNodePools", functionCallArguments.ctx, exampleCluster, exampleExistingNodePools).Return(exampleNodePools, nil)
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
