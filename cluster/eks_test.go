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

package cluster

import (
	"reflect"
	"testing"

	eksworkflow "github.com/banzaicloud/pipeline/internal/providers/amazon/eks/workflow"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
)

func TestCreateSubnetMappingFromRequest(t *testing.T) {
	eksRequest := &eks.CreateClusterEKS{
		NodePools: map[string]*eks.NodePool{
			"pool1": {
				Subnet: &eks.Subnet{SubnetId: "subnet1"},
			},
		},
		Subnets: []*eks.Subnet{
			{SubnetId: "subnet0"},
			{SubnetId: "subnet1"},
		},
	}

	expected := map[string][]*eks.Subnet{
		"default": {
			{SubnetId: "subnet0"},
			{SubnetId: "subnet1"},
		},
		"pool1": {
			{SubnetId: "subnet1"},
		},
	}
	subnetMappings := createSubnetMappingFromRequest(eksRequest)

	if !reflect.DeepEqual(subnetMappings, expected) {
		t.Errorf("Expected: %v, got: %v", expected, subnetMappings)
	}
}

func TestGetNodePoolsForSubnet(t *testing.T) {
	subnetMapping := map[string][]*eks.Subnet{
		"default": {
			&eks.Subnet{
				Cidr: "192.168.64.0/20",
			},
			&eks.Subnet{
				Cidr: "192.168.80.0/20",
			},
			&eks.Subnet{
				SubnetId: "subnet0",
			},
		},
		"pool1": {
			&eks.Subnet{
				Cidr: "192.168.64.0/20",
			},
		},
		"pool2": {
			&eks.Subnet{
				Cidr: "192.168.80.0/20",
			},
		},
		"pool3": {
			&eks.Subnet{
				SubnetId: "subnet0",
			},
			&eks.Subnet{
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
