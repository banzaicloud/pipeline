package client

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	clientAuthorization "github.com/banzaicloud/azure-aks-client/service/authorization"
	clientCompute "github.com/banzaicloud/azure-aks-client/service/compute"
)

// Storage Account Contributor role constant
const (
	StorageAccountContributor = "Storage Account Contributor"
)

// ListRoleDefinitions gets all role definitions that are applicable at scope and above.
func (a *aksClient) listRoleDefinitions(scope string) ([]authorization.RoleDefinition, error) {
	a.LogInfo("Get RoleDefinitionsClient")
	roleDefinitionsClient, err := a.azureSdk.GetRoleDefinitionsClient()
	if err != nil {
		return nil, err
	}

	return roleDefinitionsClient.ListRoleDefinitions(scope)
}

// findRoleDefinitionByName filters all role definition by role name and scope
func (a *aksClient) findRoleDefinitionByName(scope, roleName string) (*authorization.RoleDefinition, error) {
	a.LogInfo("Get RoleDefinitionsClient")
	roleDefinitionClient, err := a.azureSdk.GetRoleDefinitionsClient()
	if err != nil {
		return nil, err
	}

	a.LogInfof("List role definition with [%s] scope", scope)
	roles, err := roleDefinitionClient.ListRoleDefinitions(scope)
	if err != nil {
		return nil, err
	}

	for _, r := range roles {
		if *r.Properties.RoleName == roleName {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("no role found with the given name[%s]", roleName)
}

// assignStorageAccountContributorRole creates 'Storage Account Contributor' role for all VM in the given resource group
func (a *aksClient) assignStorageAccountContributorRole(resourceGroup, clusterName, location string) error {

	a.LogInfo("Get RoleAssignmentsClient")
	roleAssignClient, err := a.azureSdk.GetRoleAssignmentsClient()
	if err != nil {
		return err
	}

	a.LogInfo("Get ResourceGroupClient")
	resourceGroupClient, err := a.azureSdk.GetResourceGroupClient()
	if err != nil {
		return err
	}

	a.LogInfo("Get VirtualMachineClient")
	vmClient, err := a.azureSdk.GetVirtualMachineClient()
	if err != nil {
		return err
	}

	a.LogInfof("Find infrastructure resource group [%s, %s %s]", resourceGroup, clusterName, location)
	irg, err := resourceGroupClient.FindInfrastructureResourceGroup(resourceGroup, clusterName, location)
	if err != nil {
		return err
	}

	a.LogInfof("infrastructure resource group name: %s", *irg.Name)
	scope := a.getResourceGroupScope(*irg.Name)
	a.LogDebugf("Resource group scope: %s", scope)

	a.LogInfof("Search %s role", StorageAccountContributor)
	role, err := a.findRoleDefinitionByName(scope, StorageAccountContributor)
	if err != nil {
		return err
	}

	a.LogDebugf("Role id: %s", *role.ID)

	a.LogInfof("List virtual machines in %s rg", *irg.Name)
	virtualMachines, err := vmClient.ListVirtualMachines(*irg.Name)
	if err != nil {
		return err
	}

	if err := a.enableMSIVMs(vmClient, virtualMachines, irg, location); err != nil {
		return err
	}

	a.LogInfof("List virtual machines in %s rg", *irg.Name)
	virtualMachines, err = vmClient.ListVirtualMachines(*irg.Name)
	if err != nil {
		return err
	}

	a.LogInfo("List role assignments")
	roleAssignments, err := roleAssignClient.ListRoleAssignments()
	if err != nil {
		return err
	}

	if err := a.assignRoleVMs(roleAssignClient, roleAssignments, virtualMachines, role, scope); err != nil {
		return err
	}

	a.LogInfo("Role assigned to all VM")

	return nil

}

func (a *aksClient) assignRoleVMs(roleAssignClient *clientAuthorization.RoleAssignmentsClient, roleAssignments []authorization.RoleAssignment, virtualMachines []compute.VirtualMachine, role *authorization.RoleDefinition, scope string) error {
	for _, vm := range virtualMachines {
		principalID := vm.Identity.PrincipalID
		roleId := role.ID

		if isRoleAssignedBefore(roleAssignments, scope, *roleId, *principalID) {
			a.LogInfo("The role assignment already exists")
		} else {
			a.LogInfof("Assign role [%s] with scope [%s] to VM [%s] with principalId[%s]", *role.ID, scope, *vm.Name, *principalID)
			_, err := roleAssignClient.CreateRoleAssignment(scope, *roleId, *principalID)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func (a *aksClient) enableMSIVMs(vmClient *clientCompute.VirtualMachinesClient, virtualMachines []compute.VirtualMachine,
	irg *resources.Group, location string) error {
	for _, vm := range virtualMachines {
		if vm.Identity == nil || vm.Identity.Type != compute.ResourceIdentityTypeSystemAssigned {
			a.LogInfof("Enable MSI in vm [%s]", *vm.ID)
			_, err := vmClient.EnableManagedServiceIdentity(&vm, *irg.Name, location)
			if err != nil {
				return err
			}
		} else {
			a.LogInfof("MSI is enabled before in vm [%s]", *vm.ID)
		}
	}

	return nil
}

func isRoleAssignedBefore(roleAssignments []authorization.RoleAssignment, scope, roleId, principalId string) bool {
	for _, assignment := range roleAssignments {
		if assignment.Properties != nil {
			if *assignment.Properties.RoleDefinitionID == roleId && *assignment.Properties.Scope == scope && *assignment.Properties.PrincipalID == principalId {
				return true
			}
		}
	}

	return false
}

// deleteStorageAccountContributorRole deletes 'Storage Account Contributor' role for all VM in the given resource group
func (a *aksClient) deleteStorageAccountContributorRole(resourceGroup, clusterName, location string) error {

	a.LogInfo("Get RoleAssignmentsClient")
	roleAssignClient, err := a.azureSdk.GetRoleAssignmentsClient()
	if err != nil {
		return err
	}

	a.LogInfo("Get VirtualMachineClient")
	resourceGroupClient, err := a.azureSdk.GetResourceGroupClient()
	if err != nil {
		return err
	}

	a.LogInfo("Get ResourceGroupClient")
	vmClient, err := a.azureSdk.GetVirtualMachineClient()
	if err != nil {
		return err
	}

	a.LogInfof("Find infrastructure resource group [%s, %s %s]", resourceGroup, clusterName, location)
	irg, err := resourceGroupClient.FindInfrastructureResourceGroup(resourceGroup, clusterName, location)
	if err != nil {
		return err
	}

	a.LogInfof("List virtual machines in %s rg", *irg.Name)
	virtualMachines, err := vmClient.ListVirtualMachines(*irg.Name)
	if err != nil {
		return err
	}

	scope := a.getResourceGroupScope(*irg.Name)
	a.LogDebugf("Resource group scope: %s", scope)

	for _, vm := range virtualMachines {
		if vm.Identity != nil {

			principalId := vm.Identity.PrincipalID
			a.LogInfof("Get role assignment which assigned to %s", *principalId)
			roles, err := roleAssignClient.GetRoleAssignmentByAssignedTo(*principalId)
			if err != nil {
				return err
			}

			for _, r := range roles {
				assignmentName := *r.Name
				a.LogInfof("Delete role assignment [%s] with scope [%s]: ", assignmentName, scope)
				_, err := roleAssignClient.DeleteRoleAssignments(scope, assignmentName)
				if err != nil {
					return err
				}
			}

			a.LogInfof("Disable MSI in VM [%s-%s]: ", *vm.Name, *vm.ID, scope)
			if vm.Identity.Type == compute.ResourceIdentityTypeSystemAssigned {
				_, err = vmClient.DisableManagedServiceIdentity(&vm, *irg.Name, location)
				if err != nil {
					return err
				}
			}

		}
	}

	return nil
}

// createRoleAssignment creates a role assignment
func (a *aksClient) createRoleAssignment(scope, roleDefinitionId, principalId string) (*authorization.RoleAssignment, error) {
	a.LogInfo("Get RoleAssignmentsClient")
	assignmentsClient, err := a.azureSdk.GetRoleAssignmentsClient()
	if err != nil {
		return nil, err
	}

	a.LogInfof("Create role [%s] assignment in scope [%s] to [%s]", roleDefinitionId, scope, principalId)
	role, err := assignmentsClient.CreateRoleAssignment(scope, roleDefinitionId, principalId)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// deleteRoleAssignment deletes a role assignment
func (a *aksClient) deleteRoleAssignment(scope, roleAssignmentName string) (*authorization.RoleAssignment, error) {
	a.LogInfo("Get RoleAssignmentsClient")
	assignmentsClient, err := a.azureSdk.GetRoleAssignmentsClient()
	if err != nil {
		return nil, err
	}

	a.LogInfof("Delete role [%s] assignment with scope [%s]", roleAssignmentName, scope)
	role, err := assignmentsClient.DeleteRoleAssignments(scope, roleAssignmentName)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// listRoleAssignments returns all role assignment
func (a *aksClient) listRoleAssignments() ([]authorization.RoleAssignment, error) {
	a.LogInfo("Get RoleAssignmentsClient")
	assignmentsClient, err := a.azureSdk.GetRoleAssignmentsClient()
	if err != nil {
		return nil, err
	}

	return assignmentsClient.ListRoleAssignments()
}
