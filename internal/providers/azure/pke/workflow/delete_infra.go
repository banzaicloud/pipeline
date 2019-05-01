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

	LoadBalancerName     string
	PublicIPAddressNames []string
	RouteTableName       string
	ScaleSetNames        []string
	SecurityGroupNames   []string
	VirtualNetworkName   string
}

func DeleteInfrastructureWorkflow(ctx workflow.Context, input DeleteAzureInfrastructureWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// Delete VMSSs
	{
		futures := make([]workflow.Future, 0, len(input.ScaleSetNames))
		for _, n := range input.ScaleSetNames {
			activityInput := DeleteVMSSActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				VMSSName:          n,
			}

			futures = append(futures, workflow.ExecuteActivity(ctx, DeleteVMSSActivityName, activityInput))
		}
		for _, future := range futures {
			if err := future.Get(ctx, nil); err != nil {
				return err
			}
		}
	}

	// Delete load balancer
	{
		activityInput := DeleteLoadBalancerActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			LoadBalancerName:  input.LoadBalancerName,
		}

		if err := workflow.ExecuteActivity(ctx, DeleteLoadBalancerActivityName, activityInput).Get(ctx, nil); err != nil {
			return err
		}
	}

	// Delete public IP
	for _, n := range input.PublicIPAddressNames {
		activityInput := DeletePublicIPActivityInput{
			OrganizationID:      input.OrganizationID,
			SecretID:            input.SecretID,
			ClusterName:         input.ClusterName,
			ResourceGroupName:   input.ResourceGroupName,
			PublicIPAddressName: n,
		}

		if err := workflow.ExecuteActivity(ctx, DeletePublicIPActivityName, activityInput).Get(ctx, nil); err != nil {
			return err
		}
	}

	// Delete virtual network
	{
		activityInput := DeleteVNetActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			VNetName:          input.VirtualNetworkName,
		}

		if err := workflow.ExecuteActivity(ctx, DeleteVNetActivityName, activityInput).Get(ctx, nil); err != nil {
			return err
		}
	}

	// Delete route table
	{
		activityInput := DeleteRouteTableActivityInput{
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
			ClusterName:       input.ClusterName,
			ResourceGroupName: input.ResourceGroupName,
			RouteTableName:    input.RouteTableName,
		}

		if err := workflow.ExecuteActivity(ctx, DeleteRouteTableActivityName, activityInput).Get(ctx, nil); err != nil {
			return err
		}
	}

	// Delete network security groups
	{
		futures := make([]workflow.Future, len(input.SecurityGroupNames))
		for i, n := range input.SecurityGroupNames {
			activityInput := DeleteNSGActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				ClusterName:       input.ClusterName,
				ResourceGroupName: input.ResourceGroupName,
				NSGName:           n,
			}
			futures[i] = workflow.ExecuteActivity(ctx, DeleteNSGActivityName, activityInput)
		}
		for _, future := range futures {
			if err := future.Get(ctx, nil); err != nil {
				return err
			}
		}
	}

	return nil
}
