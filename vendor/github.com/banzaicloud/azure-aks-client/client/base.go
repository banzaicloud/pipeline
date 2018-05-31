package client

import (
	"errors"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/azure-aks-client/utils"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"net/http"
	"time"
)

type ClusterManager interface {
	createOrUpdateCluster(request *cluster.CreateClusterRequest, managedCluster *containerservice.ManagedCluster) (*azure.ResponseWithValue, error)
	deleteCluster(resourceGroup, name string) (*http.Response, error)
	getCluster(resourceGroup, name string) (*containerservice.ManagedCluster, error)
	listClusters() ([]containerservice.ManagedCluster, error)
	getAccessProfiles(resourceGroup, name, roleName string) (*containerservice.ManagedClusterAccessProfile, error)

	listLocations() ([]subscriptions.Location, error)
	listVirtualMachineSizes(location string) ([]compute.VirtualMachineSize, error)
	listK8SVersions(locations, resourceType string) (result *containerservice.OrchestratorVersionProfileListResult, err error)

	listResourceGroups() ([]resources.Group, error)
	findInfrastructureResourceGroup(resourceGroup, clusterName, location string) (*resources.Group, error)

	createVirtualMachine(rg, location, vnetName, subnetName, nsgName, ipName, vmName, nicName string) (*compute.VirtualMachine, error)
	getVirtualMachine(resourceGroup, clusterName, location, vmName string) (*compute.VirtualMachine, error)
	listVirtualMachines(resourceGroup, clusterName, location string) ([]compute.VirtualMachine, error)
	enableManagedServiceIdentity(resourceGroup, clusterName, location string) error
	disableManagedServiceIdentity(resourceGroup, clusterName, location string) error
	assignStorageAccountContributorRole(resourceGroup, clusterName, location string) error
	deleteStorageAccountContributorRole(resourceGroup, clusterName, location string) error

	createNetworkInterface(rg, location, vnetName, subnetName, nsgName, ipName, nicName string) (*network.Interface, error)

	listRoleAssignments() ([]authorization.RoleAssignment, error)
	createRoleAssignment(scope, roleDefinitionId, principalId string) (*authorization.RoleAssignment, error)
	deleteRoleAssignment(scope, roleAssignmentName string) (*authorization.RoleAssignment, error)

	listRoleDefinitions(scope string) ([]authorization.RoleDefinition, error)
	findRoleDefinitionByName(scope, roleName string) (*authorization.RoleDefinition, error)

	getClientId() string
	getClientSecret() string

	With(i interface{})

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

	managedCluster := cluster.GetManagedCluster(request, manager.getClientId(), manager.getClientSecret())
	manager.LogDebugf("Created managed cluster model - %#v", &managedCluster)
	result, err := manager.createOrUpdateCluster(request, managedCluster)
	if err != nil {
		return nil, err
	}

	manager.LogInfo("Create response model")

	return result, nil

}

// DeleteCluster deletes the managed cluster with a specified resource group and name.
func DeleteCluster(manager ClusterManager, name string, resourceGroup string) error {
	manager.LogInfof("Start deleting cluster %s in %s resource group", name, resourceGroup)
	manager.LogDebug("Send request to azure")

	response, err := manager.deleteCluster(resourceGroup, name)
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
		managedCluster, err := manager.getCluster(resourceGroup, name)
		if err != nil {
			return nil, err
		}

		statusCode := managedCluster.StatusCode
		manager.LogInfof("Cluster polling status code: %d", statusCode)

		convertManagedClusterToValue(managedCluster)

		switch statusCode {
		case http.StatusOK:
			response := convertManagedClusterToValue(managedCluster)

			stage := utils.ToS(managedCluster.ProvisioningState)
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

	managedCluster, err := manager.getCluster(resourceGroup, name)
	if err != nil {
		return nil, err
	}

	manager.LogInfof("Status code: %d", managedCluster.StatusCode)

	return &azure.ResponseWithValue{
		StatusCode: managedCluster.StatusCode,
		Value:      *convertManagedClusterToValue(managedCluster),
	}, nil
}

// ListClusters gets a list of managed clusters in the specified subscription. The operation returns properties of each managed
// cluster.
func ListClusters(manager ClusterManager) (*azure.ListResponse, error) {
	manager.LogInfo("Start listing clusters")

	managedClusters, err := manager.listClusters()
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
	profile, err := manager.getAccessProfiles(resourceGroup, name, roleName)
	if err != nil {
		return nil, err
	}

	manager.LogInfof("Status code: %d", profile.StatusCode)
	manager.LogInfo("Create response model")
	return &azure.Config{
		Location: utils.ToS(profile.Location),
		Name:     utils.ToS(profile.Name),
		Properties: struct {
			KubeConfig string `json:"kubeConfig"`
		}{
			KubeConfig: utils.FromBToS(profile.KubeConfig),
		},
	}, nil
}

// GetLocations returns all the locations that are available for resource providers
func GetLocations(manager ClusterManager) ([]string, error) {

	manager.LogInfo("Start listing locations")
	locationList, err := manager.listLocations()
	if err != nil {
		return nil, err
	}

	var locations []string
	for _, loc := range locationList {
		locations = append(locations, *loc.Name)
	}

	return locations, nil
}

// GetVmSizes lists all available virtual machine sizes for a subscription in a location.
func GetVmSizes(manager ClusterManager, location string) ([]string, error) {

	manager.LogInfo("Start listing vm sizes")
	virtualMachineSizes, err := manager.listVirtualMachineSizes(location)
	if err != nil {
		return nil, err
	}

	var sizes []string
	for _, vm := range virtualMachineSizes {
		sizes = append(sizes, *vm.Name)
	}
	return sizes, nil
}

// CreateNetworkInterface create a network interface
func CreateNetworkInterface(manager ClusterManager, rg, location, vnetName, subnetName, nsgName, ipName, nicName string) (*network.Interface, error) {
	manager.LogInfo("Start creating network interface")
	return manager.createNetworkInterface(rg, location, vnetName, subnetName, nsgName, ipName, nicName)
}

// CreateVirtualMachine creates a VM
func CreateVirtualMachine(manager ClusterManager, rg, location, vnetName, subnetName, nsgName, ipName, vmName, nicName string) (*compute.VirtualMachine, error) {
	manager.LogInfo("Start creating virtual machine")
	return manager.createVirtualMachine(rg, location, vnetName, subnetName, nsgName, ipName, vmName, nicName)
}

// ListVirtualMachines returns all VM
func ListVirtualMachines(manager ClusterManager, resourceGroup, clusterName, location string) ([]compute.VirtualMachine, error) {
	manager.LogInfo("Start listing virtual machines")
	return manager.listVirtualMachines(resourceGroup, clusterName, location)
}

// DisableManagedServiceIdentity enables MSI
func EnableManagedServiceIdentity(manager ClusterManager, resourceGroup, clusterName, location string) error {
	manager.LogInfo("Start enabling MSI")
	return manager.enableManagedServiceIdentity(resourceGroup, clusterName, location)
}

// DisableManagedServiceIdentity disables MSI
func DisableManagedServiceIdentity(manager ClusterManager, resourceGroup, clusterName, location string) error {
	manager.LogInfo("Start disabling MSI")
	return manager.disableManagedServiceIdentity(resourceGroup, clusterName, location)
}

// getVirtualMachine retrieves information about a virtual machine
func GetVirtualMachine(manager ClusterManager, resourceGroup, clusterName, location, vmName string) (*compute.VirtualMachine, error) {
	manager.LogInfo("Start getting virtual machine")
	return manager.getVirtualMachine(resourceGroup, clusterName, location, vmName)
}

// FindInfrastructureResourceGroup returns with the infrastructure resource group of the resource group
func FindInfrastructureResourceGroup(manager ClusterManager, resourceGroup, clusterName, location string) (*resources.Group, error) {
	manager.LogInfo("Start finding infrastructure resource group")
	return manager.findInfrastructureResourceGroup(resourceGroup, clusterName, location)
}

// ListGroups returns all resource group
func ListGroups(manager ClusterManager) ([]resources.Group, error) {
	manager.LogInfo("Start listing resource groups")
	return manager.listResourceGroups()
}

// ListRoleAssignments returns all role assignment
func ListRoleAssignments(manager ClusterManager) ([]authorization.RoleAssignment, error) {
	return manager.listRoleAssignments()
}

// CreateRoleAssignment creates role assignment
func CreateRoleAssignment(manager ClusterManager, scope, roleDefinitionId, principalId string) (*authorization.RoleAssignment, error) {
	return manager.createRoleAssignment(scope, roleDefinitionId, principalId)
}

// DeleteRoleAssignment deletes role assignment
func DeleteRoleAssignment(manager ClusterManager, scope, assignmentName string) (*authorization.RoleAssignment, error) {
	return manager.deleteRoleAssignment(scope, assignmentName)
}

// ListRoleDefinitions returns all role definition
func ListRoleDefinitions(manager ClusterManager, scope string) ([]authorization.RoleDefinition, error) {
	return manager.listRoleDefinitions(scope)
}

// FindRoleDefinitionByName filters all role definition by role name and scope
func FindRoleDefinitionByName(manager ClusterManager, scope, roleName string) (*authorization.RoleDefinition, error) {
	return manager.findRoleDefinitionByName(scope, roleName)
}

// AssignStorageAccountContributorRole assign 'Storage Account Contributor' role for all VM in the given resource group
func AssignStorageAccountContributorRole(manager ClusterManager, resourceGroup, clusterName, location string) error {
	return manager.assignStorageAccountContributorRole(resourceGroup, clusterName, location)
}

// DeleteStorageAccountContributorRole deletes 'Storage Account Contributor' role for all VM in the given resource group
func DeleteStorageAccountContributorRole(manager ClusterManager, resourceGroup, clusterName, location string) error {
	return manager.deleteStorageAccountContributorRole(resourceGroup, clusterName, location)
}

// GetKubernetesVersions returns a list of supported kubernetes version in the specified subscription
func GetKubernetesVersions(manager ClusterManager, location string) ([]string, error) {

	manager.LogInfo("Start listing Kubernetes versions")
	resp, err := manager.listK8SVersions(location, string(compute.Kubernetes))
	if err != nil {
		return nil, err
	}

	var versions []string
	if resp.OrchestratorVersionProfileProperties != nil && resp.OrchestratorVersionProfileProperties.Orchestrators != nil {
		for _, v := range *resp.OrchestratorVersionProfileProperties.Orchestrators {
			if v.OrchestratorType != nil && *v.OrchestratorType == string(compute.Kubernetes) {
				versions = utils.AppendIfMissing(versions, *v.OrchestratorVersion)
				if v.Upgrades != nil {
					for _, up := range *v.Upgrades {
						versions = utils.AppendIfMissing(versions, *up.OrchestratorVersion)
					}
				}
			}
		}
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
				Name:  utils.ToS(p.Name),
				Count: utils.ToI(p.Count),
			})
		}
	}

	return &azure.Value{
		Id:       utils.ToS(managedCluster.ID),
		Location: utils.ToS(managedCluster.Location),
		Name:     utils.ToS(managedCluster.Name),
		Properties: azure.Properties{
			ProvisioningState: utils.ToS(managedCluster.ProvisioningState),
			AgentPoolProfiles: profiles,
			Fqdn:              utils.ToS(managedCluster.Fqdn),
		},
	}
}
