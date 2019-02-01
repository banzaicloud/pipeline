// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2018-03-31/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	uuid "github.com/satori/go.uuid"
)

// ListKubernetesVersions returns available k8s versions at the specified location
func (client *ContainerServicesClient) ListKubernetesVersions(ctx context.Context, location string) (res []string, err error) {
	l, err := client.ListOrchestrators(ctx, location, string(compute.Kubernetes))
	if err != nil {
		return
	}
	if l.Orchestrators == nil {
		return
	}
	vs := make(map[string]bool)
	for _, o := range *l.Orchestrators {
		if o.OrchestratorType != nil && *o.OrchestratorType == string(compute.Kubernetes) {
			vs[*o.OrchestratorVersion] = true
			if o.Upgrades != nil {
				for _, u := range *o.Upgrades {
					vs[*u.OrchestratorVersion] = true
				}
			}
		}
	}
	res = make([]string, 0, len(vs))
	for v := range vs {
		res = append(res, v)
	}
	return
}

// ListAll returns all resource groups
func (client *GroupsClient) ListAll(ctx context.Context, filter string, top *int32) (res []resources.Group, err error) {
	rp, err := client.List(ctx, filter, top)
	for rp.NotDone() {
		if err != nil {
			return
		}
		res = append(res, rp.Values()...)
		err = rp.NextWithContext(ctx)
	}
	return
}

// CreateOrUpdateAndWaitForIt creates or updates the specified cluster and waits for the operation to finish, returning the resulting cluster
func (client *ManagedClustersClient) CreateOrUpdateAndWaitForIt(ctx context.Context, resourceGroupName, clusterName string, params *containerservice.ManagedCluster) (*containerservice.ManagedCluster, error) {
	future, err := client.CreateOrUpdate(ctx, resourceGroupName, clusterName, *params)
	if err != nil {
		return nil, err
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return nil, err
	}
	res, err := future.Result(client.ManagedClustersClient)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// DeleteAndWaitForIt deletes the specified cluster and waits for the operation to finish
func (client *ManagedClustersClient) DeleteAndWaitForIt(ctx context.Context, resourceGroupName, clusterName string) error {
	future, err := client.Delete(ctx, resourceGroupName, clusterName)
	if err != nil {
		return err
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return err
	}
	_, err = future.Result(client.ManagedClustersClient)
	if err != nil {
		return err
	}
	return nil
}

// AssignRole creates a role assignment featuring the specified role definition and principal and returns the assignment
func (client *RoleAssignmentsClient) AssignRole(ctx context.Context, scope, roleDefinitionID, principalID string) (authorization.RoleAssignment, error) {
	roleAssignmentName := uuid.NewV1().String()
	return client.Create(ctx, scope, roleAssignmentName, authorization.RoleAssignmentCreateParameters{
		Properties: &authorization.RoleAssignmentProperties{
			RoleDefinitionID: &roleDefinitionID,
			PrincipalID:      &principalID,
		},
	})
}

// ListAll returns all role assignments
func (client *RoleAssignmentsClient) ListAll(ctx context.Context, filter string) (res []authorization.RoleAssignment, err error) {
	rp, err := client.List(ctx, filter)
	for rp.NotDone() {
		if err != nil {
			return
		}
		res = append(res, rp.Values()...)
		err = rp.NextWithContext(ctx)
	}
	return
}

// FindByRoleName returns a role definition with the matching name or nil, if no such definition can be found
func (client *RoleDefinitionsClient) FindByRoleName(ctx context.Context, scope, roleName string) (*authorization.RoleDefinition, error) {
	rp, err := client.List(ctx, scope, "")
	for rp.NotDone() {
		if err != nil {
			return nil, err
		}
		for _, def := range rp.Values() {
			if *def.Properties.RoleName == roleName {
				return &def, nil
			}
		}
		err = rp.NextWithContext(ctx)
	}
	return nil, err
}

// CreateOrUpdateAndWaitForIt creates or updates the specified virtual machine and waits for the operation to finish, returning the resulting VM
func (client *VirtualMachinesClient) CreateOrUpdateAndWaitForIt(ctx context.Context, resourceGroupName string, vm *compute.VirtualMachine) (*compute.VirtualMachine, error) {
	future, err := client.CreateOrUpdate(ctx, resourceGroupName, *vm.Name, *vm)
	if err != nil {
		return nil, err
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return nil, err
	}
	res, err := future.Result(client.VirtualMachinesClient)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// ListAll returns all virtual machines belonging to the specified resource group
func (client *VirtualMachinesClient) ListAll(ctx context.Context, resourceGroupName string) (res []compute.VirtualMachine, err error) {
	rp, err := client.List(ctx, resourceGroupName)
	for rp.NotDone() {
		if err != nil {
			return res, err
		}
		res = append(res, rp.Values()...)
		err = rp.NextWithContext(ctx)
	}
	return
}

// ListMachineTypes returns all machine types available at the specified location
func (client *VirtualMachineSizesClient) ListMachineTypes(ctx context.Context, location string) (res cluster.MachineTypes, err error) {
	l, err := client.List(ctx, location)
	if err != nil {
		return
	}
	if l.Value == nil {
		return
	}
	res = make([]string, 0, len(*l.Value))
	for _, vmSize := range *l.Value {
		res = append(res, *vmSize.Name)
	}
	return
}
