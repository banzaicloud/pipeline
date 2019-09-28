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

	"emperror.dev/errors"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
)

const CollectUpdateClusterProvidersActivityName = "pke-azure-collect-update-cluster-providers"

type CollectUpdateClusterProvidersActivity struct {
	azureClientFactory *AzureClientFactory
}

func MakeCollectUpdateClusterProvidersActivity(azureClientFactory *AzureClientFactory) CollectUpdateClusterProvidersActivity {
	return CollectUpdateClusterProvidersActivity{
		azureClientFactory: azureClientFactory,
	}
}

type CollectUpdateClusterProvidersActivityInput struct {
	OrganizationID uint
	SecretID       string

	ClusterName         string
	ResourceGroupName   string
	PublicIPAddressName string
	RouteTableName      string
	VirtualNetworkName  string
}

type CollectUpdateClusterProvidersActivityOutput struct {
	BackendAddressPoolIDProviders   []MapResourceIDByNameProvider
	InboundNATPoolIDProviders       []MapResourceIDByNameProvider
	PublicIPAddressProvider         ConstantIPAddressProvider
	RouteTableIDProvider            MapResourceIDByNameProvider
	SecurityGroupIDProvider         MapResourceIDByNameProvider
	SubnetIDProvider                MapResourceIDByNameProvider
	ApiServerPrivateAddressProvider ConstantIPAddressProvider
}

func (a CollectUpdateClusterProvidersActivity) Execute(ctx context.Context, input CollectUpdateClusterProvidersActivityInput) (output CollectUpdateClusterProvidersActivityOutput, err error) {

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
		return
	}

	lbs, err := cc.GetLoadBalancersClient().List(ctx, input.ResourceGroupName)
	if err = errors.WrapIf(err, "failed to list load balancers in resource group"); err != nil {
		return
	}

	var backendAddressPoolProviders, inboundNATPoolProviders []MapResourceIDByNameProvider

	for lbs.NotDone() {
		for _, lb := range lbs.Values() {
			if !HasOwnedTag(input.ClusterName, to.StringMap(lb.Tags)) && !HasSharedTag(input.ClusterName, to.StringMap(lb.Tags)) {
				continue
			}

			if len(*lb.BackendAddressPools) > 0 {
				resourceIDProvider := make(MapResourceIDByNameProvider)
				for _, bap := range *lb.BackendAddressPools {
					resourceIDProvider[to.String(bap.Name)] = to.String(bap.ID)
				}
				backendAddressPoolProviders = append(backendAddressPoolProviders, resourceIDProvider)
			}

			if len(*lb.InboundNatPools) > 0 {
				resourceIDProvider := make(MapResourceIDByNameProvider)
				for _, inp := range *lb.InboundNatPools {
					resourceIDProvider[to.String(inp.Name)] = to.String(inp.ID)
				}
				inboundNATPoolProviders = append(inboundNATPoolProviders, resourceIDProvider)
			}

			if lb.FrontendIPConfigurations != nil && lb.LoadBalancingRules != nil {
				for _, lbRule := range *lb.LoadBalancingRules {
					if to.String(lbRule.Name) == pke.GetApiServerLBRuleName() {
						for _, fic := range *lb.FrontendIPConfigurations {
							if to.String(fic.ID) == to.String(lbRule.FrontendIPConfiguration.ID) && fic.PrivateIPAddress != nil {
								output.ApiServerPrivateAddressProvider = ConstantIPAddressProvider(to.String(fic.PrivateIPAddress))
								break
							}
						}
						break
					}
				}
			}
		}

		err = lbs.NextWithContext(ctx)
		if err = errors.WrapIf(err, "retrieving load balancers failed"); err != nil {
			return
		}
	}

	if len(backendAddressPoolProviders) > 0 {
		output.BackendAddressPoolIDProviders = backendAddressPoolProviders
	}

	if len(inboundNATPoolProviders) > 0 {
		output.InboundNATPoolIDProviders = inboundNATPoolProviders
	}

	pip, err := cc.GetPublicIPAddressesClient().Get(ctx, input.ResourceGroupName, input.PublicIPAddressName, "")
	if err = errors.WrapIf(err, "failed to get public IP address"); err != nil {
		return
	}
	output.PublicIPAddressProvider = ConstantIPAddressProvider(to.String(pip.IPAddress))

	rt, err := cc.GetRouteTablesClient().Get(ctx, input.ResourceGroupName, input.RouteTableName, "")
	if err = errors.WrapIf(err, "failed to get route table"); err != nil {
		return
	}
	output.RouteTableIDProvider = MapResourceIDByNameProvider(map[string]string{to.String(rt.Name): to.String(rt.ID)})

	{
		var page network.SecurityGroupListResultPage
		page, err = cc.GetSecurityGroupsClient().List(ctx, input.ResourceGroupName)
		if err = errors.WrapIf(err, "failed to list security groups"); err != nil {
			return
		}

		output.SecurityGroupIDProvider = make(MapResourceIDByNameProvider)

		for iter := network.NewSecurityGroupListResultIterator(page); iter.NotDone(); err = iter.NextWithContext(ctx) {
			if err = errors.WrapIf(err, "failed to advance security group list result iterator"); err != nil {
				return
			}
			sg := iter.Value()
			output.SecurityGroupIDProvider[to.String(sg.Name)] = to.String(sg.ID)
		}
	}

	{
		var page network.SubnetListResultPage
		page, err = cc.GetSubnetsClient().List(ctx, input.ResourceGroupName, input.VirtualNetworkName)
		if err = errors.WrapIf(err, "failed to list subnets"); err != nil {
			return
		}

		output.SubnetIDProvider = make(MapResourceIDByNameProvider)

		for iter := network.NewSubnetListResultIterator(page); iter.NotDone(); err = iter.NextWithContext(ctx) {
			if err = errors.WrapIf(err, "failed to advance subnet list result iterator"); err != nil {
				return
			}
			subnet := iter.Value()
			output.SubnetIDProvider[to.String(subnet.Name)] = to.String(subnet.ID)
		}
	}

	return
}
