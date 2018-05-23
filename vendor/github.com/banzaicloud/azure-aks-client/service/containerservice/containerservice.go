package containerservice

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"context"
)

// ContainerServicesClient responsible for K8S versions
type ContainerServicesClient struct {
	client *containerservice.ContainerServicesClient
}

// NewContainerServicesClient creates a new 'ContainerServicesClient' instance
func NewContainerServicesClient(authorizer autorest.Authorizer, subscriptionId string) *ContainerServicesClient {
	containerServicesClient := containerservice.NewContainerServicesClient(subscriptionId)
	containerServicesClient.Authorizer = authorizer

	return &ContainerServicesClient{
		client: &containerServicesClient,
	}
}

// ListOrchestrators lists all supported Kubernetes verison in the given location
func (csc *ContainerServicesClient) ListOrchestrators(location, resourceType string) (*containerservice.OrchestratorVersionProfileListResult, error) {
	result, err := csc.client.ListOrchestrators(context.Background(), location, resourceType)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
