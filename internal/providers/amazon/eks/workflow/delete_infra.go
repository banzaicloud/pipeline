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

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
)

const DeleteInfraWorkflowName = "eks-delete-infra"

// DeleteInfrastructureWorkflowInput holds data needed by the delete EKS cluster infrastructure workflow
type DeleteInfrastructureWorkflowInput struct {
	OrganizationID uint
	SecretID       string
	Region         string

	ClusterName string
}

// DeleteInfrastructureWorkflow executes the Cadence workflow responsible for deleting EKS
// cluster infrastructure such as VPC, subnets, EKS master nodes, worker nodes, etc
func DeleteInfrastructureWorkflow(ctx workflow.Context, input DeleteInfrastructureWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", ErrReasonStackFailed},
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	eksActivityInput := EKSActivityInput{
		OrganizationID: input.OrganizationID,
		SecretID:       input.SecretID,
		Region:         input.Region,
		ClusterName:    input.ClusterName,
	}

	// get VPC ID
	var getVpcConfigActivityOutput GetVpcConfigActivityOutput
	{
		activityInput := GetVpcConfigActivityInput{
			EKSActivityInput: eksActivityInput,
			StackName:        generateStackNameForCluster(input.ClusterName),
		}

		if err := workflow.ExecuteActivity(ctx, GetVpcConfigActivityName, activityInput).Get(ctx, &getVpcConfigActivityOutput); err != nil {
			return err
		}
	}

	// get ELBs created by the EKS cluster
	var ownedELBsOutput GetOwnedELBsActivityOutput
	{
		activityInput := GetOwnedELBsActivityInput{
			OrganizationID: input.OrganizationID,
			SecretID:       input.SecretID,
			Region:         input.Region,
			ClusterName:    input.ClusterName,
			VpcID:          getVpcConfigActivityOutput.VpcID,
		}

		if err := workflow.ExecuteActivity(ctx, GetOwnedELBsActivityName, activityInput).Get(ctx, &ownedELBsOutput); err != nil {
			return err
		}

	}

	// wait for ELBs to be releases by EKS
	{
		if len(ownedELBsOutput.LoadBalancerNames) > 0 {
			activityInput := WaitELBsDeletionActivityActivityInput{
				OrganizationID:    input.OrganizationID,
				SecretID:          input.SecretID,
				Region:            input.Region,
				ClusterName:       input.ClusterName,
				LoadBalancerNames: ownedELBsOutput.LoadBalancerNames,
			}

			if err := workflow.ExecuteActivity(ctx, WaitELBsDeletionActivityName, activityInput).Get(ctx, nil); err != nil {
				return err
			}
		}
	}

	//TODO: delete node pools
	//TODO: delete EKS control plane
	//TODO: delete SSH key
	//TODO: delete cluster user
	//TODO: delete orphan NICs
	//TODO: delete subnets
	//TODO: delete VPC
	//TODO: delete IAM user and roles

	return nil
}
