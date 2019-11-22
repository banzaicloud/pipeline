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
)

const DeleteInfraWorkflowName = "eks-delete-infra"

// DeleteInfrastructureWorkflowInput holds data needed by the delete EKS cluster infrastructure workflow
type DeleteInfrastructureWorkflowInput struct {
	OrganizationID uint
	SecretID       string
	Region         string
	ClusterName    string
	ClusterUID     string
	NodePoolNames  []string
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

	ctx = workflow.WithActivityOptions(ctx, ao)

	eksActivityInput := EKSActivityInput{
		OrganizationID:            input.OrganizationID,
		SecretID:                  input.SecretID,
		Region:                    input.Region,
		ClusterName:               input.ClusterName,
		AWSClientRequestTokenBase: input.ClusterUID,
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

	// retrieve node pool stack names
	var getNodepoolStacksActivityOutput GetNodepoolStacksActivityOutput
	{
		activityInput := GetNodepoolStacksActivityInput{
			EKSActivityInput: eksActivityInput,
			NodePoolNames:    input.NodePoolNames,
		}

		if err := workflow.ExecuteActivity(ctx, GetNodepoolStacksActivityName, activityInput).Get(ctx, &getNodepoolStacksActivityOutput); err != nil {
			return err
		}
	}

	// delete node pool stacks
	asgDeleteFutures := make([]workflow.Future, 0)
	for _, stackName := range getNodepoolStacksActivityOutput.StackNames {
		logger.With("nodePoolStackName", stackName).Info("node pool stack will be deleted")

		activityInput := DeleteStackActivityInput{
			EKSActivityInput: eksActivityInput,
			StackName:        stackName,
		}
		f := workflow.ExecuteActivity(ctx, DeleteStackActivityName, activityInput)
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

		if err := workflow.ExecuteActivity(ctx, DeleteControlPlaneActivityName, activityInput).Get(ctx, nil); err != nil {
			return err
		}
	}

	// delete SSH key
	var deleteSSHKeyAcitivityFeature workflow.Future
	{
		activityInput := DeleteSshKeyActivityInput{
			EKSActivityInput: eksActivityInput,
			SSHKeyName:       generateSSHKeyNameForCluster(input.ClusterName),
		}
		deleteSSHKeyAcitivityFeature = workflow.ExecuteActivity(ctx, DeleteSshKeyActivityName, activityInput)
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

	if err := deleteSSHKeyAcitivityFeature.Get(ctx, nil); err != nil {
		return err
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
			StackName:        generateStackNameForCluster(input.ClusterName),
		}

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

		if err := workflow.ExecuteActivity(ctx, DeleteStackActivityName, activityInput).Get(ctx, nil); err != nil {
			return err
		}
	}

	// delete cluster user acess key & sercret from secret store
	var deleteClusterUserAccessKeyActivityFeature workflow.Future
	{
		activityInput := DeleteClusterUserAccessKeyActivityInput{
			EKSActivityInput: eksActivityInput,
		}
		deleteClusterUserAccessKeyActivityFeature = workflow.ExecuteActivity(ctx, DeleteClusterUserAccessKeyActivityName, activityInput)
	}

	if err := deleteClusterUserAccessKeyActivityFeature.Get(ctx, nil); err != nil {
		return err
	}

	return nil
}
