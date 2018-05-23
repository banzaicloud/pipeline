package client

import "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"

// listLocations lists all supported location
func (a *aksClient) listLocations() ([]subscriptions.Location, error) {
	a.LogInfo("Get SubscriptionsClient")
	subsClient, err := a.azureSdk.GetSubscriptionsClient()
	if err != nil {
		return nil, err
	}

	return subsClient.ListLocations()
}
