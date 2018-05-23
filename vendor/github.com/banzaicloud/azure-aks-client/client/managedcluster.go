package client

import (
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"io/ioutil"
	"fmt"
	"net/http"
	"github.com/banzaicloud/azure-aks-client/utils"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"encoding/json"
)

// createOrUpdateCluster creates or updates a managed cluster
func (a *aksClient) createOrUpdateCluster(request *cluster.CreateClusterRequest, managedCluster *containerservice.ManagedCluster) (*azure.ResponseWithValue, error) {

	a.LogInfo("Get ManagedClusterClient")
	managedClusterClient, err := a.azureSdk.GetManagedClusterClient()
	if err != nil {
		return nil, err
	}

	a.LogInfof("Send request to Azure: %#v", *request)
	res, err := managedClusterClient.CreateOrUpdate(request.ResourceGroup, request.Name, managedCluster)
	if err != nil {
		return nil, err
	}

	a.LogInfo("Read response body")
	resp := res.Response()
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error during cluster creation: %s", err.Error())
	}

	a.LogInfof("Status code: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// something went wrong, create failed
		errResp := utils.CreateErrorFromValue(resp.StatusCode, value)
		return nil, errResp
	}

	a.LogInfo("Create response model")
	v := azure.Value{}
	json.Unmarshal([]byte(value), &v)

	return &azure.ResponseWithValue{
		StatusCode: resp.StatusCode,
		Value:      v,
	}, nil

}

// getCluster returns managed cluster info from cloud
func (a *aksClient) getCluster(resourceGroup, name string) (*containerservice.ManagedCluster, error) {
	a.LogInfo("Get ManagedClusterClient")
	managedClusterClient, err := a.azureSdk.GetManagedClusterClient()
	if err != nil {
		return nil, err
	}
	a.LogInfo("Send request to Azure [%s in %s]", name, resourceGroup)
	return managedClusterClient.GetManagedCLuster(resourceGroup, name)
}

// getAccessProfiles returns access profiles including kubeconfig
func (a *aksClient) getAccessProfiles(resourceGroup, name, roleName string) (*containerservice.ManagedClusterAccessProfile, error) {
	a.LogInfo("Get ManagedClusterClient")
	managedClusterClient, err := a.azureSdk.GetManagedClusterClient()
	if err != nil {
		return nil, err
	}

	return managedClusterClient.GetAccessProfiles(resourceGroup, name, roleName)
}

// listClusters returns all managed cluster in the cloud
func (a *aksClient) listClusters() ([]containerservice.ManagedCluster, error) {
	a.LogInfo("Get ManagedClusterClient")
	managedClusterClient, err := a.azureSdk.GetManagedClusterClient()
	if err != nil {
		return nil, err
	}

	return managedClusterClient.ListClusters()
}

// delete deletes a managed cluster
func (a *aksClient) deleteCluster(resourceGroup, name string) (*http.Response, error) {
	a.LogInfo("Get ManagedClusterClient")
	managedClusterClient, err := a.azureSdk.GetManagedClusterClient()
	if err != nil {
		return nil, err
	}

	resp, err := managedClusterClient.DeleteManagedCluster(resourceGroup, name)
	if err != nil {
		return nil, err
	}
	return resp.Response(), nil
}
