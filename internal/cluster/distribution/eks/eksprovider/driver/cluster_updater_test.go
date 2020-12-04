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

package driver

import (
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/ekscluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	pkgerrors "github.com/banzaicloud/pipeline/pkg/errors"
)

func TestNewASGsFromRequestedUpdatedNodePools(t *testing.T) {
	type inputType struct {
		requestedUpdatedNodePools map[string]*ekscluster.NodePool
		currentNodePools          []*eksmodel.AmazonNodePoolsModel
	}

	testCases := []struct {
		caseDescription          string
		input                    inputType
		expectedUpdatedNodePools []workflow.AutoscaleGroup
	}{
		{
			caseDescription: "3 updated node pools -> success",
			input: inputType{
				requestedUpdatedNodePools: map[string]*ekscluster.NodePool{
					"pool1": {
						InstanceType:     "instance-type-1",
						SpotPrice:        "0.1",
						Autoscaling:      true,
						MinCount:         1,
						MaxCount:         1,
						Count:            1,
						VolumeEncryption: nil,
						VolumeSize:       1,
						Image:            "image-1",
						Labels: map[string]string{
							"label-1": "value-1",
						},
						SecurityGroups: []string{
							"security-group-1",
							"security-group-11",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-1",
							Cidr:             "cidr-1",
							AvailabilityZone: "availability-zone-1",
						},
					},
					"pool2": {
						InstanceType: "instance-type-2",
						SpotPrice:    "0.2",
						Autoscaling:  false,
						MinCount:     0,
						MaxCount:     0,
						Count:        2,
						VolumeEncryption: &ekscluster.NodePoolVolumeEncryption{
							Enabled: true,
						},
						VolumeSize: 2,
						Image:      "image-2",
						Labels: map[string]string{
							"label-2": "value-2",
						},
						SecurityGroups: []string{
							"security-group-2",
							"security-group-22",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-2",
							Cidr:             "cidr-2",
							AvailabilityZone: "availability-zone-2",
						},
					},
					"pool3": {
						InstanceType: "instance-type-3",
						SpotPrice:    "0.3",
						Autoscaling:  true,
						MinCount:     3,
						MaxCount:     3,
						Count:        3,
						VolumeEncryption: &ekscluster.NodePoolVolumeEncryption{
							Enabled:          true,
							EncryptionKeyARN: "encryption-key-arn-3",
						},
						VolumeSize: 3,
						Image:      "image-3",
						Labels: map[string]string{
							"label-3": "value-3",
						},
						SecurityGroups: []string{
							"security-group-3",
							"security-group-33",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-3",
							Cidr:             "cidr-3",
							AvailabilityZone: "availability-zone-3",
						},
					},
				},
				currentNodePools: []*eksmodel.AmazonNodePoolsModel{
					{
						Name:      "pool1",
						CreatedBy: 1,
					},
					{
						Name:      "pool2",
						CreatedBy: 2,
					},
					{
						Name:      "pool3",
						CreatedBy: 3,
					},
				},
			},
			expectedUpdatedNodePools: []workflow.AutoscaleGroup{
				{
					Name:                 "pool1",
					NodeSpotPrice:        "0.1",
					Autoscaling:          true,
					NodeMinCount:         1,
					NodeMaxCount:         1,
					Count:                1,
					NodeVolumeEncryption: nil,
					NodeVolumeSize:       1,
					NodeImage:            "image-1",
					NodeInstanceType:     "instance-type-1",
					SecurityGroups: []string{
						"security-group-1",
						"security-group-11",
					},
					Labels: map[string]string{
						"label-1": "value-1",
					},
					Delete:    false,
					Create:    false,
					CreatedBy: 1,
				},
				{
					Name:          "pool2",
					NodeSpotPrice: "0.2",
					Autoscaling:   false,
					NodeMinCount:  0,
					NodeMaxCount:  0,
					Count:         2,
					NodeVolumeEncryption: &eks.NodePoolVolumeEncryption{
						Enabled: true,
					},
					NodeVolumeSize:   2,
					NodeImage:        "image-2",
					NodeInstanceType: "instance-type-2",
					SecurityGroups: []string{
						"security-group-2",
						"security-group-22",
					},
					Labels: map[string]string{
						"label-2": "value-2",
					},
					Delete:    false,
					Create:    false,
					CreatedBy: 2,
				},
				{
					Name:          "pool3",
					NodeSpotPrice: "0.3",
					Autoscaling:   true,
					NodeMinCount:  3,
					NodeMaxCount:  3,
					Count:         3,
					NodeVolumeEncryption: &eks.NodePoolVolumeEncryption{
						Enabled:          true,
						EncryptionKeyARN: "encryption-key-arn-3",
					},
					NodeVolumeSize:   3,
					NodeImage:        "image-3",
					NodeInstanceType: "instance-type-3",
					SecurityGroups: []string{
						"security-group-3",
						"security-group-33",
					},
					Labels: map[string]string{
						"label-3": "value-3",
					},
					Delete:    false,
					Create:    false,
					CreatedBy: 3,
				},
			},
		},
		{
			caseDescription: "empty requested updated node pools, empty current node pools -> empty success",
			input: inputType{
				requestedUpdatedNodePools: nil,
				currentNodePools:          nil,
			},
			expectedUpdatedNodePools: []workflow.AutoscaleGroup{},
		},
		{
			caseDescription: "empty requested updated node pools, not empty current node pools -> empty success",
			input: inputType{
				requestedUpdatedNodePools: nil,
				currentNodePools: []*eksmodel.AmazonNodePoolsModel{
					{
						CreatedBy: 1,
					},
					{
						CreatedBy: 2,
					},
					{
						CreatedBy: 3,
					},
				},
			},
			expectedUpdatedNodePools: []workflow.AutoscaleGroup{},
		},
		{
			caseDescription: "not empty requested updated node pools, empty current node pools -> not empty success with 0 creator IDs",
			input: inputType{
				requestedUpdatedNodePools: map[string]*ekscluster.NodePool{
					"pool1": {
						InstanceType: "instance-type-1",
						SpotPrice:    "0.1",
						Autoscaling:  true,
						MinCount:     1,
						MaxCount:     1,
						Count:        1,
						VolumeSize:   1,
						Image:        "image-1",
						Labels: map[string]string{
							"label-1": "value-1",
						},
						SecurityGroups: []string{
							"security-group-1",
							"security-group-11",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-1",
							Cidr:             "cidr-1",
							AvailabilityZone: "availability-zone-1",
						},
					},
				},
				currentNodePools: nil,
			},
			expectedUpdatedNodePools: []workflow.AutoscaleGroup{
				{
					Name:             "pool1",
					NodeSpotPrice:    "0.1",
					Autoscaling:      true,
					NodeMinCount:     1,
					NodeMaxCount:     1,
					Count:            1,
					NodeVolumeSize:   1,
					NodeImage:        "image-1",
					NodeInstanceType: "instance-type-1",
					SecurityGroups: []string{
						"security-group-1",
						"security-group-11",
					},
					Labels: map[string]string{
						"label-1": "value-1",
					},
					Delete:    false,
					Create:    false,
					CreatedBy: 0, // Note: not available from current node pools.
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualUpdatedNodePools := newASGsFromRequestedUpdatedNodePools(
				testCase.input.requestedUpdatedNodePools,
				testCase.input.currentNodePools,
			)

			require.Equal(t, testCase.expectedUpdatedNodePools, actualUpdatedNodePools)
		})
	}
}

func TestNewClusterUpdateSubnetsFromModels(t *testing.T) {
	type outputType struct {
		expectedClusterSubnets []workflow.Subnet
		expectedErr            error
	}

	testCases := []struct {
		caseDescription          string
		inputClusterSubnetModels []*eksmodel.EKSSubnetModel
		output                   outputType
	}{
		{
			caseDescription: "3 cluster subnets -> success",
			inputClusterSubnetModels: []*eksmodel.EKSSubnetModel{
				{
					SubnetId:         aws.String("subnet-id-1"),
					Cidr:             aws.String("cidr-1"),
					AvailabilityZone: aws.String("availability-zone-1"),
				},
				{
					SubnetId:         aws.String("subnet-id-2"),
					Cidr:             aws.String("cidr-2"),
					AvailabilityZone: aws.String("availability-zone-2"),
				},
				{
					SubnetId:         aws.String("subnet-id-3"),
					Cidr:             aws.String("cidr-3"),
					AvailabilityZone: aws.String("availability-zone-3"),
				},
			},
			output: outputType{
				expectedClusterSubnets: []workflow.Subnet{
					{
						SubnetID:         "subnet-id-1",
						Cidr:             "cidr-1",
						AvailabilityZone: "availability-zone-1",
					},
					{
						SubnetID:         "subnet-id-2",
						Cidr:             "cidr-2",
						AvailabilityZone: "availability-zone-2",
					},
					{
						SubnetID:         "subnet-id-3",
						Cidr:             "cidr-3",
						AvailabilityZone: "availability-zone-3",
					},
				},
				expectedErr: nil,
			},
		},
		{
			caseDescription:          "empty cluster subnet models -> error",
			inputClusterSubnetModels: nil,
			output: outputType{
				expectedClusterSubnets: nil,
				expectedErr:            errors.New("no cluster subnet is available"),
			},
		},
		{
			caseDescription: "not existing cluster subnet model -> error",
			inputClusterSubnetModels: []*eksmodel.EKSSubnetModel{
				{
					SubnetId:         aws.String("subnet-id-1"),
					Cidr:             aws.String("cidr-1"),
					AvailabilityZone: aws.String("availability-zone-1"),
				},
				{
					SubnetId:         aws.String(""),
					Cidr:             aws.String("cidr-2"),
					AvailabilityZone: aws.String("availability-zone-2"),
				},
				{
					SubnetId:         aws.String("subnet-id-3"),
					Cidr:             aws.String("cidr-3"),
					AvailabilityZone: aws.String("availability-zone-3"),
				},
			},
			output: outputType{
				expectedClusterSubnets: nil,
				expectedErr: errors.New(
					"cluster subnet CIDR cidr-2 lacks an ID and subnet creation is not supported during cluster update",
				),
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualClusterSubnets, actualErr := newClusterUpdateSubnetsFromModels(testCase.inputClusterSubnetModels)

			if testCase.output.expectedErr == nil {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, testCase.output.expectedErr.Error())
			}
			require.Equal(t, testCase.output.expectedClusterSubnets, actualClusterSubnets)
		})
	}
}

func TestNewNodePoolNamesFromRequestedDeletedNodePools(t *testing.T) {
	testCases := []struct {
		caseDescription       string
		inputNodePoolModels   map[string]*eksmodel.AmazonNodePoolsModel
		expectedNodePoolNames []string
	}{
		{
			caseDescription: "3 node pool models -> success",
			inputNodePoolModels: map[string]*eksmodel.AmazonNodePoolsModel{
				"pool1": nil,
				"pool2": nil,
				"pool3": nil,
			},
			expectedNodePoolNames: []string{
				"pool1",
				"pool2",
				"pool3",
			},
		},
		{
			caseDescription:       "empty node pool models -> empty success",
			inputNodePoolModels:   nil,
			expectedNodePoolNames: []string{},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualNodePoolNames := newNodePoolNamesFromRequestedDeletedNodePools(testCase.inputNodePoolModels)

			require.Equal(t, testCase.expectedNodePoolNames, actualNodePoolNames)
		})
	}
}

func TestNewNodePoolsFromUpdateRequest(t *testing.T) {
	type inputType struct {
		currentNodePools   []*eksmodel.AmazonNodePoolsModel
		requestedNodePools map[string]*ekscluster.NodePool
	}

	type outputType struct {
		expectedRequestedDeletedNodePools map[string]*eksmodel.AmazonNodePoolsModel
		expectedRequestedNewNodePools     map[string]*ekscluster.NodePool
		expectedRequestedUpdatedNodePools map[string]*ekscluster.NodePool
		expectedErr                       error
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "deleted node pools and new node pools without spot price -> success",
			input: inputType{
				currentNodePools: []*eksmodel.AmazonNodePoolsModel{
					{
						Name: "delete-pool-1",
					},
					{
						ID:               3,
						CreatedAt:        time.Time{},
						CreatedBy:        3,
						ClusterID:        3,
						Name:             "update-pool-3",
						StackID:          "stack-id-3",
						NodeSpotPrice:    "0.3",
						Autoscaling:      true,
						NodeMinCount:     3,
						NodeMaxCount:     3,
						Count:            3,
						NodeVolumeSize:   3,
						NodeImage:        "image-3",
						NodeInstanceType: "instance-type-3",
						Status:           eks.NodePoolStatusReady,
						StatusMessage:    "",
						Labels: map[string]string{
							"label-3": "value-3",
						},
						Delete: false,
					},
				},
				requestedNodePools: map[string]*ekscluster.NodePool{
					"new-pool-2": {
						InstanceType:     "instance-type-2",
						SpotPrice:        "0.2",
						Autoscaling:      false,
						MinCount:         0,
						MaxCount:         0,
						Count:            2,
						VolumeEncryption: nil,
						VolumeSize:       2,
						Image:            "image-2",
						Labels: map[string]string{
							"label-2": "value-2",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-2",
							Cidr:             "cidr-2",
							AvailabilityZone: "availability-zone-2",
						},
					},
					"update-pool-3": {
						InstanceType: "instance-type-3",
						SpotPrice:    "0.3",
						Autoscaling:  true,
						MinCount:     3,
						MaxCount:     3,
						Count:        3,
						VolumeEncryption: &ekscluster.NodePoolVolumeEncryption{
							Enabled: true,
						},
						VolumeSize: 33333,
						Image:      "image-3",
						Labels: map[string]string{
							"label-3": "value-3",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-3",
							Cidr:             "cidr-3",
							AvailabilityZone: "availability-zone-3",
						},
					},
					"new-pool-4": {
						InstanceType: "instance-type-4",
						SpotPrice:    "",
						Autoscaling:  false,
						MinCount:     0,
						MaxCount:     0,
						Count:        4,
						VolumeEncryption: &ekscluster.NodePoolVolumeEncryption{
							Enabled:          true,
							EncryptionKeyARN: "encryption-key-arn-4",
						},
						VolumeSize: 4,
						Image:      "image-4",
						Labels: map[string]string{
							"label-4": "value-4",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-4",
							Cidr:             "cidr-4",
							AvailabilityZone: "availability-zone-4",
						},
					},
				},
			},
			output: outputType{
				expectedRequestedDeletedNodePools: map[string]*eksmodel.AmazonNodePoolsModel{
					"delete-pool-1": {
						Name: "delete-pool-1",
					},
				},
				expectedRequestedNewNodePools: map[string]*ekscluster.NodePool{
					"new-pool-2": {
						InstanceType:     "instance-type-2",
						SpotPrice:        "0.2",
						Autoscaling:      false,
						MinCount:         0,
						MaxCount:         0,
						Count:            2,
						VolumeEncryption: nil,
						VolumeSize:       2,
						Image:            "image-2",
						Labels: map[string]string{
							"label-2": "value-2",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-2",
							Cidr:             "cidr-2",
							AvailabilityZone: "availability-zone-2",
						},
					},
					"new-pool-4": {
						InstanceType: "instance-type-4",
						SpotPrice:    "0.0",
						Autoscaling:  false,
						MinCount:     0,
						MaxCount:     0,
						Count:        4,
						VolumeEncryption: &ekscluster.NodePoolVolumeEncryption{
							Enabled:          true,
							EncryptionKeyARN: "encryption-key-arn-4",
						},
						VolumeSize: 4,
						Image:      "image-4",
						Labels: map[string]string{
							"label-4": "value-4",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-4",
							Cidr:             "cidr-4",
							AvailabilityZone: "availability-zone-4",
						},
					},
				},
				expectedRequestedUpdatedNodePools: map[string]*ekscluster.NodePool{
					"update-pool-3": {
						InstanceType: "instance-type-3",
						SpotPrice:    "0.3",
						Autoscaling:  true,
						MinCount:     3,
						MaxCount:     3,
						Count:        3,
						VolumeEncryption: &ekscluster.NodePoolVolumeEncryption{
							Enabled: true,
						},
						VolumeSize: 33333,
						Image:      "image-3",
						Labels: map[string]string{
							"label-3": "value-3",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-3",
							Cidr:             "cidr-3",
							AvailabilityZone: "availability-zone-3",
						},
					},
				},
				expectedErr: nil,
			},
		},
		{
			caseDescription: "empty instance type field -> error",
			input: inputType{
				currentNodePools: nil,
				requestedNodePools: map[string]*ekscluster.NodePool{
					"new-pool-2": {
						InstanceType: "",
						SpotPrice:    "0.2",
						Autoscaling:  false,
						MinCount:     0,
						MaxCount:     0,
						Count:        2,
						VolumeSize:   2,
						Image:        "image-2",
						Labels: map[string]string{
							"label-2": "value-2",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-2",
							Cidr:             "cidr-2",
							AvailabilityZone: "availability-zone-2",
						},
					},
				},
			},
			output: outputType{
				expectedRequestedDeletedNodePools: nil,
				expectedRequestedNewNodePools:     nil,
				expectedRequestedUpdatedNodePools: nil,
				expectedErr:                       pkgerrors.ErrorInstancetypeFieldIsEmpty,
			},
		},
		{
			caseDescription: "empty image field -> error",
			input: inputType{
				currentNodePools: nil,
				requestedNodePools: map[string]*ekscluster.NodePool{
					"new-pool-2": {
						InstanceType: "instance-type-2",
						SpotPrice:    "0.2",
						Autoscaling:  false,
						MinCount:     0,
						MaxCount:     0,
						Count:        2,
						VolumeSize:   2,
						Image:        "",
						Labels: map[string]string{
							"label-2": "value-2",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-2",
							Cidr:             "cidr-2",
							AvailabilityZone: "availability-zone-2",
						},
					},
				},
			},
			output: outputType{
				expectedRequestedDeletedNodePools: nil,
				expectedRequestedNewNodePools:     nil,
				expectedRequestedUpdatedNodePools: nil,
				expectedErr:                       pkgerrors.ErrorAmazonImageFieldIsEmpty,
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualRequestedDeletedNodePools, actualRequestedNewNodePools, actualRequestedUpdatedNodePools, actualErr :=
				newNodePoolsFromUpdateRequest(testCase.input.currentNodePools, testCase.input.requestedNodePools)

			if testCase.output.expectedErr == nil {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, testCase.output.expectedErr.Error())
			}
			require.Equal(t, testCase.output.expectedRequestedDeletedNodePools, actualRequestedDeletedNodePools)
			require.Equal(t, testCase.output.expectedRequestedNewNodePools, actualRequestedNewNodePools)
			require.Equal(t, testCase.output.expectedRequestedUpdatedNodePools, actualRequestedUpdatedNodePools)
		})
	}
}

func TestNewNodePoolsFromRequestedNewNodePools(t *testing.T) {
	type inputType struct {
		requestedNewNodePools map[string]*ekscluster.NodePool
		newNodePoolSubnetIDs  map[string][]string
	}

	type outputType struct {
		expectedNewNodePools []eks.NewNodePool
		expectedErr          error
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "success",
			input: inputType{
				requestedNewNodePools: map[string]*ekscluster.NodePool{
					"pool-1": {
						InstanceType:     "instance-type-1",
						SpotPrice:        "0.1",
						Autoscaling:      true,
						MinCount:         1,
						MaxCount:         1,
						Count:            1,
						VolumeEncryption: nil,
						VolumeSize:       1,
						Image:            "image-1",
						Labels: map[string]string{
							"label-1": "value-1",
						},
						SecurityGroups: []string{
							"security-group-1",
							"security-group-11",
						},
						Subnet: &ekscluster.Subnet{
							SubnetId:         "subnet-id-1",
							Cidr:             "cidr-1",
							AvailabilityZone: "availability-zone-1",
						},
					},
					"pool-2": {
						InstanceType: "instance-type-2",
						SpotPrice:    "0.2",
						Autoscaling:  false,
						MinCount:     0,
						MaxCount:     0,
						Count:        2,
						VolumeEncryption: &ekscluster.NodePoolVolumeEncryption{
							Enabled: true,
						},
						VolumeSize: 2,
						Image:      "image-2",
						Labels: map[string]string{
							"label-2": "value-2",
						},
						Subnet: nil,
					},
					"pool-3": {
						InstanceType: "instance-type-3",
						SpotPrice:    "0.3",
						Autoscaling:  false,
						MinCount:     0,
						MaxCount:     0,
						Count:        3,
						VolumeEncryption: &ekscluster.NodePoolVolumeEncryption{
							Enabled:          true,
							EncryptionKeyARN: "encryption-key-arn-3",
						},
						VolumeSize: 3,
						Image:      "image-3",
						Labels: map[string]string{
							"label-3": "value-3",
						},
						Subnet: &ekscluster.Subnet{
							Cidr: "cidr-3",
						},
					},
				},
				newNodePoolSubnetIDs: map[string][]string{
					"pool-1": {"subnet-id-1"},
					"pool-2": {"subnet-id-2"},
					"pool-3": {"subnet-id-3"},
				},
			},
			output: outputType{
				expectedNewNodePools: []eks.NewNodePool{
					{
						Name: "pool-1",
						Labels: map[string]string{
							"label-1": "value-1",
						},
						Size: 1,
						Autoscaling: eks.Autoscaling{
							Enabled: true,
							MinSize: 1,
							MaxSize: 1,
						},
						VolumeEncryption: nil,
						VolumeSize:       1,
						InstanceType:     "instance-type-1",
						Image:            "image-1",
						SpotPrice:        "0.1",
						SecurityGroups: []string{
							"security-group-1",
							"security-group-11",
						},
						SubnetID: "subnet-id-1",
					},
					{
						Name: "pool-2",
						Labels: map[string]string{
							"label-2": "value-2",
						},
						Size: 2,
						Autoscaling: eks.Autoscaling{
							Enabled: false,
							MinSize: 0,
							MaxSize: 0,
						},
						VolumeEncryption: &eks.NodePoolVolumeEncryption{
							Enabled: true,
						},
						VolumeSize:   2,
						InstanceType: "instance-type-2",
						Image:        "image-2",
						SpotPrice:    "0.2",
						SubnetID:     "subnet-id-2",
					},
					{
						Name: "pool-3",
						Labels: map[string]string{
							"label-3": "value-3",
						},
						Size: 3,
						Autoscaling: eks.Autoscaling{
							Enabled: false,
							MinSize: 0,
							MaxSize: 0,
						},
						VolumeEncryption: &eks.NodePoolVolumeEncryption{
							Enabled:          true,
							EncryptionKeyARN: "encryption-key-arn-3",
						},
						VolumeSize:   3,
						InstanceType: "instance-type-3",
						Image:        "image-3",
						SpotPrice:    "0.3",
						SubnetID:     "subnet-id-3",
					},
				},
				expectedErr: nil,
			},
		},
		{
			caseDescription: "nil new subnet ID map -> error",
			input: inputType{
				requestedNewNodePools: map[string]*ekscluster.NodePool{
					"pool-1": {},
				},
				newNodePoolSubnetIDs: nil,
			},
			output: outputType{
				expectedNewNodePools: nil,
				expectedErr:          errors.New("nil new subnet ID map"),
			},
		},
		{
			caseDescription: "missing subnet ID -> error",
			input: inputType{
				requestedNewNodePools: map[string]*ekscluster.NodePool{
					"pool-1": {},
				},
				newNodePoolSubnetIDs: map[string][]string{},
			},
			output: outputType{
				expectedNewNodePools: nil,
				expectedErr:          errors.New("no subnet ID specified for node pool pool-1"),
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualNewNodePools, actualErr := newNodePoolsFromRequestedNewNodePools(
				testCase.input.requestedNewNodePools,
				testCase.input.newNodePoolSubnetIDs,
			)

			if testCase.output.expectedErr == nil {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, testCase.output.expectedErr.Error())
			}
			require.Equal(t, testCase.output.expectedNewNodePools, actualNewNodePools)
		})
	}
}

func TestNewNodePoolSubnetIDsFromRequestedNewNodePools(t *testing.T) {
	type inputType struct {
		requestedNewNodePools map[string]*ekscluster.NodePool
		clusterSubnets        []workflow.Subnet
	}

	type outputType struct {
		expectedNewNodePoolSubnetIDs map[string][]string
		expectedErr                  error
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "success",
			input: inputType{
				requestedNewNodePools: map[string]*ekscluster.NodePool{
					"default": {
						Subnet: nil,
					},
					"id": {
						Subnet: &ekscluster.Subnet{
							SubnetId: "subnet-id-2",
						},
					},
					"cidr": {
						Subnet: &ekscluster.Subnet{
							Cidr: "cidr-3",
						},
					},
				},
				clusterSubnets: []workflow.Subnet{
					{
						SubnetID:         "subnet-id-1",
						Cidr:             "cidr-1",
						AvailabilityZone: "availability-zone-1",
					},
					{
						SubnetID:         "subnet-id-2",
						Cidr:             "cidr-2",
						AvailabilityZone: "availability-zone-2",
					},
					{
						SubnetID:         "subnet-id-3",
						Cidr:             "cidr-3",
						AvailabilityZone: "availability-zone-3",
					},
				},
			},
			output: outputType{
				expectedNewNodePoolSubnetIDs: map[string][]string{
					"default": {"subnet-id-1"},
					"id":      {"subnet-id-2"},
					"cidr":    {"subnet-id-3"},
				},
				expectedErr: nil,
			},
		},
		{
			caseDescription: "empty cluster subnet list -> error",
			input: inputType{
				requestedNewNodePools: map[string]*ekscluster.NodePool{},
				clusterSubnets:        nil,
			},
			output: outputType{
				expectedNewNodePoolSubnetIDs: nil,
				expectedErr:                  errors.New("empty cluster subnet list"),
			},
		},
		{
			caseDescription: "missing subnet ID and CIDR -> error",
			input: inputType{
				requestedNewNodePools: map[string]*ekscluster.NodePool{
					"missing-subnet-id-and-cidr": {
						Subnet: &ekscluster.Subnet{},
					},
				},
				clusterSubnets: []workflow.Subnet{
					{
						SubnetID:         "subnet-id-1",
						Cidr:             "cidr-1",
						AvailabilityZone: "availability-zone-1",
					},
				},
			},
			output: outputType{
				expectedNewNodePoolSubnetIDs: nil,
				expectedErr: errors.New(
					"node pool missing-subnet-id-and-cidr is missing both subnet ID and CIDR: " +
						"&{SubnetId: Cidr: AvailabilityZone:}",
				),
			},
		},
		{
			caseDescription: "node pool subnet ID not found -> error",
			input: inputType{
				requestedNewNodePools: map[string]*ekscluster.NodePool{
					"not-existing-subnet-id": {
						Subnet: &ekscluster.Subnet{
							SubnetId: "not-existing-subnet-id",
						},
					},
				},
				clusterSubnets: []workflow.Subnet{
					{
						SubnetID:         "subnet-id-1",
						Cidr:             "cidr-1",
						AvailabilityZone: "availability-zone-1",
					},
				},
			},
			output: outputType{
				expectedNewNodePoolSubnetIDs: nil,
				expectedErr: errors.New(
					"subnet ID not found for node pool not-existing-subnet-id with subnet " +
						"&{SubnetId:not-existing-subnet-id Cidr: AvailabilityZone:}",
				),
			},
		},
		{
			caseDescription: "node pool subnet CIDR not found -> error",
			input: inputType{
				requestedNewNodePools: map[string]*ekscluster.NodePool{
					"not-existing-subnet-cidr": {
						Subnet: &ekscluster.Subnet{
							Cidr: "not-existing-subnet-cidr",
						},
					},
				},
				clusterSubnets: []workflow.Subnet{
					{
						SubnetID:         "subnet-id-1",
						Cidr:             "cidr-1",
						AvailabilityZone: "availability-zone-1",
					},
				},
			},
			output: outputType{
				expectedNewNodePoolSubnetIDs: nil,
				expectedErr: errors.New(
					"subnet ID not found for node pool not-existing-subnet-cidr with subnet " +
						"&{SubnetId: Cidr:not-existing-subnet-cidr AvailabilityZone:}",
				),
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualNewNodePoolSubnetIDs, actualErr := newNodePoolSubnetIDsFromRequestedNewNodePools(
				testCase.input.requestedNewNodePools,
				testCase.input.clusterSubnets,
			)

			if testCase.output.expectedErr == nil {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, testCase.output.expectedErr.Error())
			}
			require.Equal(t, testCase.output.expectedNewNodePoolSubnetIDs, actualNewNodePoolSubnetIDs)
		})
	}
}
