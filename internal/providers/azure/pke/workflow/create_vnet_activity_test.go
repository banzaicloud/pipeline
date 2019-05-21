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

package workflow

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/stretchr/testify/assert"
)

func TestGetCreateOrUpdateVirtualNetworkParams(t *testing.T) {
	t.Run("typical input", func(t *testing.T) {
		input := CreateVnetActivityInput{
			OrganizationID:    1,
			SecretID:          "0123456789abcdefghijklmnopqrstuvwxyz",
			ClusterName:       "test-cluster",
			ResourceGroupName: "test-rg",
			VirtualNetwork: VirtualNetwork{
				CIDRs: []string{
					"1.2.3.4/16",
				},
				Location: "test-location",
				Name:     "test-vnet",
				Subnets: []Subnet{
					{
						CIDR:                   "1.2.3.4/32",
						Name:                   "test-subnet",
						NetworkSecurityGroupID: "/subscription/test-subscription/resourceGroup/test-rg/providers/Microsoft.Network/networkSecurityGroups/test-nsg",
						RouteTableID:           "/subscription/test-subscription/resourceGroup/test-rg/providers/Microsoft.Network/routeTables/test-route-table",
					},
				},
			},
		}
		expected := network.VirtualNetwork{
			Location: to.StringPtr("test-location"),
			Tags: map[string]*string{
				"kubernetesCluster-test-cluster": to.StringPtr("owned"),
			},
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{
						"1.2.3.4/16",
					},
				},
				Subnets: &[]network.Subnet{
					{
						Name: to.StringPtr("test-subnet"),
						SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr("1.2.3.4/32"),
							NetworkSecurityGroup: &network.SecurityGroup{
								ID: to.StringPtr("/subscription/test-subscription/resourceGroup/test-rg/providers/Microsoft.Network/networkSecurityGroups/test-nsg"),
							},
							RouteTable: &network.RouteTable{
								ID: to.StringPtr("/subscription/test-subscription/resourceGroup/test-rg/providers/Microsoft.Network/routeTables/test-route-table"),
							},
						},
					},
				},
			},
		}
		result := input.getCreateOrUpdateVirtualNetworkParams()
		assert.Equal(t, expected, result)
	})
}
