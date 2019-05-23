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

func TestGetCreateOrUpdateLoadBalancerParams(t *testing.T) {
	t.Run("typical input", func(t *testing.T) {
		input := CreateLoadBalancerActivityInput{
			OrganizationID:    1,
			SecretID:          "0123456789abcdefghijklmnopqrstuvwxyz",
			ClusterName:       "test-cluster",
			ResourceGroupName: "test-rg",
			LoadBalancer: LoadBalancer{
				BackendAddressPools: []BackendAddressPool{
					{
						Name: "test-bap",
					},
				},
				FrontendIPConfigurations: []FrontendIPConfiguration{
					{
						Name:              "test-fic",
						PublicIPAddressID: "test-public-ip",
						Zones:             []string{"1", "3"},
					},
				},
				InboundNATPools: []InboundNATPool{
					{
						BackendPort: int32(42),
						FrontendIPConfig: &FrontendIPConfiguration{
							Name:              "test-fic",
							PublicIPAddressID: "test-public-ip",
							Zones:             []string{"1", "3"},
						},
						FrontendPortRangeEnd:   int32(42424),
						FrontendPortRangeStart: int32(42422),
						Name:                   "test-inp",
						Protocol:               "Tcp",
					},
				},
				LoadBalancingRules: []LoadBalancingRule{
					{
						BackendAddressPool: &BackendAddressPool{
							Name: "test-bap",
						},
						BackendPort:         int32(4242),
						DisableOutboundSNAT: false,
						FrontendIPConfig: &FrontendIPConfiguration{
							Name:              "test-fic",
							PublicIPAddressID: "test-public-ip",
							Zones:             []string{"1", "3"},
						},
						FrontendPort: int32(24242),
						Name:         "test-lbr",
						Probe: &Probe{
							Name:     "test-probe",
							Port:     1234,
							Protocol: "Tcp",
						},
						Protocol: "Tcp",
					},
				},
				Location: "test-location",
				Name:     "test-lb",
				Probes: []Probe{
					{
						Name:     "test-probe",
						Port:     1234,
						Protocol: "Tcp",
					},
				},
				SKU:           "Standard",
				OutboundRules: []OutboundRule{},
			},
		}
		expected := network.LoadBalancer{
			LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
				BackendAddressPools: &[]network.BackendAddressPool{
					{
						Name: to.StringPtr("test-bap"),
					},
				},
				FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
					{
						FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
							PrivateIPAllocationMethod: network.Dynamic,
							PublicIPAddress: &network.PublicIPAddress{
								ID: to.StringPtr("test-public-ip"),
							},
						},
						Name:  to.StringPtr("test-fic"),
						Zones: to.StringSlicePtr([]string{"1", "3"}),
					},
				},
				InboundNatPools: &[]network.InboundNatPool{
					{
						InboundNatPoolPropertiesFormat: &network.InboundNatPoolPropertiesFormat{
							BackendPort: to.Int32Ptr(int32(42)),
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/frontendIPConfigurations/test-fic"),
							},
							FrontendPortRangeEnd:   to.Int32Ptr(int32(42424)),
							FrontendPortRangeStart: to.Int32Ptr(int32(42422)),
							Protocol:               network.TransportProtocolTCP,
						},
						Name: to.StringPtr("test-inp"),
					},
				},
				LoadBalancingRules: &[]network.LoadBalancingRule{
					{
						LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
							BackendAddressPool: &network.SubResource{
								ID: to.StringPtr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/backendAddressPools/test-bap"),
							},
							BackendPort:         to.Int32Ptr(int32(4242)),
							DisableOutboundSnat: to.BoolPtr(false),
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/frontendIPConfigurations/test-fic"),
							},
							FrontendPort: to.Int32Ptr(int32(24242)),
							Probe: &network.SubResource{
								ID: to.StringPtr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/probes/test-probe"),
							},
							Protocol: network.TransportProtocolTCP,
						},
						Name: to.StringPtr("test-lbr"),
					},
				},
				Probes: &[]network.Probe{
					{
						ProbePropertiesFormat: &network.ProbePropertiesFormat{
							Port:     to.Int32Ptr(1234),
							Protocol: network.ProbeProtocolTCP,
						},
						Name: to.StringPtr("test-probe"),
					},
				},
				OutboundRules: &[]network.OutboundRule{},
			},
			Location: to.StringPtr("test-location"),
			Sku: &network.LoadBalancerSku{
				Name: network.LoadBalancerSkuNameStandard,
			},
			Tags: map[string]*string{
				"kubernetesCluster-test-cluster": to.StringPtr("owned"),
			},
		}

		result := input.getCreateOrUpdateLoadBalancerParams("test-subscription")
		assert.Equal(t, expected, result)
	})
}
