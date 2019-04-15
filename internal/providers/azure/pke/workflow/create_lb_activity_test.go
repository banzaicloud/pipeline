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

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/stretchr/testify/assert"
)

func TestFillLoadBalancerParams(t *testing.T) {
	t.Run("", func(t *testing.T) {
		input := CreateLoadBalancerActivityInput{
			BackendAddressPools: []BackendAddressPool{
				{
					Name: "bap-1",
				},
				{
					Name: "bap-2",
				},
			},
			ClusterName: "cluster-1",
			FrontendIPConfigurations: []FrontendIPConfiguration{
				{
					Name: "fic-1",
					PublicIPAddress: PublicIPAddress{
						Location: "location-1",
						Name:     "public-ip-1",
						SKU:      "Standard",
					},
				},
			},
			LoadBalancingRules: []LoadBalancingRule{
				{
					Name:      "lbr-1",
					ProbeName: "probe-1",
				},
				{
					Name: "lbr-2",
				},
			},
			Location:       "location-1",
			Name:           "lb-1",
			OrganizationID: 1,
			Probes: []Probe{
				{
					Name:     "probe-1",
					Port:     1234,
					Protocol: "Tcp",
				},
			},
			ResourceGroupName: "rg-1",
			SKU:               "Standard",
			SecretID:          "0123456789abcdefghijklmnopqrstuvwxyz",
		}
		expected := network.LoadBalancer{
			LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
				BackendAddressPools: &[]network.BackendAddressPool{
					{
						Name: to.StringPtr("bap-1"),
					},
					{
						Name: to.StringPtr("bap-2"),
					},
				},
				FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
					{
						FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
							PrivateIPAllocationMethod: network.Dynamic,
							PublicIPAddress: &network.PublicIPAddress{
								Location: to.StringPtr("location-1"),
								Name:     to.StringPtr("public-ip-1"),
								PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
									PublicIPAddressVersion:   network.IPv4,
									PublicIPAllocationMethod: network.Static,
								},
								Sku: &network.PublicIPAddressSku{
									Name: network.PublicIPAddressSkuNameStandard,
								},
							},
						},
						Name: to.StringPtr("fic-1"),
					},
				},
				LoadBalancingRules: &[]network.LoadBalancingRule{
					{
						LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
							Probe: &network.SubResource{
								ID: to.StringPtr("/subscriptions/subscription-1/resourceGroups/rg-1/providers/Microsoft.Network/loadBalancers/lb-1/probes/probe-1"),
							},
						},
						Name: to.StringPtr("lbr-1"),
					},
					{
						Name: to.StringPtr("lbr-2"),
					},
				},
				Probes: &[]network.Probe{
					{
						ProbePropertiesFormat: &network.ProbePropertiesFormat{
							Port:     to.Int32Ptr(1234),
							Protocol: network.ProbeProtocolTCP,
						},
						Name: to.StringPtr("probe-1"),
					},
				},
			},
			Location: to.StringPtr("location-1"),
			Sku: &network.LoadBalancerSku{
				Name: network.LoadBalancerSkuNameStandard,
			},
			Tags: map[string]*string{
				"kubernetesCluster-cluster-1": to.StringPtr("owned"),
			},
		}

		var result network.LoadBalancer
		fillLoadBalancerParams(&result, input, "subscription-1")
		assert.Equal(t, expected, result)
	})
}
