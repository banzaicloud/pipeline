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

const CreateInfraWorkflowName = "pke-azure-create-infra"

type CreateAzureInfrastructureWorkflowInput struct {
	OrganizationID    uint
	SecretID          string
	ResourceGroupName string
}

func CreateInfrastructureWorkflow(ctx workflow.Context, input CreateAzureInfrastructureWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToCloseTimeout: 15 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// Create VNET
	{
		activityInput := CreateVnetActivityInput{
			Name:              "",
			CIDR:              "",
			Location:          "",
			ResourceGroupName: input.ResourceGroupName,
			OrganizationID:    input.OrganizationID,
			SecretID:          input.SecretID,
		}

		err := workflow.ExecuteActivity(ctx, CreateVnetActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	// Create Subnet
	{
		activityInput := CreateSubnetActivityInput{
			Name:               "",
			CIDR:               "",
			VirtualNetworkName: "",
			ResourceGroupName:  input.ResourceGroupName,
			OrganizationID:     input.OrganizationID,
			SecretID:           input.SecretID,
		}

		err := workflow.ExecuteActivity(ctx, CreateSubnetActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	// CreateNetworkSecurity Group

	// Create BasicLoadbalancer

	// Create ScaleSet

	// Set AssignRolePolicy
	return nil
}
