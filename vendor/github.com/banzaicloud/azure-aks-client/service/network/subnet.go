package network

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"fmt"
	"context"
)

// SubnetClient responsible for subnet
type SubnetClient struct {
	client *network.SubnetsClient
}

// NewSubnetClient creates a new 'SubnetClient' instance
func NewSubnetClient(authorizer autorest.Authorizer, subscriptionId string) *SubnetClient {
	subnetClient := network.NewSubnetsClient(subscriptionId)
	subnetClient.Authorizer = authorizer

	return &SubnetClient{
		client: &subnetClient,
	}
}

// CreateOrUpdateVirtualNetworkSubnet creates a subnet in an existing vnet
func (s *SubnetClient) CreateOrUpdateVirtualNetworkSubnet(rg, vnetName, subnetName string) (subnet network.Subnet, err error) {

	future, err := s.client.CreateOrUpdate(
		context.Background(),
		rg,
		vnetName,
		subnetName,
		network.Subnet{
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("10.0.0.0/16"),
			},
		})
	if err != nil {
		return subnet, fmt.Errorf("cannot create subnet: %v", err)
	}

	err = future.WaitForCompletion(context.Background(), s.client.Client)
	if err != nil {
		return subnet, fmt.Errorf("cannot get the subnet create or update future response: %v", err)
	}

	return future.Result(*s.client)
}

// GetVirtualNetworkSubnet returns an existing subnet from a virtual network
func (s *SubnetClient) GetVirtualNetworkSubnet(rg, vnetName, subnetName string) (network.Subnet, error) {
	return s.client.Get(context.Background(), rg, vnetName, subnetName, "")
}
