package client

import (
	"errors"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/constants"
	"net/http"
	"time"
)

type ClusterManager interface {
	CreateOrUpdate(request *cluster.CreateClusterRequest, managedCluster *containerservice.ManagedCluster) (*containerservice.ManagedCluster, error)
	Delete(resourceGroup, name string) (*http.Response, error)
	Get(resourceGroup, name string) (containerservice.ManagedCluster, error)
	List() ([]containerservice.ManagedCluster, error)
	GetAccessProfiles(resourceGroup, name, roleName string) (containerservice.ManagedClusterAccessProfile, error)
	ListLocations() (subscriptions.LocationListResult, error)
	ListVmSizes(location string) (result compute.VirtualMachineSizeListResult, err error)
	ListVersions(locations, resourceType string) (result containerservice.OrchestratorVersionProfileListResult, err error)

	GetClientId() string
	GetClientSecret() string

	LogDebug(args ...interface{})
	LogInfo(args ...interface{})
	LogWarn(args ...interface{})
	LogError(args ...interface{})
	LogFatal(args ...interface{})
	LogPanic(args ...interface{})
	LogDebugf(format string, args ...interface{})
	LogInfof(format string, args ...interface{})
	LogWarnf(format string, args ...interface{})
	LogErrorf(format string, args ...interface{})
	LogFatalf(format string, args ...interface{})
	LogPanicf(format string, args ...interface{})
}

// CreateUpdateCluster creates or updates a managed cluster with the specified configuration for agents and Kubernetes
// version.
func CreateUpdateCluster(manager ClusterManager, request *cluster.CreateClusterRequest) (*azure.ResponseWithValue, error) {

	if request == nil {
		return nil, errors.New("Empty request")
	}

	manager.LogInfo("Start create/update cluster")
	manager.LogDebugf("CreateRequest: %v", request)
	manager.LogInfo("Validate cluster create/update request")

	if err := request.Validate(); err != nil {
		return nil, err
	}
	manager.LogInfo("Validate passed")

	managedCluster := cluster.GetManagedCluster(request, manager.GetClientId(), manager.GetClientSecret())
	manager.LogDebugf("Created managed cluster model - %#v", &managedCluster)
	manager.LogDebug("Send request to azure")
	result, err := manager.CreateOrUpdate(request, managedCluster)
	if err != nil {
		return nil, err
	}

	manager.LogInfo("Create response model")

	return &azure.ResponseWithValue{
		StatusCode: result.Response.StatusCode,
		Value:      *convertManagedClusterToValue(result),
	}, nil

}

// DeleteCluster deletes the managed cluster with a specified resource group and name.
func DeleteCluster(manager ClusterManager, name string, resourceGroup string) error {
	manager.LogInfof("Start deleting cluster %s in %s resource group", name, resourceGroup)
	manager.LogDebug("Send request to azure")

	response, err := manager.Delete(resourceGroup, name)
	if err != nil {
		return err
	}

	manager.LogInfof("Status code: %d", response.StatusCode)

	return nil
}

// PollingCluster polls until the cluster ready or an error occurs
func PollingCluster(manager ClusterManager, name string, resourceGroup string) (*azure.ResponseWithValue, error) {
	const stageSuccess = "Succeeded"
	const stageFailed = "Failed"
	const waitInSeconds = 10

	manager.LogInfof("Start polling cluster: %s [%s]", name, resourceGroup)

	manager.LogDebug("Start loop")

	result := azure.ResponseWithValue{}
	for isReady := false; !isReady; {

		manager.LogDebug("Send request to azure")
		managedCluster, err := manager.Get(resourceGroup, name)
		if err != nil {
			return nil, err
		}

		statusCode := managedCluster.StatusCode
		manager.LogInfof("Cluster polling status code: %d", statusCode)

		convertManagedClusterToValue(&managedCluster)

		switch statusCode {
		case http.StatusOK:
			response := convertManagedClusterToValue(&managedCluster)

			stage := *managedCluster.ProvisioningState
			manager.LogInfof("Cluster stage is %s", stage)

			switch stage {
			case stageSuccess:
				isReady = true
				result.Update(http.StatusCreated, *response)
			case stageFailed:
				return nil, constants.ErrorAzureCLusterStageFailed
			default:
				manager.LogInfo("Waiting for cluster ready...")
				time.Sleep(waitInSeconds * time.Second)
			}

		default:
			return nil, errors.New("status code is not OK")
		}
	}

	return &result, nil
}

// GetCluster gets the details of the managed cluster with a specified resource group and name.
func GetCluster(manager ClusterManager, name string, resourceGroup string) (*azure.ResponseWithValue, error) {

	manager.LogInfof("Start getting aks cluster: %s [%s]", name, resourceGroup)

	managedCluster, err := manager.Get(resourceGroup, name)
	if err != nil {
		return nil, err
	}

	manager.LogInfof("Status code: %d", managedCluster.StatusCode)

	return &azure.ResponseWithValue{
		StatusCode: managedCluster.StatusCode,
		Value:      *convertManagedClusterToValue(&managedCluster),
	}, nil
}

// ListClusters gets a list of managed clusters in the specified subscription. The operation returns properties of each managed
// cluster.
func ListClusters(manager ClusterManager) (*azure.ListResponse, error) {
	manager.LogInfo("Start listing clusters")

	managedClusters, err := manager.List()
	if err != nil {
		return nil, err
	}

	manager.LogInfo("Create response model")
	response := azure.ListResponse{StatusCode: http.StatusOK, Value: azure.Values{
		Value: convertManagedClustersToValues(managedClusters),
	}}
	return &response, nil
}

// GetClusterConfig gets the given cluster kubeconfig
func GetClusterConfig(manager ClusterManager, name, resourceGroup, roleName string) (*azure.Config, error) {

	manager.LogInfof("Start getting %s cluster's config in %s, role name: %s", name, resourceGroup, roleName)

	manager.LogDebug("Send request to azure")
	profile, err := manager.GetAccessProfiles(resourceGroup, name, roleName)
	if err != nil {
		return nil, err
	}

	manager.LogInfof("Status code: %d", profile.StatusCode)
	manager.LogInfo("Create response model")
	return &azure.Config{
		Location: *profile.Location,
		Name:     *profile.Name,
		Properties: struct {
			KubeConfig string `json:"kubeConfig"`
		}{
			KubeConfig: string(*profile.KubeConfig),
		},
	}, nil
}

// GetLocations returns all the locations that are available for resource providers
func GetLocations(manager ClusterManager) ([]string, error) {

	manager.LogInfo("Start listing locations")
	resp, err := manager.ListLocations()
	if err != nil {
		return nil, err
	}

	var locations []string
	for _, loc := range *resp.Value {
		locations = append(locations, *loc.Name)
	}

	return locations, nil
}

// GetVmSizes lists all available virtual machine sizes for a subscription in a location.
func GetVmSizes(manager ClusterManager, location string) ([]string, error) {

	manager.LogInfo("Start listing vm sizes")
	resp, err := manager.ListVmSizes(location)
	if err != nil {
		return nil, err
	}

	var sizes []string
	for _, vm := range *resp.Value {
		sizes = append(sizes, *vm.Name)
	}
	return sizes, nil
}

// GetKubernetesVersions returns a list of supported kubernetes version in the specified subscription
func GetKubernetesVersions(manager ClusterManager, location string) ([]string, error) {

	manager.LogInfo("Start listing Kubernetes versions")
	resp, err := manager.ListVersions(location, string(compute.Kubernetes))
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, v := range *resp.OrchestratorVersionProfileProperties.Orchestrators {
		versions = append(versions, *v.OrchestratorVersion)
	}

	return versions, nil
}

// convertManagedClustersToValues returns []Value with the managed clusters properties
func convertManagedClustersToValues(managedCluster []containerservice.ManagedCluster) []azure.Value {
	var values []azure.Value
	for _, mc := range managedCluster {
		values = append(values, *convertManagedClusterToValue(&mc))
	}
	return values
}

// convertManagedClusterToValue returns Value with the ManagedCluster properties
func convertManagedClusterToValue(managedCluster *containerservice.ManagedCluster) *azure.Value {

	var profiles []azure.Profile
	if managedCluster.AgentPoolProfiles != nil {
		for _, p := range *managedCluster.AgentPoolProfiles {
			profiles = append(profiles, azure.Profile{
				Name:  *p.Name,
				Count: int(*p.Count),
			})
		}
	}

	return &azure.Value{
		Id:       *managedCluster.ID,
		Location: *managedCluster.Location,
		Name:     *managedCluster.Name,
		Properties: azure.Properties{
			ProvisioningState: *managedCluster.ProvisioningState,
			AgentPoolProfiles: profiles,
			Fqdn:              *managedCluster.Fqdn,
		},
	}
}
