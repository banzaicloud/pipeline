package network

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"fmt"
	"context"
)

// SecurityGroupsClient responsible for security group
type SecurityGroupsClient struct {
	client *network.SecurityGroupsClient
}

// NewSecurityGroupsClient creates a new 'SecurityGroupsClient' instance
func NewSecurityGroupsClient(authorizer autorest.Authorizer, subscriptionId string) *SecurityGroupsClient {
	securityGroupClient := network.NewSecurityGroupsClient(subscriptionId)
	securityGroupClient.Authorizer = authorizer

	return &SecurityGroupsClient{
		client: &securityGroupClient,
	}
}

// CreateOrUpdateSimpleNetworkSecurityGroup creates a new network security group, without rules (rules can be set later)
func (s *SecurityGroupsClient) CreateOrUpdateSimpleNetworkSecurityGroup(rg, location, nsgName string) (*network.SecurityGroup, error) {
	future, err := s.client.CreateOrUpdate(
		context.Background(),
		rg,
		nsgName,
		network.SecurityGroup{
			Location: to.StringPtr(location),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("cannot create nsg: %v", err)
	}

	err = future.WaitForCompletion(context.Background(), s.client.Client)
	if err != nil {
		return nil, fmt.Errorf("cannot get nsg create or update future response: %v", err)
	}

	nsg, err := future.Result(*s.client)
	if err != nil {
		return nil, err
	}
	return &nsg, nil
}

// GetNetworkSecurityGroup returns an existing network security group
func (s *SecurityGroupsClient) GetNetworkSecurityGroup(rg, nsgName string) (network.SecurityGroup, error) {
	return s.client.Get(context.Background(), rg, nsgName, "")
}
