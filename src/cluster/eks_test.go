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
