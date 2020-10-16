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

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
)

const DeleteInfraWorkflowName = "eks-delete-infra"

// DeleteInfrastructureWorkflowInput holds data needed by the delete EKS cluster infrastructure workflow
type DeleteInfrastructureWorkflowInput struct {
	OrganizationID   uint
	SecretID         string
	Region           string
	ClusterName      string
	ClusterID        uint
	ClusterUID       string
	NodePoolNames    []string
	DefaultUser      bool
	GeneratedSSHUsed bool
}

// DeleteInfrastructureWorkflow executes the Cadence workflow responsible for deleting EKS
// cluster infrastructure such as VPC, subnets, EKS master nodes, worker nodes, etc
func DeleteInfrastructureWorkflow(ctx workflow.Context, input DeleteInfrastructureWorkflowInput) error {
	logger := workflow.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
	)

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

	aoWithHeartbeat := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		HeartbeatTimeout:       45 * time.Second,
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
		OrganizationID:            input.OrganizationID,
		SecretID:                  input.SecretID,
		Region:                    input.Region,
		ClusterName:               input.ClusterName,
		AWSClientRequestTokenBase: sdkAmazon.NewNormalizedClientRequestToken(workflow.GetInfo(ctx).WorkflowExecution.ID),
	}

	// get VPC ID
	var getVpcConfigActivityOutput GetVpcConfigActivityOutput
	{
		activityInput := GetVpcConfigActivityInput{
			EKSActivityInput: eksActivityInput,
			StackName:        GenerateStackNameForCluster(input.ClusterName),
		}

		if err := workflow.ExecuteActivity(ctx, GetVpcConfigActivityName, activityInput).Get(ctx, &getVpcConfigActivityOutput); err != nil {
			return err
		}
	}

	if getVpcConfigActivityOutput.VpcID != "" {
		// get ELBs created by the EKS cluster
		var ownedELBsOutput GetOwnedELBsActivityOutput
		{
			activityInput := GetOwnedELBsActivityInput{
				EKSActivityInput: eksActivityInput,
				VpcID:            getVpcConfigActivityOutput.VpcID,
			}

			if err := workflow.ExecuteActivity(ctx, GetOwnedELBsActivityName, activityInput).Get(ctx, &ownedELBsOutput); err != nil {
				return err
			}
		}

		// wait for ELBs to be releases by EKS
		{
			if len(ownedELBsOutput.LoadBalancerNames) > 0 {
				activityInput := WaitELBsDeletionActivityActivityInput{
					EKSActivityInput:  eksActivityInput,
					LoadBalancerNames: ownedELBsOutput.LoadBalancerNames,
				}

				if err := workflow.ExecuteActivity(ctx, WaitELBsDeletionActivityName, activityInput).Get(ctx, nil); err != nil {
					return err
				}
			}
		}
	}

	// delete node pool stacks
	asgDeleteFutures := make([]workflow.Future, 0)
	for _, nodePoolName := range input.NodePoolNames {
		logger.With("nodePoolName", nodePoolName).Info("node pool stack will be deleted")

		activityInput := DeleteNodePoolWorkflowInput{
			ClusterID:                 input.ClusterID,
			ClusterName:               input.ClusterName,
			NodePoolName:              nodePoolName,
			OrganizationID:            input.OrganizationID,
			Region:                    input.Region,
			SecretID:                  input.SecretID,
			ShouldUpdateClusterStatus: false,
		}
		ctx := workflow.WithActivityOptions(ctx, aoWithHeartbeat)
		f := workflow.ExecuteChildWorkflow(ctx, DeleteNodePoolWorkflowName, activityInput)
		asgDeleteFutures = append(asgDeleteFutures, f)
	}

	// wait for ASG's to be deleted
	errs := make([]error, len(asgDeleteFutures))
	for i, future := range asgDeleteFutures {
		errs[i] = future.Get(ctx, nil)
	}
	if err := errors.Combine(errs...); err != nil {
		return err
	}

	// delete EKS control plane
	{
		activityInput := DeleteControlPlaneActivityInput{
			EKSActivityInput: eksActivityInput,
		}
		ctx := workflow.WithActivityOptions(ctx, aoWithHeartbeat)
		if err := workflow.ExecuteActivity(ctx, DeleteControlPlaneActivityName, activityInput).Get(ctx, nil); err != nil {
			return err
		}
	}

	// delete SSH key
	var deleteSSHKeyActivityFeature workflow.Future
	if input.GeneratedSSHUsed {
		{
			activityInput := DeleteSshKeyActivityInput{
				EKSActivityInput: eksActivityInput,
				SSHKeyName:       GenerateSSHKeyNameForCluster(input.ClusterName),
			}
			deleteSSHKeyActivityFeature = workflow.ExecuteActivity(ctx, DeleteSshKeyActivityName, activityInput)
		}
	}

	if getVpcConfigActivityOutput.VpcID != "" {
		// retrieve orphan NICs
		var getNicsOutput GetOrphanNICsActivityOutput
		{
			securityGroups := []string{getVpcConfigActivityOutput.NodeSecurityGroupID, getVpcConfigActivityOutput.SecurityGroupID}
			activityInput := GetOrphanNICsActivityInput{
				EKSActivityInput: eksActivityInput,
				VpcID:            getVpcConfigActivityOutput.VpcID,
				SecurityGroupIDs: securityGroups,
			}

			if err := workflow.ExecuteActivity(ctx, GetOrphanNICsActivityName, activityInput).Get(ctx, &getNicsOutput); err != nil {
				return err
			}
		}

		// delete orphan NIC's
		deleteNICFutures := make([]workflow.Future, 0)
		for _, nicID := range getNicsOutput.NicList {
			activityInput := DeleteOrphanNICActivityInput{
				EKSActivityInput: eksActivityInput,
				NicID:            nicID,
			}
			f := workflow.ExecuteActivity(ctx, DeleteOrphanNICActivityName, activityInput)
			deleteNICFutures = append(deleteNICFutures, f)
		}

		// wait for NIC's to be deleted
		errs = make([]error, len(deleteNICFutures))
		for i, future := range deleteNICFutures {
			errs[i] = future.Get(ctx, nil)
		}
		if err := errors.Combine(errs...); err != nil {
			return err
		}
	}

	if deleteSSHKeyActivityFeature != nil {
		if err := deleteSSHKeyActivityFeature.Get(ctx, nil); err != nil {
			return err
		}
	}

	// get subnet stack names
	var subnetStackOutput GetSubnetStacksActivityOutput
	{
		activityInput := GetSubnetStacksActivityInput{
			eksActivityInput,
		}

		if err := workflow.ExecuteActivity(ctx, GetSubnetStacksActivityName, activityInput).Get(ctx, &subnetStackOutput); err != nil {
			return err
		}
	}

	// delete subnets
	deleteSubnetFutures := make([]workflow.Future, 0)
	for _, subnetStackName := range subnetStackOutput.StackNames {
		activityInput := DeleteStackActivityInput{
			EKSActivityInput: eksActivityInput,
			StackName:        subnetStackName,
		}
		ctx := workflow.WithActivityOptions(ctx, aoWithHeartbeat)
		f := workflow.ExecuteActivity(ctx, DeleteStackActivityName, activityInput)
		deleteSubnetFutures = append(deleteSubnetFutures, f)
	}

	// wait for stacks to be deleted
	errs = make([]error, len(deleteSubnetFutures))
	for i, future := range deleteSubnetFutures {
		errs[i] = future.Get(ctx, nil)
	}
	if err := errors.Combine(errs...); err != nil {
		return err
	}

	// delete cluster stack (VPC, etc)
	{
		activityInput := DeleteStackActivityInput{
			EKSActivityInput: eksActivityInput,
			StackName:        GenerateStackNameForCluster(input.ClusterName),
		}
		ctx := workflow.WithActivityOptions(ctx, aoWithHeartbeat)
		if err := workflow.ExecuteActivity(ctx, DeleteStackActivityName, activityInput).Get(ctx, nil); err != nil {
			return err
		}
	}

	// delete IAM user and roles
	{
		activityInput := DeleteStackActivityInput{
			EKSActivityInput: eksActivityInput,
			StackName:        generateStackNameForIam(input.ClusterName),
		}
		ctx := workflow.WithActivityOptions(ctx, aoWithHeartbeat)
		if err := workflow.ExecuteActivity(ctx, DeleteStackActivityName, activityInput).Get(ctx, nil); err != nil {
			return err
		}
	}

	return nil
}
