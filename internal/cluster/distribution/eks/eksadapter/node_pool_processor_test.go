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
	"testing"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	pkggormtest "github.com/banzaicloud/pipeline/pkg/gorm/test"
)

type fakeImageSelector struct {
	err error
}

func (selector *fakeImageSelector) SelectImage(
	ctx context.Context,
	criteria eks.ImageSelectionCriteria,
) (string, error) {
	if selector == nil {
		return "", errors.New("selector is nil")
	}

	if selector.err != nil {
		return "", selector.err
	}

	return "image-id", nil
}

func TestNodePoolProcessorProcessNewNodePool(t *testing.T) {
	type inputType struct {
		p        *nodePoolProcessor
		ctx      context.Context
		c        cluster.Cluster
		nodePool eks.NewNodePool
	}

	type outputType struct {
		expectedError       error
		expectedNewNodePool eks.NewNodePool
	}

	testCases := []struct {
		caseDescription string
		input           inputType
		output          outputType
	}{
		{
			caseDescription: "cluster not found error -> error",
			input: inputType{
				p: &nodePoolProcessor{
					db: pkggormtest.NewFakeDatabase(t).
						CreateTablesFromEntities(
							t,
							&eksmodel.EKSClusterModel{},
							&eksmodel.EKSSubnetModel{},
						).DB,
				},
				ctx:      context.Background(),
				c:        cluster.Cluster{},
				nodePool: eks.NewNodePool{},
			},
			output: outputType{
				expectedError:       errors.New("cluster model is inconsistent"),
				expectedNewNodePool: eks.NewNodePool{},
			},
		},
		{
			caseDescription: "database error -> error",
			input: inputType{
				p: &nodePoolProcessor{
					db: pkggormtest.NewFakeDatabase(t).
						CreateTablesFromEntities(
							t,
							&eksmodel.EKSClusterModel{},
							&eksmodel.EKSSubnetModel{},
						).
						SaveEntities(
							t,
							&eksmodel.EKSClusterModel{
								ClusterID: 1,
							},
						).
						SetError(t, errors.New("test error")).DB,
				},
				ctx: context.Background(),
				c: cluster.Cluster{
					ID: 1,
				},
				nodePool: eks.NewNodePool{},
			},
			output: outputType{
				expectedError:       errors.New("failed to get cluster info: test error"),
				expectedNewNodePool: eks.NewNodePool{},
			},
		},
		{
			caseDescription: "select image error -> error",
			input: inputType{
				p: &nodePoolProcessor{
					db: pkggormtest.NewFakeDatabase(t).
						CreateTablesFromEntities(
							t,
							&eksmodel.EKSClusterModel{},
							&eksmodel.EKSSubnetModel{},
						).
						SaveEntities(
							t,
							&eksmodel.EKSClusterModel{
								ClusterID: 1,
							},
						).DB,
					imageSelector: &fakeImageSelector{
						err: errors.New("no images found matching the selection criteria"),
					},
				},
				ctx: context.Background(),
				c: cluster.Cluster{
					ID: 1,
				},
				nodePool: eks.NewNodePool{},
			},
			output: outputType{
				expectedError:       errors.New("no images found matching the selection criteria"),
				expectedNewNodePool: eks.NewNodePool{},
			},
		},
		{
			caseDescription: "subnet error -> error",
			input: inputType{
				p: &nodePoolProcessor{
					db: pkggormtest.NewFakeDatabase(t).
						CreateTablesFromEntities(
							t,
							&eksmodel.EKSClusterModel{},
							&eksmodel.EKSSubnetModel{},
						).
						SaveEntities(
							t,
							&eksmodel.EKSClusterModel{
								ClusterID: 1,
							},
						).DB,
					imageSelector: &fakeImageSelector{},
				},
				ctx: context.Background(),
				c: cluster.Cluster{
					ID: 1,
				},
				nodePool: eks.NewNodePool{
					InstanceType: "instance-type",
				},
			},
			output: outputType{
				expectedError: errors.New("cannot resolve subnet"),
				expectedNewNodePool: eks.NewNodePool{
					Image:        "image-id",
					InstanceType: "instance-type",
				},
			},
		},
		{
			caseDescription: "success",
			input: inputType{
				p: &nodePoolProcessor{
					db: pkggormtest.NewFakeDatabase(t).
						CreateTablesFromEntities(
							t,
							&eksmodel.EKSClusterModel{},
							&eksmodel.EKSSubnetModel{},
						).
						SaveEntities(
							t,
							&eksmodel.EKSClusterModel{
								ClusterID: 1,
								Subnets: []*eksmodel.EKSSubnetModel{
									{
										SubnetId: aws.String("subnet-id"),
									},
								},
							},
						).DB,
					imageSelector: &fakeImageSelector{},
				},
				ctx: context.Background(),
				c: cluster.Cluster{
					ID: 1,
				},
				nodePool: eks.NewNodePool{
					InstanceType: "instance-type",
				},
			},
			output: outputType{
				expectedError: nil,
				expectedNewNodePool: eks.NewNodePool{
					Image:        "image-id",
					InstanceType: "instance-type",
					SubnetID:     "subnet-id",
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualNewNodePool, actualError := testCase.input.p.ProcessNewNodePool(
				testCase.input.ctx,
				testCase.input.c,
				testCase.input.nodePool,
			)

			if testCase.output.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.output.expectedError.Error())
			}
			require.Equal(t, testCase.output.expectedNewNodePool, actualNewNodePool)
		})
	}
}
