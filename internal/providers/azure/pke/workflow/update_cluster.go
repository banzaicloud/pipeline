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

	RoleAssigments  []RoleAssignmentTemplate
	SubnetsToCreate []SubnetTemplate
	SubnetsToDelete []string
	VMSSToCreate    []VirtualMachineScaleSetTemplate
	VMSSToDelete    []string
	VMSSToUpdate    []VirtualMachineScaleSetTemplate
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
	// TODO: update VMSS
	// TODO: create subnets
	// TODO: create VMSS
	// TODO: assign roles
	// TODO: redeploy autoscaler
	return nil
}
