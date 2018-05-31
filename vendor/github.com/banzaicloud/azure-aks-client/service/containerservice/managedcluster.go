package containerservice

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"context"
)

// ManagedClustersClient responsible for AKS clusters
type ManagedClustersClient struct {
	client *containerservice.ManagedClustersClient
}

// NewManagedClustersClient creates a new 'ManagedClustersClient' instance
func NewManagedClustersClient(authorizer autorest.Authorizer, subscriptionId string) *ManagedClustersClient {
	managedClusterClient := containerservice.NewManagedClustersClient(subscriptionId)
	managedClusterClient.Authorizer = authorizer

	return &ManagedClustersClient{
		client: &managedClusterClient,
	}
}

// CreateOrUpdate creates or updates a managed cluster
func (mcc *ManagedClustersClient) CreateOrUpdate(resourceGroup, name string, managedCluster *containerservice.ManagedCluster) (*containerservice.ManagedClustersCreateOrUpdateFuture, error) {
	future, err := mcc.client.CreateOrUpdate(context.Background(), resourceGroup, name, *managedCluster)
	if err != nil {
		return nil, err
	}
	return &future, nil
}

// GetManagedCLuster returns managed cluster info from cloud
func (mcc *ManagedClustersClient) GetManagedCLuster(resourceGroup, name string) (*containerservice.ManagedCluster, error) {
	managedCluster, err := mcc.client.Get(context.Background(), resourceGroup, name)
	if err != nil {
		return nil, err
	}
	return &managedCluster, nil
}

// DeleteManagedCluster deletes a managed cluster
func (mcc *ManagedClustersClient) DeleteManagedCluster(resourceGroup, name string) (*containerservice.ManagedClustersDeleteFuture, error) {
	future, err := mcc.client.Delete(context.Background(), resourceGroup, name)
	if err != nil {
		return nil, err
	}
	return &future, err
}

// GetAccessProfiles returns access profiles including kubeconfig
func (mcc *ManagedClustersClient) GetAccessProfiles(resourceGroup, name, roleName string) (*containerservice.ManagedClusterAccessProfile, error) {
	profile, err := mcc.client.GetAccessProfiles(context.Background(), resourceGroup, name, roleName)
	if err != nil {
		return nil, err
	}
	return &profile, err
}

// ListClusters returns all managed cluster in the cloud
func (mcc *ManagedClustersClient) ListClusters() ([]containerservice.ManagedCluster, error) {
	page, err := mcc.client.List(context.Background())
	if err != nil {
		return nil, err
	}
	return page.Values(), nil
}
