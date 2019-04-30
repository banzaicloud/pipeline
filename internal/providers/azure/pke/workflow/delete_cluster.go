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

const DeleteClusterWorkflowName = "pke-azure-delete-cluster"

type DeleteClusterWorkflowInput struct {
	OrganizationID       uint
	SecretID             string
	ClusterID            uint
	ClusterName          string
	ResourceGroupName    string
	LoadBalancerName     string
	PublicIPAddressNames []string
	RouteTableName       string
	ScaleSetNames        []string
	SecurityGroupNames   []string
	VirtualNetworkName   string
}

func DeleteClusterWorkflow(ctx workflow.Context, input DeleteClusterWorkflowInput) error {

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}
	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: 30 * time.Minute,
	}
	ctx = workflow.WithChildOptions(workflow.WithActivityOptions(ctx, ao), cwo)

	{
		infraInput := DeleteAzureInfrastructureWorkflowInput{
			OrganizationID:       input.OrganizationID,
			SecretID:             input.SecretID,
			ClusterName:          input.ClusterName,
			ResourceGroupName:    input.ResourceGroupName,
			LoadBalancerName:     input.LoadBalancerName,
			PublicIPAddressNames: input.PublicIPAddressNames,
			RouteTableName:       input.RouteTableName,
			ScaleSetNames:        input.ScaleSetNames,
			SecurityGroupNames:   input.SecurityGroupNames,
			VirtualNetworkName:   input.VirtualNetworkName,
		}
		err := workflow.ExecuteChildWorkflow(ctx, DeleteInfraWorkflowName, infraInput).Get(ctx, nil)
		if err != nil {
			setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	{
		activityInput := DeleteClusterFromStoreActivityInput{
			ClusterID: input.ClusterID,
		}
		err := workflow.ExecuteActivity(ctx, DeleteClusterFromStoreActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			setClusterErrorStatus(ctx, input.ClusterID, err)
			return err
		}
	}

	return nil
}
