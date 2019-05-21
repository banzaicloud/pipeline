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

	"github.com/banzaicloud/pipeline/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/goph/emperror"
	"go.uber.org/cadence/workflow"
)

const UpdateClusterWorkflowName = "pke-azure-update-cluster"

// UpdateClusterWorkflowInput
type UpdateClusterWorkflowInput struct {
	OrganizationID      uint
	SecretID            string
	ClusterID           uint
	ClusterName         string
	ResourceGroupName   string
	LoadBalancerName    string
	PublicIPAddressName string
	RouteTableName      string
	VirtualNetworkName  string

	RoleAssignments []RoleAssignmentTemplate
	SubnetsToCreate []SubnetTemplate
	SubnetsToDelete []string
	VMSSToCreate    []VirtualMachineScaleSetTemplate
	VMSSToDelete    []NodePoolAndVMSS
	VMSSToUpdate    []VirtualMachineScaleSetChanges
}

type NodePoolAndVMSS struct {
	NodePoolName string
	VMSSName     string
}

func UpdateClusterWorkflow(ctx workflow.Context, input UpdateClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var providers CollectUpdateClusterProvidersActivityOutput
	{
		activityInput := CollectUpdateClusterProvidersActivityInput{
			OrganizationID:      input.OrganizationID,
			SecretID:            input.SecretID,
			ResourceGroupName:   input.ResourceGroupName,
			LoadBalancerName:    input.LoadBalancerName,
			PublicIPAddressName: input.PublicIPAddressName,
			RouteTableName:      input.RouteTableName,
			VirtualNetworkName:  input.VirtualNetworkName,
		}
		err := workflow.ExecuteActivity(ctx, CollectUpdateClusterProvidersActivityName, activityInput).Get(ctx, &providers)
		if err = emperror.Wrapf(err, "%q activity failed", CollectUpdateClusterProvidersActivityName); err != nil {
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
			return err
		}
	}
	{
		futures := make([]workflow.Future, len(input.VMSSToDelete))
		for i, npAndVMSS := range input.VMSSToDelete {
			activityInput := DeleteVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				VMSSName:          npAndVMSS.VMSSName,
			}
			futures[i] = workflow.ExecuteActivity(ctx, DeleteVMSSActivityName, activityInput)
		}
		for _, f := range futures {
			if err := emperror.Wrapf(f.Get(ctx, nil), "%q activity failed", DeleteVMSSActivityName); err != nil {
				setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
				return err
			}
		}
	}
	{
		nodePoolsToDelete := make([]string, len(input.VMSSToDelete))
		for i, npAndVMSS := range input.VMSSToDelete {
			nodePoolsToDelete[i] = npAndVMSS.NodePoolName
		}
		activityInput := DeleteNodePoolFromStoreActivityInput{
			ClusterID:     input.ClusterID,
			NodePoolNames: nodePoolsToDelete,
		}
		if err := workflow.ExecuteActivity(ctx, DeleteNodePoolFromStoreActivityName, activityInput).Get(ctx, nil); err != nil {
			err = emperror.Wrapf(err, "%q activity failed", DeleteNodePoolFromStoreActivityName)
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
			return err
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
			if err := emperror.Wrapf(f.Get(ctx, nil), "%q activity failed", DeleteSubnetActivityName); err != nil {
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
			if err := emperror.Wrapf(f.Get(ctx, nil), "%q activity failed", UpdateVMSSActivityName); err != nil {
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
				Subnet:             subnet.Render(providers.RouteTableIDProvider, providers.SecurityGroupIDProvider),
			}
			futures[activityInput.Subnet.Name] = workflow.ExecuteActivity(ctx, CreateSubnetActivityName, activityInput)
		}
		for name, f := range futures {
			var activityOutput CreateSubnetActivityOutput
			if err := emperror.Wrapf(f.Get(ctx, &activityOutput), "%q activity failed", CreateSubnetActivityName); err != nil {
				setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
				return err
			}
			providers.SubnetIDProvider.Put(name, activityOutput.SubnetID)
		}
	}
	createdVMSSOutputs := make(map[string]CreateVMSSActivityOutput)
	{
		type futureAndNodePoolName struct {
			future       workflow.Future
			nodePoolName string
		}
		futures := make(map[string]futureAndNodePoolName, len(input.VMSSToCreate))
		for _, vmss := range input.VMSSToCreate {
			activityInput := CreateVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterID:         input.ClusterID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				ScaleSet:          vmss.Render(providers.BackendAddressPoolIDProvider, providers.InboundNATPoolIDProvider, providers.PublicIPAddressProvider, providers.SecurityGroupIDProvider, providers.SubnetIDProvider),
			}
			futures[activityInput.ScaleSet.Name] = futureAndNodePoolName{
				future:       workflow.ExecuteActivity(ctx, CreateVMSSActivityName, activityInput),
				nodePoolName: vmss.NodePoolName,
			}
		}
		var createVMSSErr error
		var nodePoolsToDelete []string
		for name, f := range futures {
			var activityOutput CreateVMSSActivityOutput
			if createVMSSErr = emperror.Wrapf(f.future.Get(ctx, &activityOutput), "%q activity failed", CreateVMSSActivityName); createVMSSErr != nil {
				setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, createVMSSErr.Error())
				nodePoolsToDelete = append(nodePoolsToDelete, f.nodePoolName)
			} else {
				createdVMSSOutputs[name] = activityOutput
			}
		}
		if createVMSSErr != nil {
			{
				activityInput := DeleteNodePoolFromStoreActivityInput{
					ClusterID:     input.ClusterID,
					NodePoolNames: nodePoolsToDelete,
				}
				if err := workflow.ExecuteActivity(ctx, DeleteNodePoolFromStoreActivityName, activityInput).Get(ctx, nil); err != nil {
					err = emperror.Wrapf(err, "%q activity failed", DeleteNodePoolFromStoreActivityName)
					setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
				}
			}
			return createVMSSErr
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
			if err := emperror.Wrapf(f.Get(ctx, nil), "%q activity failed", AssignRoleActivityName); err != nil {
				setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
				return err
			}
		}
	}
	// redeploy autoscaler
	{
		activityInput := cluster.RunPostHookActivityInput{
			ClusterID: input.ClusterID,
			HookName:  pkgCluster.InstallClusterAutoscalerPostHook,
			Status:    pkgCluster.Updating,
		}

		err := workflow.ExecuteActivity(ctx, cluster.RunPostHookActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			err = emperror.Wrapf(err, "%q activity failed", cluster.RunPostHookActivityName)
			setClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error())
			return err
		}
	}

	setClusterStatus(ctx, input.ClusterID, pkgCluster.Running, pkgCluster.RunningMessage)

	return nil
}
