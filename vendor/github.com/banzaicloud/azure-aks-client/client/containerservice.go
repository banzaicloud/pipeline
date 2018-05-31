package client

import "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"

// listK8SVersions lists all supported Kubernetes verison in the given location
func (a *aksClient) listK8SVersions(location, resourceType string) (result *containerservice.OrchestratorVersionProfileListResult, err error) {
	a.LogInfo("Get ContainerServicesClient")
	containerServicesClient, err := a.azureSdk.GetContainerServicesClient()
	if err != nil {
		return nil, err
	}

	return containerServicesClient.ListOrchestrators(location, resourceType)
}
