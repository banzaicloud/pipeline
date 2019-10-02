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

	ResourceGroupName   string
	LoadBalancerName    string
	PublicIPAddressName string
	RouteTableName      string
	VirtualNetworkName  string
}

type CollectUpdateClusterProvidersActivityOutput struct {
	BackendAddressPoolIDProvider MapResourceIDByNameProvider
	InboundNATPoolIDProvider     MapResourceIDByNameProvider
	PublicIPAddressProvider      ConstantIPAddressProvider
	RouteTableIDProvider         MapResourceIDByNameProvider
	SecurityGroupIDProvider      MapResourceIDByNameProvider
	SubnetIDProvider             MapResourceIDByNameProvider
}

func (a CollectUpdateClusterProvidersActivity) Execute(ctx context.Context, input CollectUpdateClusterProvidersActivityInput) (output CollectUpdateClusterProvidersActivityOutput, err error) {

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
		return
	}

	lb, err := cc.GetLoadBalancersClient().Get(ctx, input.ResourceGroupName, input.LoadBalancerName, "")
	if err = errors.WrapIf(err, "failed to get load balancer"); err != nil {
		return
	}

	if lb.BackendAddressPools != nil {
		output.BackendAddressPoolIDProvider = make(MapResourceIDByNameProvider, len(*lb.BackendAddressPools))
		for _, bap := range *lb.BackendAddressPools {
			output.BackendAddressPoolIDProvider[to.String(bap.Name)] = to.String(bap.ID)
		}
	}

	if lb.InboundNatPools != nil {
		output.InboundNATPoolIDProvider = make(MapResourceIDByNameProvider, len(*lb.InboundNatPools))
		for _, inp := range *lb.InboundNatPools {
			output.InboundNATPoolIDProvider[to.String(inp.Name)] = to.String(inp.ID)
		}
	}

	pip, err := cc.GetPublicIPAddressesClient().Get(ctx, input.ResourceGroupName, input.PublicIPAddressName, "")
	if err = errors.WrapIf(err, "failed to get public IP address"); err != nil {
		return
	}
	output.PublicIPAddressProvider = ConstantIPAddressProvider(to.String(pip.ID))

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
