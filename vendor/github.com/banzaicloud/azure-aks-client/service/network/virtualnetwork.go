package network

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"fmt"
	"context"
)

// VirtualNetworksClient responsible for virtual network
type VirtualNetworksClient struct {
	client *network.VirtualNetworksClient
}

// NewVirtualNetworksClient creates a new 'VirtualNetworksClient' instance
func NewVirtualNetworksClient(authorizer autorest.Authorizer, subscriptionId string) *VirtualNetworksClient {
	virtualNetworksClient := network.NewVirtualNetworksClient(subscriptionId)
	virtualNetworksClient.Authorizer = authorizer

	return &VirtualNetworksClient{
		client: &virtualNetworksClient,
	}
}

// CreateOrUpdateVirtualNetwork creates a virtual network
func (v *VirtualNetworksClient) CreateOrUpdateVirtualNetwork(rg, location, vnetName string) (vnet network.VirtualNetwork, err error) {
	future, err := v.client.CreateOrUpdate(
		context.Background(),
		rg,
		vnetName,
		network.VirtualNetwork{
			Location: to.StringPtr(location),
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{"10.0.0.0/8"},
				},
			},
		})

	if err != nil {
		return vnet, fmt.Errorf("cannot create virtual network: %v", err)
	}

	err = future.WaitForCompletion(context.Background(), v.client.Client)
	if err != nil {
		return vnet, fmt.Errorf("cannot get the vnet create or update future response: %v", err)
	}

	return future.Result(*v.client)
}
