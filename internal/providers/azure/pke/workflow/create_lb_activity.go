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

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/to"
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
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	LoadBalancer      LoadBalancer
}

type LoadBalancer struct {
	Name                     string
	Location                 string
	SKU                      string
	BackendAddressPools      []BackendAddressPool
	FrontendIPConfigurations []FrontendIPConfiguration
	InboundNATPools          []InboundNATPool
	LoadBalancingRules       []LoadBalancingRule
	OutboundRules            []OutboundRule
	Probes                   []Probe
}

type BackendAddressPool struct {
	Name string
}

type FrontendIPConfiguration struct {
	Name              string
	PublicIPAddressID string
	SubnetID          string
	Zones             []string
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

type OutboundRule struct {
	Name               string
	BackendAddressPool *BackendAddressPool
	FrontendIPConfigs  []*FrontendIPConfiguration
}

type Probe struct {
	Name     string
	Port     int32
	Protocol string
}

type CreateLoadBalancerActivityOutput struct {
	BackendAddressPoolIDs   map[string]string
	InboundNATPoolIDs       map[string]string
	ApiServerPrivateAddress string
}

// Execute performs the activity
func (a CreateLoadBalancerActivity) Execute(ctx context.Context, input CreateLoadBalancerActivityInput) (output CreateLoadBalancerActivityOutput, err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"loadBalancer", input.LoadBalancer.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"loadBalancer", input.LoadBalancer.Name,
	}

	logger.Info("create load balancer")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetLoadBalancersClient()

	params := input.getCreateOrUpdateLoadBalancerParams(cc.GetSubscriptionID())

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.LoadBalancer.Name, params)
	if err = errors.WrapIfWithDetails(err, "sending request to create or update load balancer failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of create or update load balancer operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = errors.WrapIfWithDetails(err, "waiting for the completion of create or update load balancer operation failed", keyvals...); err != nil {
		return
	}

	lb, err := future.Result(client.LoadBalancersClient)
	if err = errors.WrapIfWithDetails(err, "getting load balancer create or update result failed", keyvals...); err != nil {
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
	if lb.FrontendIPConfigurations != nil && lb.LoadBalancingRules != nil {
		for _, lbRule := range *lb.LoadBalancingRules {
			if to.String(lbRule.Name) == pke.GetApiServerLBRuleName() {
				for _, fic := range *lb.FrontendIPConfigurations {
					if to.String(fic.ID) == to.String(lbRule.FrontendIPConfiguration.ID) && fic.PrivateIPAddress != nil {
						output.ApiServerPrivateAddress = to.String(fic.PrivateIPAddress)
						break
					}
				}
				break
			}
		}
	}

	return
}

func (input CreateLoadBalancerActivityInput) getCreateOrUpdateLoadBalancerParams(subscriptionID string) network.LoadBalancer {
	backendAddressPools := make([]network.BackendAddressPool, 0, len(input.LoadBalancer.BackendAddressPools))
	for _, bap := range input.LoadBalancer.BackendAddressPools {
		if bap.Name != "" {
			backendAddressPools = append(backendAddressPools, network.BackendAddressPool{
				Name: to.StringPtr(bap.Name),
			})
		}
	}

	frontendIPConfigurations := make([]network.FrontendIPConfiguration, len(input.LoadBalancer.FrontendIPConfigurations))
	for i, fic := range input.LoadBalancer.FrontendIPConfigurations {
		var pip *network.PublicIPAddress
		var subnet *network.Subnet

		if fic.PublicIPAddressID != "" {
			pip = &network.PublicIPAddress{
				ID: to.StringPtr(fic.PublicIPAddressID),
			}
		}

		if fic.SubnetID != "" {
			subnet = &network.Subnet{
				ID: to.StringPtr(fic.SubnetID),
			}
		}

		frontendIPConfigurations[i] = network.FrontendIPConfiguration{
			Name: to.StringPtr(fic.Name),
			FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
				PrivateIPAllocationMethod: network.Dynamic,
				PublicIPAddress:           pip,
				Subnet:                    subnet,
			},
			Zones: to.StringSlicePtr(fic.Zones),
		}
	}

	inboundNATPools := make([]network.InboundNatPool, len(input.LoadBalancer.InboundNATPools))
	for i, inp := range input.LoadBalancer.InboundNATPools {
		var ficRef *network.SubResource
		if inp.FrontendIPConfig != nil {
			ficRef = &network.SubResource{
				ID: to.StringPtr(getLoadBalancerFrontendIPConfigurationID(subscriptionID, input.ResourceGroupName, input.LoadBalancer.Name, inp.FrontendIPConfig.Name)),
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

	loadBalancingRules := make([]network.LoadBalancingRule, len(input.LoadBalancer.LoadBalancingRules))
	for i, lbr := range input.LoadBalancer.LoadBalancingRules {
		var bapRef *network.SubResource
		if lbr.BackendAddressPool != nil {
			bapRef = &network.SubResource{
				ID: to.StringPtr(getLoadBalancerBackendAddressPoolID(subscriptionID, input.ResourceGroupName, input.LoadBalancer.Name, lbr.BackendAddressPool.Name)),
			}
		}
		var ficRef *network.SubResource
		if lbr.FrontendIPConfig != nil {
			ficRef = &network.SubResource{
				ID: to.StringPtr(getLoadBalancerFrontendIPConfigurationID(subscriptionID, input.ResourceGroupName, input.LoadBalancer.Name, lbr.FrontendIPConfig.Name)),
			}
		}
		var probeRef *network.SubResource
		if lbr.Probe != nil {
			probeRef = &network.SubResource{
				ID: to.StringPtr(getLoadBalancerProbeID(subscriptionID, input.ResourceGroupName, input.LoadBalancer.Name, lbr.Probe.Name)),
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

	outboundRules := make([]network.OutboundRule, len(input.LoadBalancer.OutboundRules))
	for i, onr := range input.LoadBalancer.OutboundRules {
		var bapRef *network.SubResource
		if onr.BackendAddressPool != nil {
			bapRef = &network.SubResource{
				ID: to.StringPtr(getLoadBalancerBackendAddressPoolID(subscriptionID, input.ResourceGroupName, input.LoadBalancer.Name, onr.BackendAddressPool.Name)),
			}
		}
		var ficRefs []network.SubResource
		if l := len(onr.FrontendIPConfigs); l != 0 {
			ficRefs = make([]network.SubResource, l)
			for i, fic := range onr.FrontendIPConfigs {
				ficRefs[i] = network.SubResource{
					ID: to.StringPtr(getLoadBalancerFrontendIPConfigurationID(subscriptionID, input.ResourceGroupName, input.LoadBalancer.Name, fic.Name)),
				}
			}
		}
		outboundRules[i] = network.OutboundRule{
			Name: to.StringPtr(onr.Name),
			OutboundRulePropertiesFormat: &network.OutboundRulePropertiesFormat{
				BackendAddressPool:       bapRef,
				FrontendIPConfigurations: &ficRefs,
			},
		}
	}

	probes := make([]network.Probe, len(input.LoadBalancer.Probes))
	for i, p := range input.LoadBalancer.Probes {
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
			OutboundRules:            &outboundRules,
			Probes:                   &probes,
		},
		Location: to.StringPtr(input.LoadBalancer.Location),
		Sku: &network.LoadBalancerSku{
			Name: network.LoadBalancerSkuName(input.LoadBalancer.SKU),
		},
		Tags: getClusterTags(input.ClusterName),
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
