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

package clusteradapter

import (
	"context"
	"strings"
	"testing"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
)

func TestEksServiceCreateNodePool(t *testing.T) {
	type inputType struct {
		service     eksService
		ctx         context.Context
		clusterID   uint
		rawNodePool cluster.NewRawNodePool
	}

	testCases := []struct {
		caseName      string
		expectedError error
		input         inputType
	}{
		{
			caseName:      "decode error -> error",
			expectedError: errors.New("invalid node pool creation request"),
			input: inputType{
				service: eksService{
					service: &eks.MockService{},
				},
				ctx:       context.Background(),
				clusterID: 1,
				rawNodePool: cluster.NewRawNodePool{
					"name": false,
				},
			},
		},
		{
			caseName:      "create node pool error -> error",
			expectedError: errors.New("create node pool error"),
			input: inputType{
				service: eksService{
					service: &eks.MockService{},
				},
				ctx:         context.Background(),
				clusterID:   1,
				rawNodePool: cluster.NewRawNodePool{},
			},
		},
		{
			caseName:      "success",
			expectedError: nil,
			input: inputType{
				service: eksService{
					service: &eks.MockService{},
				},
				ctx:         context.Background(),
				clusterID:   1,
				rawNodePool: cluster.NewRawNodePool{},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			var newNodePool eks.NewNodePool
			if testCase.expectedError == nil ||
				!strings.HasPrefix(testCase.expectedError.Error(), "invalid node pool creation request") {
				err := mapstructure.Decode(testCase.input.rawNodePool, &newNodePool)
				require.NoError(t, err)
			}

			createNodePoolMock := testCase.input.service.service.(*eks.MockService).On(
				"CreateNodePool", testCase.input.ctx, testCase.input.clusterID, newNodePool,
			)
			if testCase.expectedError != nil &&
				strings.HasPrefix(testCase.expectedError.Error(), "create node pool error") {
				createNodePoolMock.Return(testCase.expectedError)
			} else {
				createNodePoolMock.Return(nil)
			}

			actualError := testCase.input.service.CreateNodePool(
				testCase.input.ctx,
				testCase.input.clusterID,
				testCase.input.rawNodePool,
			)

			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}

func TestEksServiceDeleteNodePool(t *testing.T) {
	type inputType struct {
		clusterID    uint
		nodePoolName string
		service      eksService
	}

	type outputType struct {
		expectedError     error
		expectedIsDeleted bool
	}

	testCases := []struct {
		caseName string
		input    inputType
		output   outputType
	}{
		{
			caseName: "error",
			input: inputType{
				clusterID:    1,
				nodePoolName: "node-pool-name",
				service: eksService{
					service: &eks.MockService{},
				},
			},
			output: outputType{
				expectedError:     errors.New("test error: GetCluster"),
				expectedIsDeleted: false,
			},
		},
		{
			caseName: "already deleted",
			input: inputType{
				clusterID:    1,
				nodePoolName: "node-pool-name",
				service: eksService{
					service: &eks.MockService{},
				},
			},
			output: outputType{
				expectedError:     nil,
				expectedIsDeleted: true,
			},
		},
		{
			caseName: "deleting",
			input: inputType{
				clusterID:    1,
				nodePoolName: "node-pool-name",
				service: eksService{
					service: &eks.MockService{},
				},
			},
			output: outputType{
				expectedError:     nil,
				expectedIsDeleted: false,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			testCase.input.service.service.(*eks.MockService).On(
				"DeleteNodePool", context.Background(), testCase.input.clusterID, testCase.input.nodePoolName,
			).Return(testCase.output.expectedIsDeleted, testCase.output.expectedError).Once()

			actualIsDeleted, actualError := testCase.input.service.DeleteNodePool(
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

func TestEksServiceListNodePools(t *testing.T) {
	exampleEKSNodePools := []eks.NodePool{
		{
			Name: "cluster-node-pool-name-2",
			Labels: map[string]string{
				"label-1": "value-1",
				"label-2": "value-2",
			},
			Size: 4,
			Autoscaling: eks.Autoscaling{
				Enabled: true,
				MinSize: 1,
				MaxSize: 2,
			},
			VolumeEncryption: &eks.NodePoolVolumeEncryption{
				Enabled:          true,
				EncryptionKeyARN: "encryption-key-arn",
			},
			VolumeSize:   40,
			InstanceType: "instance-type",
			Image:        "image",
			SecurityGroups: []string{
				"security-group-1",
				"security-group-2",
			},
			SpotPrice: "5",
		},
		{
			Name: "cluster-node-pool-name-3",
			Labels: map[string]string{
				"label-3": "value-3",
			},
			Size: 6,
			Autoscaling: eks.Autoscaling{
				Enabled: false,
				MinSize: 0,
				MaxSize: 0,
			},
			VolumeSize:   50,
			InstanceType: "instance-type",
			Image:        "image",
			SpotPrice:    "7",
		},
	}
	exampleNodePools := make([]interface{}, 0, len(exampleEKSNodePools))
	for _, eksNodePool := range exampleEKSNodePools {
		exampleNodePools = append(exampleNodePools, eksNodePool)
	}

	type constructionArgumentType struct {
		service eks.Service
	}
	type functionCallArgumentType struct {
		ctx       context.Context
		clusterID uint
	}
	testCases := []struct {
		caseName              string
		constructionArguments constructionArgumentType
		expectedNodePools     cluster.RawNodePoolList
		expectedNotNilError   bool
		functionCallArguments functionCallArgumentType
		setupMockFunction     func(constructionArgumentType, functionCallArgumentType)
	}{
		{
			caseName: "ServiceListNodePoolsFailed",
			constructionArguments: constructionArgumentType{
				service: &eks.MockService{},
			},
			expectedNodePools:   nil,
			expectedNotNilError: true,
			functionCallArguments: functionCallArgumentType{
				ctx:       context.Background(),
				clusterID: 1,
			},
			setupMockFunction: func(
				constructionArguments constructionArgumentType,
				functionCallArguments functionCallArgumentType,
			) {
				eksServiceMock := constructionArguments.service.(*eks.MockService)
				eksServiceMock.On("ListNodePools", functionCallArguments.ctx, functionCallArguments.clusterID).Return(([]eks.NodePool)(nil), errors.NewPlain("ServiceListNodePoolsFailed"))
			},
		},
		{
			caseName: "EKSServiceListNodePoolsSuccess",
			constructionArguments: constructionArgumentType{
				service: &eks.MockService{},
			},
			expectedNodePools:   exampleNodePools,
			expectedNotNilError: false,
			functionCallArguments: functionCallArgumentType{
				ctx:       context.Background(),
				clusterID: 1,
			},
			setupMockFunction: func(
				constructionArguments constructionArgumentType,
				functionCallArguments functionCallArgumentType,
			) {
				eksServiceMock := constructionArguments.service.(*eks.MockService)
				eksServiceMock.On("ListNodePools", functionCallArguments.ctx, functionCallArguments.clusterID).Return(exampleEKSNodePools, (error)(nil))
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			testCase.setupMockFunction(testCase.constructionArguments, testCase.functionCallArguments)

			object := eksService{
				service: testCase.constructionArguments.service,
			}

			actualNodePools, actualError := object.ListNodePools(
				testCase.functionCallArguments.ctx,
				testCase.functionCallArguments.clusterID,
			)

			require.Truef(t, (actualError != nil) == testCase.expectedNotNilError,
				"error value doesn't match the expectation, is expected: %+v, actual error value: %+v", testCase.expectedNotNilError, actualError)
			require.Equal(t, testCase.expectedNodePools, actualNodePools)
		})
	}
}
