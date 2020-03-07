// Copyright © 2019 Banzai Cloud
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
	"reflect"
	"testing"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/ekscluster"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
)

func TestGetNodePoolsForSubnet(t *testing.T) {
	subnetMapping := map[string][]*ekscluster.Subnet{
		"default": {
			&ekscluster.Subnet{
				Cidr: "192.168.64.0/20",
			},
			&ekscluster.Subnet{
				Cidr: "192.168.80.0/20",
			},
			&ekscluster.Subnet{
				SubnetId: "subnet0",
			},
		},
		"pool1": {
			&ekscluster.Subnet{
				Cidr: "192.168.64.0/20",
			},
		},
		"pool2": {
			&ekscluster.Subnet{
				Cidr: "192.168.80.0/20",
			},
		},
		"pool3": {
			&ekscluster.Subnet{
				SubnetId: "subnet0",
			},
			&ekscluster.Subnet{
				Cidr: "192.168.80.0/20",
			},
		},
	}

	testCases := []struct {
		name              string
		subnet            eksworkflow.Subnet
		expectedNodePools []string
	}{
		{
			name: "get node pools by subnet cidr",
			subnet: eksworkflow.Subnet{
				Cidr: "192.168.64.0/20",
			},
			expectedNodePools: []string{"default", "pool1"},
		},
		{
			name: "get node pools by subnet subnet id",
			subnet: eksworkflow.Subnet{
				SubnetID: "subnet0",
			},
			expectedNodePools: []string{"default", "pool3"},
		},
		{
			name: "no matching subnet mapping",
			subnet: eksworkflow.Subnet{
				SubnetID: "subnetx",
			},
			expectedNodePools: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nodePools := getNodePoolsForSubnet(subnetMapping, tc.subnet)

			if tc.expectedNodePools == nil && nodePools != nil {
				t.Errorf("Expected: %v, got: %v", tc.expectedNodePools, nodePools)
			}

			expected := make(map[string]bool)
			actual := make(map[string]bool)

			for _, np := range tc.expectedNodePools {
				expected[np] = true
			}
			for _, np := range nodePools {
				actual[np] = true
			}

			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("Expected: %v, got: %v", expected, actual)
			}
		})
	}
}
