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
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/goph/logur/adapters/zapadapter"
	"go.uber.org/cadence/activity"
)

// CreateVnetActivityName is the default registration name of the activity
const CreateVnetActivityName = "pke-azure-create-vnet"

// CreateVnetActivity represents an activity for creating an Azure virtual network
type CreateVnetActivity struct {
	azureClientFactory *AzureClientFactory
}

// MakeCreateVnetActivity returns a new CreateVnetActivity
func MakeCreateVnetActivity(azureClientFactory *AzureClientFactory) CreateVnetActivity {
	return CreateVnetActivity{
		azureClientFactory: azureClientFactory,
	}
}

// CreateVnetActivityInput represents the input needed for executing a CreateVnetActivity
type CreateVnetActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	VirtualNetwork    VirtualNetwork
}

type VirtualNetwork struct {
	Name     string
	CIDRs    []string
	Location string
	Subnets  []Subnet
}

type Subnet struct {
	Name                   string
	CIDR                   string
	NetworkSecurityGroupID string
	RouteTableID           string
}

type CreateVnetActivityOutput struct {
	VirtualNetworkID string
	SubnetIDs        map[string]string
}

// Execute performs the activity
func (a CreateVnetActivity) Execute(ctx context.Context, input CreateVnetActivityInput) (output CreateVnetActivityOutput, err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"networkName", input.VirtualNetwork.Name,
		"networkLocation", input.VirtualNetwork.Location,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"networkName", input.VirtualNetwork.Name,
	}

	logger.Info("create or update virtual network")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetVirtualNetworksClient()

	vnet, err := client.Get(ctx, input.ResourceGroupName, input.VirtualNetwork.Name, "")
	if vnet.StatusCode == http.StatusNotFound {
		vnet = input.getCreateOrUpdateVirtualNetworkParams()
	} else if err = emperror.Wrap(err, "failed to get virtual network"); err != nil {
		return
	} else {
		input.extendVirtualNetwork(&vnet, zapadapter.New(logger.Desugar()))
	}

	logger.Debug("sending request to create or update virtual network")

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.VirtualNetwork.Name, vnet)
	if err = emperror.WrapWith(err, "sending request to create or update virtual network failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of create or update virtual network operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of create or update virtual network operation failed", keyvals...); err != nil {
		return
	}

	vnet, err = future.Result(client.VirtualNetworksClient)
	if err = emperror.WrapWith(err, "getting virtual network create or update result failed", keyvals...); err != nil {
		return
	}

	output.VirtualNetworkID = to.String(vnet.ID)
	output.SubnetIDs = make(map[string]string)
	if vnet.Subnets != nil {
		for _, s := range *vnet.Subnets {
			if s.Name != nil && s.ID != nil {
				output.SubnetIDs[*s.Name] = *s.ID
			}
		}
	}

	return
}

func (input CreateVnetActivityInput) extendVirtualNetwork(vnet *network.VirtualNetwork, logger logur.Logger) {
	if vnet == nil {
		return
	}

	if vnet.VirtualNetworkPropertiesFormat == nil {
		vnet.VirtualNetworkPropertiesFormat = new(network.VirtualNetworkPropertiesFormat)
		logger.Debug("vnet.VirtualNetworkPropertiesFormat was nil, created new")
	}

	if loc := to.String(vnet.Location); loc != input.VirtualNetwork.Location {
		logger.Warn("virtual network location mismatch", map[string]interface{}{
			"existing": loc,
			"incoming": input.VirtualNetwork.Location,
		})
	}

	if len(input.VirtualNetwork.CIDRs) > 0 {
		if vnet.AddressSpace == nil {
			vnet.AddressSpace = new(network.AddressSpace)
			logger.Debug("vnet.AddressSpace was nil, created new")
		}
		lookup := make(map[string]bool)
		aps := to.StringSlice(vnet.AddressSpace.AddressPrefixes)
		for _, ap := range aps {
			lookup[ap] = true
		}
		for _, cidr := range input.VirtualNetwork.CIDRs {
			if !lookup[cidr] {
				lookup[cidr] = true
				aps = append(aps, cidr)
			}
		}
		vnet.AddressSpace.AddressPrefixes = to.StringSlicePtr(aps)
	}

	if len(input.VirtualNetwork.Subnets) > 0 {
		lookup := make(map[string]bool)
		subnets := toSubnetSlice(vnet.Subnets)
		for _, s := range subnets {
			lookup[to.String(s.Name)] = true
		}
		for _, subnet := range input.VirtualNetwork.Subnets {
			if !lookup[subnet.Name] {
				lookup[subnet.Name] = true
				subnets = append(subnets, subnetToNetworkSubnet(subnet))
			}
		}
		vnet.Subnets = &subnets
	}

	tag := getSharedTag(input.ClusterName)
	if _, exists := vnet.Tags[tag.Key]; !exists {
		if vnet.Tags == nil {
			vnet.Tags = make(map[string]*string)
		}
		vnet.Tags[tag.Key] = to.StringPtr(tag.Value)
	}
}

func (input CreateVnetActivityInput) getCreateOrUpdateVirtualNetworkParams() network.VirtualNetwork {
	subnets := make([]network.Subnet, len(input.VirtualNetwork.Subnets))
	for i, s := range input.VirtualNetwork.Subnets {
		subnets[i] = subnetToNetworkSubnet(s)
	}

	return network.VirtualNetwork{
		Location: to.StringPtr(input.VirtualNetwork.Location),
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: to.StringSlicePtr(input.VirtualNetwork.CIDRs),
			},
			Subnets: &subnets,
		},
		Tags: *to.StringMapPtr(getOwnedTag(input.ClusterName).Map()),
	}
}

func toSubnetSlice(ptr *[]network.Subnet) []network.Subnet {
	if ptr == nil {
		return nil
	}
	return *ptr
}

func subnetToNetworkSubnet(subnet Subnet) network.Subnet {
	var nsg *network.SecurityGroup
	if subnet.NetworkSecurityGroupID != "" {
		nsg = &network.SecurityGroup{
			ID: to.StringPtr(subnet.NetworkSecurityGroupID),
		}
	}

	return network.Subnet{
		Name: to.StringPtr(subnet.Name),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix:        to.StringPtr(subnet.CIDR),
			NetworkSecurityGroup: nsg,
			RouteTable: &network.RouteTable{
				ID: to.StringPtr(subnet.RouteTableID),
			},
		},
	}
}
