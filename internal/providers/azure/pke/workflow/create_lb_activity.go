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
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
)

// CreateLoadBalancerActivityName is the default registration name of the activity
const CreateLoadBalancerActivityName = "pke-azure-create-load-balancer"

// CreateLoadBalancerActivity represents an activity for creating an Azure load balancer
type CreateLoadBalancerActivity struct {
	azureClientFactory *AzureClientFactory
}

// MakeCreateLoadBalancerActivity returns a new CreateLoadBalancerActivity
func MakeCreateLoadBalancerActivity(azureClientFactory *AzureClientFactory) CreateLoadBalancerActivity {
	return CreateLoadBalancerActivity{
		azureClientFactory: azureClientFactory,
	}
}

// CreateLoadBalancerActivityInput represents the input needed for executing a CreateLoadBalancerActivity
type CreateLoadBalancerActivityInput struct {
	Name                     string
	Location                 string
	SKU                      string
	BackendAddressPools      []BackendAddressPool
	FrontendIPConfigurations []FrontendIPConfiguration
	InboundNATPools          []InboundNATPool
	LoadBalancingRules       []LoadBalancingRule
	Probes                   []Probe

	ResourceGroupName string
	OrganizationID    uint
	ClusterName       string
	SecretID          string
}

type BackendAddressPool struct {
	Name string
}

type FrontendIPConfiguration struct {
	Name            string
	PublicIPAddress PublicIPAddress
	Zones           []string
}

type InboundNATPool struct {
	Name                   string
	BackendPort            int32
	FrontendIPConfig       *FrontendIPConfiguration
	FrontendPortRangeEnd   int32
	FrontendPortRangeStart int32
	Protocol               string
}

type LoadBalancingRule struct {
	Name                string
	BackendAddressPool  *BackendAddressPool
	BackendPort         int32
	DisableOutboundSNAT bool
	FrontendIPConfig    *FrontendIPConfiguration
	FrontendPort        int32
	Probe               *Probe
	Protocol            string
}

type Probe struct {
	Name     string
	Port     int32
	Protocol string
}

type PublicIPAddress struct {
	Location string
	Name     string
	SKU      string
}

type CreateLoadBalancerActivityOutput struct {
	BackendAddressPoolIDs map[string]string
	InboundNATPoolIDs     map[string]string
}

// Execute performs the activity
func (a CreateLoadBalancerActivity) Execute(ctx context.Context, input CreateLoadBalancerActivityInput) (output CreateLoadBalancerActivityOutput, err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"loadBalancer", input.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"loadBalancer", input.Name,
	}

	logger.Info("create load balancer")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetLoadBalancersClient()

	params := input.getCreateOrUpdateLoadBalancerParams(client.SubscriptionID)

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.Name, params)
	if err = emperror.WrapWith(err, "sending request to create or update load balancer failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of create or update load balancer operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of create or update load balancer operation failed", keyvals...); err != nil {
		return
	}

	lb, err := future.Result(client.LoadBalancersClient)
	if err = emperror.WrapWith(err, "getting load balancer create or update result failed", keyvals...); err != nil {
		return
	}

	output.BackendAddressPoolIDs = make(map[string]string)
	if lb.BackendAddressPools != nil {
		for _, bap := range *lb.BackendAddressPools {
			if bap.Name != nil && bap.ID != nil {
				output.BackendAddressPoolIDs[*bap.Name] = *bap.ID
			}
		}
	}
	output.InboundNATPoolIDs = make(map[string]string)
	if lb.InboundNatPools != nil {
		for _, inp := range *lb.InboundNatPools {
			if inp.Name != nil && inp.ID != nil {
				output.InboundNATPoolIDs[*inp.Name] = *inp.ID
			}
		}
	}
	return
}

func (input CreateLoadBalancerActivityInput) getCreateOrUpdateLoadBalancerParams(subscriptionID string) network.LoadBalancer {
	backendAddressPools := make([]network.BackendAddressPool, len(input.BackendAddressPools))
	for i, bap := range input.BackendAddressPools {
		backendAddressPools[i] = network.BackendAddressPool{
			Name: to.StringPtr(bap.Name),
		}
	}

	frontendIPConfigurations := make([]network.FrontendIPConfiguration, len(input.FrontendIPConfigurations))
	for i, fic := range input.FrontendIPConfigurations {
		frontendIPConfigurations[i] = network.FrontendIPConfiguration{
			Name: to.StringPtr(fic.Name),
			FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
				PrivateIPAllocationMethod: network.Dynamic,
				PublicIPAddress: &network.PublicIPAddress{
					Name:     to.StringPtr(fic.PublicIPAddress.Name),
					Location: to.StringPtr(fic.PublicIPAddress.Location),
					PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
						PublicIPAddressVersion:   network.IPv4,
						PublicIPAllocationMethod: network.Static,
					},
					Sku: &network.PublicIPAddressSku{
						Name: network.PublicIPAddressSkuName(fic.PublicIPAddress.SKU),
					},
				},
			},
			Zones: to.StringSlicePtr(fic.Zones),
		}
	}

	inboundNATPools := make([]network.InboundNatPool, len(input.InboundNATPools))
	for i, inp := range input.InboundNATPools {
		var ficRef *network.SubResource
		if inp.FrontendIPConfig != nil {
			ficRef = &network.SubResource{
				ID: to.StringPtr(getLoadBalancerFrontendIPConfigurationID(subscriptionID, input.ResourceGroupName, input.Name, inp.FrontendIPConfig.Name)),
			}
		}
		inboundNATPools[i] = network.InboundNatPool{
			Name: to.StringPtr(inp.Name),
			InboundNatPoolPropertiesFormat: &network.InboundNatPoolPropertiesFormat{
				BackendPort:             to.Int32Ptr(inp.BackendPort),
				FrontendIPConfiguration: ficRef,
				FrontendPortRangeEnd:    to.Int32Ptr(inp.FrontendPortRangeEnd),
				FrontendPortRangeStart:  to.Int32Ptr(inp.FrontendPortRangeStart),
				Protocol:                network.TransportProtocol(inp.Protocol),
			},
		}
	}

	loadBalancingRules := make([]network.LoadBalancingRule, len(input.LoadBalancingRules))
	for i, lbr := range input.LoadBalancingRules {
		var bapRef *network.SubResource
		if lbr.BackendAddressPool != nil {
			bapRef = &network.SubResource{
				ID: to.StringPtr(getLoadBalancerBackendAddressPoolID(subscriptionID, input.ResourceGroupName, input.Name, lbr.BackendAddressPool.Name)),
			}
		}
		var ficRef *network.SubResource
		if lbr.FrontendIPConfig != nil {
			ficRef = &network.SubResource{
				ID: to.StringPtr(getLoadBalancerFrontendIPConfigurationID(subscriptionID, input.ResourceGroupName, input.Name, lbr.FrontendIPConfig.Name)),
			}
		}
		var probeRef *network.SubResource
		if lbr.Probe != nil {
			probeRef = &network.SubResource{
				ID: to.StringPtr(getLoadBalancerProbeID(subscriptionID, input.ResourceGroupName, input.Name, lbr.Probe.Name)),
			}
		}
		loadBalancingRules[i] = network.LoadBalancingRule{
			Name: to.StringPtr(lbr.Name),
			LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
				BackendAddressPool:      bapRef,
				BackendPort:             to.Int32Ptr(lbr.BackendPort),
				DisableOutboundSnat:     to.BoolPtr(lbr.DisableOutboundSNAT),
				FrontendIPConfiguration: ficRef,
				FrontendPort:            to.Int32Ptr(lbr.FrontendPort),
				Probe:                   probeRef,
				Protocol:                network.TransportProtocol(lbr.Protocol),
			},
		}
	}

	probes := make([]network.Probe, len(input.Probes))
	for i, p := range input.Probes {
		probes[i] = network.Probe{
			Name: to.StringPtr(p.Name),
			ProbePropertiesFormat: &network.ProbePropertiesFormat{
				Port:     to.Int32Ptr(p.Port),
				Protocol: network.ProbeProtocol(p.Protocol),
			},
		}
	}

	return network.LoadBalancer{
		LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
			BackendAddressPools:      &backendAddressPools,
			FrontendIPConfigurations: &frontendIPConfigurations,
			InboundNatPools:          &inboundNATPools,
			LoadBalancingRules:       &loadBalancingRules,
			Probes:                   &probes,
		},
		Location: to.StringPtr(input.Location),
		Sku: &network.LoadBalancerSku{
			Name: network.LoadBalancerSkuName(input.SKU),
		},
		Tags: *to.StringMapPtr(tagsFrom(getOwnedTag(input.ClusterName))),
	}
}

func getLoadBalancerBackendAddressPoolID(subscriptionID, resourceGroupName, loadBalancerName, poolName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/%s", subscriptionID, resourceGroupName, loadBalancerName, poolName)
}

func getLoadBalancerFrontendIPConfigurationID(subscriptionID, resourceGroupName, loadBalancerName, configName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", subscriptionID, resourceGroupName, loadBalancerName, configName)
}

func getLoadBalancerProbeID(subscriptionID, resourceGroupName, loadBalancerName, probeName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/probes/%s", subscriptionID, resourceGroupName, loadBalancerName, probeName)
}
