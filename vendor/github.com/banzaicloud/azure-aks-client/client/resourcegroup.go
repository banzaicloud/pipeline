package client

import (
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"fmt"
)

// listResourceGroups gets all the resource groups for a subscription
func (a *aksClient) listResourceGroups() ([]resources.Group, error) {
	a.LogInfo("Get ResourceGroupClient")
	groupClient, err := a.azureSdk.GetResourceGroupClient()
	if err != nil {
		return nil, err
	}

	return groupClient.ListGroups()
}

// findInfrastructureResourceGroup returns with the infrastructure resource group of the resource group
func (a *aksClient) findInfrastructureResourceGroup(resourceGroup, clusterName, location string) (*resources.Group, error) {
	a.LogInfo("Get ResourceGroupClient")
	groupClient, err := a.azureSdk.GetResourceGroupClient()
	if err != nil {
		return nil, err
	}

	return groupClient.FindInfrastructureResourceGroup(resourceGroup, clusterName, location)
}

// getResourceGroupScope returns a resource group scope
func (a *aksClient) getResourceGroupScope(infrastructureRgName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/", a.azureSdk.ServicePrincipal.SubscriptionID, infrastructureRgName)
}
