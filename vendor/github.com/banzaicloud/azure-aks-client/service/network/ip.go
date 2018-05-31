package network

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"fmt"
	"context"
)

// IPClient responsible for IP
type IPClient struct {
	client *network.PublicIPAddressesClient
}

// NewIPClient creates a new 'IPClient' instance
func NewIPClient(authorizer autorest.Authorizer, subscriptionId string) *IPClient {
	ipClient := network.NewPublicIPAddressesClient(subscriptionId)
	ipClient.Authorizer = authorizer

	return &IPClient{
		client: &ipClient,
	}
}

// CreatePublicIP creates a new public IP
func (ipc *IPClient) CreatePublicIP(rg, location, ipName string) (*network.PublicIPAddress, error) {
	future, err := ipc.client.CreateOrUpdate(
		context.Background(),
		rg,
		ipName,
		network.PublicIPAddress{
			Name:     to.StringPtr(ipName),
			Location: to.StringPtr(location),
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   network.IPv4,
				PublicIPAllocationMethod: network.Static,
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("cannot create public ip address: %v", err)
	}

	err = future.WaitForCompletion(context.Background(), ipc.client.Client)
	if err != nil {
		return nil, fmt.Errorf("cannot get public ip address create or update future response: %v", err)
	}

	ip, err := future.Result(*ipc.client)
	if err != nil {
		return nil, err
	}
	return &ip, err
}

// GetPublicIP returns an existing public IP
func (ipc *IPClient) GetPublicIP(rg, ipName string) (network.PublicIPAddress, error) {
	return ipc.client.Get(context.Background(), rg, ipName, "")
}
