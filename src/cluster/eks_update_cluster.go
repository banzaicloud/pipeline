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

package cluster

import (
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	eksWorkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/pkg/brn"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

const EKSUpdateClusterWorkflowName = "eks-update-cluster"

// EKSUpdateClusterstructureWorkflowInput holds data needed to update EKS cluster worker node pools
type EKSUpdateClusterstructureWorkflowInput struct {
	Region         string
	OrganizationID uint
	SecretID       string
	ConfigSecretID string

	ClusterID    uint
	ClusterUID   string
	ClusterName  string
	ScaleEnabled bool

	Subnets          []eksWorkflow.Subnet
	ASGSubnetMapping map[string][]eksWorkflow.Subnet

	NodeInstanceRoleID string
	AsgList            []eksWorkflow.AutoscaleGroup
	NodePoolLabels     map[string]map[string]string

	UseGeneratedSSHKey bool
}

func waitForActivities(asgFutures []workflow.Future, ctx workflow.Context, clusterID uint) error {
	errs := make([]error, len(asgFutures))
	for i, future := range asgFutures {
		var activityOutput eksWorkflow.CreateAsgActivityOutput
		errs[i] = future.Get(ctx, &activityOutput)
	}
	if err := errors.Combine(errs...); err != nil {
		_ = eksWorkflow.SetClusterStatus(ctx, clusterID, pkgCluster.Warning, err.Error())
		return err
	}
	return nil
}

// UpdateClusterstructureWorkflow executes the Cadence workflow responsible for updating EKS worker nodes
func EKSUpdateClusterWorkflow(ctx workflow.Context, input EKSUpdateClusterstructureWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", eksWorkflow.ErrReasonStackFailed},
		},
	}

	aoWithHeartBeat := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		HeartbeatTimeout:       45 * time.Second,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          2 * time.Second,
			BackoffCoefficient:       1.5,
			MaximumInterval:          30 * time.Second,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", eksWorkflow.ErrReasonStackFailed},
		},
	}

	logger := workflow.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
	)

	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	commonActivityInput := eksWorkflow.EKSActivityInput{
		OrganizationID:            input.OrganizationID,
		SecretID:                  input.SecretID,
		Region:                    input.Region,
		ClusterName:               input.ClusterName,
		AWSClientRequestTokenBase: workflowID,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// set up node pool labels set
	{
		activityInput := clustersetup.ConfigureNodePoolLabelsActivityInput{
			ConfigSecretID: brn.New(input.OrganizationID, brn.SecretResourceType, input.ConfigSecretID).String(),
			Labels:         input.NodePoolLabels,
		}
		err := workflow.ExecuteActivity(ctx, clustersetup.ConfigureNodePoolLabelsActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			err = errors.WrapIff(err, "%q activity failed", clustersetup.ConfigureNodePoolLabelsActivityName)
			eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	var vpcActivityOutput eksWorkflow.GetVpcConfigActivityOutput
	{
		activityInput := &eksWorkflow.GetVpcConfigActivityInput{
			EKSActivityInput: commonActivityInput,
			StackName:        eksWorkflow.GenerateStackNameForCluster(input.ClusterName),
		}
		err := workflow.ExecuteActivity(ctx, eksWorkflow.GetVpcConfigActivityName, activityInput).Get(ctx, &vpcActivityOutput)
		if err != nil {
			eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	nodePoolsToCreate := make(map[string]eksWorkflow.AutoscaleGroup, 0)
	nodePoolsToUpdate := make(map[string]eksWorkflow.AutoscaleGroup, 0)
	nodePoolsToDelete := make(map[string]eksWorkflow.AutoscaleGroup, 0)

	// first delete node pools
	asgFutures := make([]workflow.Future, 0)
	for _, nodePool := range input.AsgList {
		log := logger.With("nodePool", nodePool.Name)

		if nodePool.Delete {
			log.Info("node pool will be deleted")
			nodePoolsToDelete[nodePool.Name] = nodePool

			activityInput := eksWorkflow.DeleteStackActivityInput{
				EKSActivityInput: commonActivityInput,
				StackName:        eksWorkflow.GenerateNodePoolStackName(input.ClusterName, nodePool.Name),
			}
			ctx = workflow.WithActivityOptions(ctx, aoWithHeartBeat)
			f := workflow.ExecuteActivity(ctx, eksWorkflow.DeleteStackActivityName, activityInput)
			asgFutures = append(asgFutures, f)
		}
	}

	// wait for AutoScalingGroups to be deleted
	err := waitForActivities(asgFutures, ctx, input.ClusterID)
	if err != nil {
		return err
	}

	asgFutures = make([]workflow.Future, 0)
	for _, nodePool := range input.AsgList {
		log := logger.With("nodePool", nodePool.Name)

		if nodePool.Create {
			log.Info("node pool will be created")
			nodePoolsToCreate[nodePool.Name] = nodePool

			asgSubnets := input.ASGSubnetMapping[nodePool.Name]
			for i := range asgSubnets {
				for _, sn := range input.Subnets {
					if (asgSubnets[i].SubnetID == "" && sn.Cidr == asgSubnets[i].Cidr) ||
						(asgSubnets[i].SubnetID != "" && sn.SubnetID == asgSubnets[i].SubnetID) {
						asgSubnets[i].SubnetID = sn.SubnetID
						asgSubnets[i].Cidr = sn.Cidr
						asgSubnets[i].AvailabilityZone = sn.AvailabilityZone
					}
				}
			}

			activityInput := eksWorkflow.CreateAsgActivityInput{
				EKSActivityInput: commonActivityInput,
				StackName:        eksWorkflow.GenerateNodePoolStackName(input.ClusterName, nodePool.Name),

				ScaleEnabled: input.ScaleEnabled,

				Subnets: asgSubnets,

				VpcID:               vpcActivityOutput.VpcID,
				SecurityGroupID:     vpcActivityOutput.SecurityGroupID,
				NodeSecurityGroupID: vpcActivityOutput.NodeSecurityGroupID,
				NodeInstanceRoleID:  input.NodeInstanceRoleID,

				Name:             nodePool.Name,
				NodeSpotPrice:    nodePool.NodeSpotPrice,
				Autoscaling:      nodePool.Autoscaling,
				NodeMinCount:     nodePool.NodeMinCount,
				NodeMaxCount:     nodePool.NodeMaxCount,
				Count:            nodePool.Count,
				NodeImage:        nodePool.NodeImage,
				NodeInstanceType: nodePool.NodeInstanceType,
				Labels:           nodePool.Labels,
			}
			if input.UseGeneratedSSHKey {
				activityInput.SSHKeyName = eksWorkflow.GenerateSSHKeyNameForCluster(input.ClusterName)
			}

			ctx = workflow.WithActivityOptions(ctx, aoWithHeartBeat)
			f := workflow.ExecuteActivity(ctx, eksWorkflow.CreateAsgActivityName, activityInput)
			asgFutures = append(asgFutures, f)
		} else if !nodePool.Delete {
			// update nodePool
			log.Info("node pool will be updated")
			nodePoolsToUpdate[nodePool.Name] = nodePool

			activityInput := eksWorkflow.UpdateAsgActivityInput{
				EKSActivityInput: commonActivityInput,
				StackName:        eksWorkflow.GenerateNodePoolStackName(input.ClusterName, nodePool.Name),
				ScaleEnabled:     input.ScaleEnabled,
				Name:             nodePool.Name,
				NodeSpotPrice:    nodePool.NodeSpotPrice,
				Autoscaling:      nodePool.Autoscaling,
				NodeMinCount:     nodePool.NodeMinCount,
				NodeMaxCount:     nodePool.NodeMaxCount,
				Count:            nodePool.Count,
				NodeImage:        nodePool.NodeImage,
				NodeInstanceType: nodePool.NodeInstanceType,
				Labels:           nodePool.Labels,
			}
			ctx = workflow.WithActivityOptions(ctx, aoWithHeartBeat)
			f := workflow.ExecuteActivity(ctx, eksWorkflow.UpdateAsgActivityName, activityInput)
			asgFutures = append(asgFutures, f)
		}
	}

	// wait for AutoScalingGroups to be created & updated
	err = waitForActivities(asgFutures, ctx, input.ClusterID)
	if err != nil {
		return err
	}

	// delete, update, create node pools
	{
		activityInput := eksWorkflow.SaveNodePoolsActivityInput{
			ClusterID:         input.ClusterID,
			NodePoolsToCreate: nodePoolsToCreate,
			NodePoolsToUpdate: nodePoolsToUpdate,
			NodePoolsToDelete: nodePoolsToDelete,
		}

		err := workflow.ExecuteActivity(ctx, eksWorkflow.SaveNodePoolsActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	// redeploy autoscaler
	{
		activityInput := RunPostHookActivityInput{
			ClusterID: input.ClusterID,
			HookName:  pkgCluster.InstallClusterAutoscalerPostHook,
			Status:    pkgCluster.Updating,
		}

		err := workflow.ExecuteActivity(ctx, RunPostHookActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			err = errors.WrapIff(err, "%q activity failed", RunPostHookActivityName)
			eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	_ = eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Running, pkgCluster.RunningMessage)
	return nil
}
