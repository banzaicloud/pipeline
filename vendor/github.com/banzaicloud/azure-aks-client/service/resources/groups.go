package resources

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/banzaicloud/azure-aks-client/errors"
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

// CreateOrUpdate creates/updates a resource group in the given location with the given name
func (r *ResourceGroupClient) CreateOrUpdate(resourceGroup, location string) (*resources.Group, error) {

	group, err := r.client.CreateOrUpdate(context.Background(), resourceGroup, resources.Group{
		Location: to.StringPtr(location),
	})
	if err != nil {
		return nil, err
	}

	return &group, nil

}

// Delete deletes an existing resource group by name
func (r *ResourceGroupClient) Delete(resourceGroup string) error {
	future, err := r.client.Delete(context.Background(), resourceGroup)
	if err != nil {
		return err
	}

	return future.WaitForCompletion(context.Background(), r.client.Client)
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

	return nil, errors.ErrNoInfrastructureRG
}
