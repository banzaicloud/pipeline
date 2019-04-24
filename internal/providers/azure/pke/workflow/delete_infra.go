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

	"go.uber.org/cadence/workflow"
)

const DeleteInfraWorkflowName = "pke-azure-delete-infra"

type DeleteAzureInfrastructureWorkflowInput struct {
	OrganizationID    uint
	ClusterName       string
	SecretID          string
	ResourceGroupName string
	TenantID          string
}

func DeleteInfrastructureWorkflow(ctx workflow.Context, input DeleteAzureInfrastructureWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// delete VMSSes
	nodePools := []string{input.ClusterName + "-vmss-master", input.ClusterName + "-vmss-worker"}

	deleteVMSSActivityInput := DeleteVMSSActivityInput{
		OrganizationID:    input.OrganizationID,
		SecretID:          input.SecretID,
		ClusterName:       input.ClusterName,
		ResourceGroupName: input.ResourceGroupName,
	}

	for _, np := range nodePools {

		deleteVMSSActivityInput.VMSSName = np

		err := workflow.ExecuteActivity(ctx, DeleteVMSSActivityName, deleteVMSSActivityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	// delete LB
	deleteLbActivityInput := DeleteLoadBalancerActivityInput{
		OrganizationID:    input.OrganizationID,
		SecretID:          input.SecretID,
		ClusterName:       input.ClusterName,
		ResourceGroupName: input.ResourceGroupName,
		LoadBalancerName:  "kubernetes", // TODO: lb name should be unique per cluster unless it's shared by multiple clusters
	}

	err := workflow.ExecuteActivity(ctx, DeleteLoadBalancerActivityName, deleteLbActivityInput).Get(ctx, nil)
	if err != nil {
		return err
	}

	// delete public ip
	deletePublicIPActivityInput := DeletePublicIPActivityInput{
		OrganizationID:      input.OrganizationID,
		SecretID:            input.SecretID,
		ClusterName:         input.ClusterName,
		ResourceGroupName:   input.ResourceGroupName,
		PublicIPAddressName: input.ClusterName + "-pip-in",
	}
	err = workflow.ExecuteActivity(ctx, DeletePublicIPActivityName, deletePublicIPActivityInput).Get(ctx, nil)
	if err != nil {
		return err
	}

	// delete virtual network
	deleteVNetActivityInput := DeleteVNetActivityInput{
		OrganizationID:    input.OrganizationID,
		SecretID:          input.SecretID,
		ClusterName:       input.ClusterName,
		ResourceGroupName: input.ResourceGroupName,
		VNetName:          input.ClusterName + "-vnet", // TODO: vnet name should come from workflow input instead of deriving it here
	}

	err = workflow.ExecuteActivity(ctx, DeleteVNetActivityName, deleteVNetActivityInput).Get(ctx, nil)
	if err != nil {
		return err
	}

	// delete route table
	deleteRouteTableActivityInput := DeleteRouteTableActivityInput{
		OrganizationID:    input.OrganizationID,
		SecretID:          input.SecretID,
		ClusterName:       input.ClusterName,
		ResourceGroupName: input.ResourceGroupName,
		RouteTableName:    input.ClusterName + "-route-table",
	}

	err = workflow.ExecuteActivity(ctx, DeleteRouteTableActivityName, deleteRouteTableActivityInput).Get(ctx, nil)
	if err != nil {
		return err
	}

	// delete network security groups
	nsgs := []string{input.ClusterName + "-nsg-master", input.ClusterName + "-nsg-worker"}

	deleteNSGActivityInput := DeleteNSGActivityInput{
		OrganizationID:    input.OrganizationID,
		SecretID:          input.SecretID,
		ClusterName:       input.ClusterName,
		ResourceGroupName: input.ResourceGroupName,
	}

	for _, nsgName := range nsgs {

		deleteNSGActivityInput.NSGName = nsgName

		err = workflow.ExecuteActivity(ctx, DeleteNSGActivityName, deleteNSGActivityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil

}
