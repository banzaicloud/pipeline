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

package workflow

import (
	"time"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/goph/emperror"
	"go.uber.org/cadence/workflow"
)

const UpdateClusterWorkflowName = "pke-azure-update-cluster"

// UpdateClusterWorkflowInput
type UpdateClusterWorkflowInput struct {
	OrganizationID     uint
	SecretID           string
	ClusterID          uint
	ClusterName        string
	ResourceGroupName  string
	VirtualNetworkName string

	RoleAssignments []RoleAssignmentTemplate
	SubnetsToCreate []SubnetTemplate
	SubnetsToDelete []string
	VMSSToCreate    []VirtualMachineScaleSetTemplate
	VMSSToDelete    []string
	VMSSToUpdate    []VirtualMachineScaleSetChanges

	BackendAddressPoolIDProvider MapResourceIDByNameProvider
	InboundNATPoolIDProvider     MapResourceIDByNameProvider
	PublicIPAddressProvider      ConstantIPAddressProvider
	RouteTableIDProvider         ConstantResourceIDProvider
	SecurityGroupIDProvider      MapResourceIDByNameProvider
	SubnetIDProvider             MapResourceIDByNameProvider
}

func UpdateClusterWorkflow(ctx workflow.Context, input UpdateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	{
		futures := make([]workflow.Future, len(input.VMSSToDelete))
		for i, vmssName := range input.VMSSToDelete {
			activityInput := DeleteVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				VMSSName:          vmssName,
			}
			futures[i] = workflow.ExecuteActivity(ctx, DeleteVMSSActivityName, activityInput)
		}
		for _, f := range futures {
			if err := emperror.WrapWith(f.Get(ctx, nil), "activity failed", "activityName", DeleteVMSSActivityName); err != nil {
				setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
				return err
			}
		}
	}
	{
		futures := make([]workflow.Future, len(input.SubnetsToDelete))
		for i, subnetName := range input.SubnetsToDelete {
			activityInput := DeleteSubnetActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				VNetName:          input.VirtualNetworkName,
				SubnetName:        subnetName,
			}
			futures[i] = workflow.ExecuteActivity(ctx, DeleteSubnetActivityName, activityInput)
		}
		for _, f := range futures {
			if err := emperror.WrapWith(f.Get(ctx, nil), "activity failed", "activityName", DeleteSubnetActivityName); err != nil {
				setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
				return err
			}
		}
	}
	{
		futures := make([]workflow.Future, len(input.VMSSToUpdate))
		for i, vmssChanges := range input.VMSSToUpdate {
			activityInput := UpdateVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				Changes:           vmssChanges,
			}
			futures[i] = workflow.ExecuteActivity(ctx, UpdateVMSSActivityName, activityInput)
		}
		for _, f := range futures {
			if err := emperror.WrapWith(f.Get(ctx, nil), "activity failed", "activityName", UpdateVMSSActivityName); err != nil {
				setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
				return err
			}
		}
	}
	{
		futures := make(map[string]workflow.Future, len(input.SubnetsToCreate))
		for _, subnet := range input.SubnetsToCreate {
			activityInput := CreateSubnetActivityInput{
				OrganizationID:     input.OrganizationID,
				SecretID:           input.SecretID,
				ClusterName:        input.ClusterName,
				ResourceGroupName:  input.ResourceGroupName,
				VirtualNetworkName: input.VirtualNetworkName,
				Subnet:             subnet.Render(input.RouteTableIDProvider, input.SecurityGroupIDProvider),
			}
			futures[activityInput.Subnet.Name] = workflow.ExecuteActivity(ctx, CreateSubnetActivityName, activityInput)
		}
		for name, f := range futures {
			var activityOutput CreateSubnetActivityOutput
			if err := emperror.WrapWith(f.Get(ctx, &activityOutput), "activity failed", "activityName", CreateSubnetActivityName); err != nil {
				setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
				return err
			}
			input.SubnetIDProvider.Put(name, activityOutput.SubnetID)
		}
	}
	createdVMSSOutputs := make(map[string]CreateVMSSActivityOutput)
	{
		futures := make(map[string]workflow.Future, len(input.VMSSToCreate))
		for _, vmss := range input.VMSSToCreate {
			activityInput := CreateVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterID:         input.ClusterID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				ScaleSet:          vmss.Render(input.BackendAddressPoolIDProvider, input.InboundNATPoolIDProvider, input.PublicIPAddressProvider, input.SecurityGroupIDProvider, input.SubnetIDProvider),
			}
			futures[activityInput.ScaleSet.Name] = workflow.ExecuteActivity(ctx, CreateVMSSActivityName, activityInput)
		}
		for name, f := range futures {
			var activityOutput CreateVMSSActivityOutput
			if err := emperror.WrapWith(f.Get(ctx, &activityOutput), "activity failed", "activityName", CreateVMSSActivityName); err != nil {
				setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
				return err
			}
			createdVMSSOutputs[name] = activityOutput
		}
	}
	{
		futures := make([]workflow.Future, len(input.RoleAssignments))
		for i, ra := range input.RoleAssignments {
			activityInput := AssignRoleActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				RoleAssignment:    ra.Render(mapVMSSPrincipalIDProvider(createdVMSSOutputs)),
			}
			futures[i] = workflow.ExecuteActivity(ctx, AssignRoleActivityName, activityInput)
		}
		for _, f := range futures {
			if err := emperror.WrapWith(f.Get(ctx, nil), "activity failed", "activityName", AssignRoleActivityName); err != nil {
				setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
				return err
			}
		}
	}
	// TODO: redeploy autoscaler
	return nil
}
