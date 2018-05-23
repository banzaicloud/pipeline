package resources

import (
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"context"
	"fmt"
	"github.com/Azure/go-autorest/autorest"
	"github.com/banzaicloud/banzai-types/constants"
)

// ResourceGroupClient responsible for resource group
type ResourceGroupClient struct {
	client *resources.GroupsClient
}

// NewResourceGroupClient creates a new 'ResourceGroupClient' instance
func NewResourceGroupClient(authorizer autorest.Authorizer, subscriptionId string) *ResourceGroupClient {

	groupsClient := resources.NewGroupsClient(subscriptionId)
	groupsClient.Authorizer = authorizer

	return &ResourceGroupClient{
		client: &groupsClient,
	}
}

// ListGroups gets all the resource groups for a subscription
func (r *ResourceGroupClient) ListGroups() ([]resources.Group, error) {
	page, err := r.client.List(context.Background(), "", nil)
	if err != nil {
		return nil, err
	}

	return page.Values(), nil
}

// FindInfrastructureResourceGroup returns with the infrastructure resource group of the resource group
func (r *ResourceGroupClient) FindInfrastructureResourceGroup(resourceGroup, clusterName, location string) (*resources.Group, error) {
	groups, err := r.ListGroups()
	if err != nil {
		return nil, err
	}

	infrastructureRg := fmt.Sprintf("MC_%s_%s_%s", resourceGroup, clusterName, location)

	for _, g := range groups {
		if *g.Name == infrastructureRg {
			return &g, nil
		}
	}

	return nil, constants.ErrorNoInfrastructureRG
}
