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

func TestNodePoolValudatorValidateNewNodePool(t *testing.T) {
	type inputType struct {
		v        *nodePoolValidator
		ctx      context.Context
		c        cluster.Cluster
		nodePool eks.NewNodePool
	}

	testCases := []struct {
		caseDescription string
		expectedError   error
		input           inputType
	}{
		{
			caseDescription: "cluster not found error -> error",
			expectedError:   errors.New("cluster model is inconsistent"),
			input: inputType{
				v: &nodePoolValidator{
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
		},
		{
			caseDescription: "database error -> error",
			expectedError:   errors.New("failed to get cluster info: test error"),
			input: inputType{
				v: &nodePoolValidator{
					db: pkggormtest.NewFakeDatabase(t).
						CreateTablesFromEntities(
							t,
							&eksmodel.EKSClusterModel{},
							&eksmodel.EKSSubnetModel{},
						).
						SetError(t, errors.New("test error")).DB,
				},
				ctx:      context.Background(),
				c:        cluster.Cluster{},
				nodePool: eks.NewNodePool{},
			},
		},
		{
			caseDescription: "validation error -> error",
			expectedError:   errors.New("invalid node pool creation request"),
			input: inputType{
				v: &nodePoolValidator{
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
				},
				ctx: context.Background(),
				c: cluster.Cluster{
					ID: 1,
				},
				nodePool: eks.NewNodePool{
					SubnetID: "subnet-id",
				},
			},
		},
		{
			caseDescription: "success",
			expectedError:   nil,
			input: inputType{
				v: &nodePoolValidator{
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
				},
				ctx: context.Background(),
				c: cluster.Cluster{
					ID: 1,
				},
				nodePool: eks.NewNodePool{
					InstanceType: "instance-type",
					Size:         1,
					SubnetID:     "subnet-id",
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.caseDescription, func(t *testing.T) {
			actualError := testCase.input.v.ValidateNewNodePool(
				testCase.input.ctx,
				testCase.input.c,
				testCase.input.nodePool,
			)

			if testCase.expectedError == nil {
				require.NoError(t, actualError)
			} else {
				require.EqualError(t, actualError, testCase.expectedError.Error())
			}
		})
	}
}
