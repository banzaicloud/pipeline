package network

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest"
	"context"
	"fmt"
)

// InterfacesClient responsible for network interfaces
type InterfacesClient struct {
	client *network.InterfacesClient
}

// NewInterfacesClient creates a new 'InterfacesClient' instance
func NewInterfacesClient(authorizer autorest.Authorizer, subscriptionId string) *InterfacesClient {
	interfaceClient := network.NewInterfacesClient(subscriptionId)
	interfaceClient.Authorizer = authorizer

	return &InterfacesClient{
		client: &interfaceClient,
	}
}

// GetNetworkInterface returns an existing network interface
func (ic *InterfacesClient) GetNetworkInterface(rg, nicName string) (network.Interface, error) {
	return ic.client.Get(context.Background(), rg, nicName, "")
}

// CreateOrUpdateNetworkInterface creates or updates a network interface
func (ic *InterfacesClient) CreateOrUpdateNetworkInterface(rg, nicName string, nicParams network.Interface) (*network.Interface, error) {
	future, err := ic.client.CreateOrUpdate(context.Background(), rg, nicName, nicParams)
	if err != nil {
		return nil, fmt.Errorf("cannot create nic: %v", err)
	}

	err = future.WaitForCompletion(context.Background(), ic.client.Client)
	if err != nil {
		return nil, fmt.Errorf("cannot get nic create or update future response: %v", err)
	}

	i, err := future.Result(*ic.client)
	if err != nil {
		return nil, err
	}

	return &i, nil
}
