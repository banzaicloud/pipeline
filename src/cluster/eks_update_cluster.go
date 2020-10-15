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
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	eksWorkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksworkflow"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
	sdkCloudFormation "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon/cloudformation"
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
	Tags         map[string]string

	Subnets          []eksWorkflow.Subnet
	ASGSubnetMapping map[string][]eksWorkflow.Subnet

	NodeInstanceRoleID string
	AsgList            []eksWorkflow.AutoscaleGroup
	NodePoolLabels     map[string]map[string]string

	UseGeneratedSSHKey bool
}

type EKSUpdateClusterWorkflow struct {
	nodePoolStore eks.NodePoolStore
}

func NewEKSUpdateClusterWorkflow(nodePoolStore eks.NodePoolStore) (eksUpdateClusterWorkflow *EKSUpdateClusterWorkflow) {
	return &EKSUpdateClusterWorkflow{
		nodePoolStore: nodePoolStore,
	}
}

func waitForActivities(asgFutures []workflow.Future, ctx workflow.Context, clusterID uint) error {
	errs := make([]error, len(asgFutures))
	for i, future := range asgFutures {
		errs[i] = pkgCadence.UnwrapError(future.Get(ctx, nil))
	}
	if err := errors.Combine(errs...); err != nil {
		_ = eksWorkflow.SetClusterStatus(ctx, clusterID, pkgCluster.Warning, err.Error())
		return err
	}
	return nil
}

// UpdateClusterstructureWorkflow executes the Cadence workflow responsible for updating EKS worker nodes
func (w EKSUpdateClusterWorkflow) Execute(ctx workflow.Context, input EKSUpdateClusterstructureWorkflowInput) error {
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

	commonActivityInput := eksWorkflow.EKSActivityInput{
		OrganizationID:            input.OrganizationID,
		SecretID:                  input.SecretID,
		Region:                    input.Region,
		ClusterName:               input.ClusterName,
		AWSClientRequestTokenBase: sdkAmazon.NewNormalizedClientRequestToken(workflow.GetInfo(ctx).WorkflowExecution.ID),
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
			err = errors.WrapIff(pkgCadence.UnwrapError(err), "%q activity failed", clustersetup.ConfigureNodePoolLabelsActivityName)
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
			eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
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

			activityInput := eksWorkflow.DeleteNodePoolWorkflowInput{
				ClusterID:                 input.ClusterID,
				ClusterName:               input.ClusterName,
				NodePoolName:              nodePool.Name,
				OrganizationID:            input.OrganizationID,
				Region:                    input.Region,
				SecretID:                  input.SecretID,
				ShouldUpdateClusterStatus: false,
			}
			ctx = workflow.WithActivityOptions(ctx, aoWithHeartBeat)
			f := workflow.ExecuteChildWorkflow(ctx, eksWorkflow.DeleteNodePoolWorkflowName, activityInput)
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

			// Note: we need to add the node pools created to the database, so the stack ID can be set at creation.
			{
				// Note: deleted and updated node pools are saved later to the database.
				nodePoolsToKeep := make(map[string]bool, len(nodePoolsToDelete)+len(nodePoolsToUpdate))
				for _, nodePoolToDelete := range nodePoolsToDelete {
					nodePoolsToKeep[nodePoolToDelete.Name] = true
				}
				for _, nodePoolToUpdate := range nodePoolsToUpdate {
					nodePoolsToKeep[nodePoolToUpdate.Name] = true
				}

				activityInput := eksWorkflow.SaveNodePoolsActivityInput{
					ClusterID:         input.ClusterID,
					NodePoolsToCreate: nodePoolsToCreate,
					NodePoolsToUpdate: nil,
					NodePoolsToDelete: nil,
					NodePoolsToKeep:   nodePoolsToKeep,
				}

				err := workflow.ExecuteActivity(ctx, eksWorkflow.SaveNodePoolsActivityName, activityInput).Get(ctx, nil)
				if err != nil {
					eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
					return err
				}
			}

			var amiSize int
			{
				activityInput := eksWorkflow.GetAMISizeActivityInput{
					EKSActivityInput: commonActivityInput,
					ImageID:          nodePool.NodeImage,
				}
				var activityOutput eksWorkflow.GetAMISizeActivityOutput
				err = workflow.ExecuteActivity(ctx, eksWorkflow.GetAMISizeActivityName, activityInput).Get(ctx, &activityOutput)
				if err != nil {
					_ = w.nodePoolStore.UpdateNodePoolStatus(
						context.Background(),
						input.OrganizationID,
						input.ClusterID,
						input.ClusterName,
						nodePool.Name,
						eks.NodePoolStatusError,
						fmt.Sprintf("Validation failed: retrieving AMI size failed: %s", err),
					)
					eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck

					return err
				}

				amiSize = activityOutput.AMISize
			}

			var volumeSize int
			{
				activityInput := eksWorkflow.SelectVolumeSizeActivityInput{
					AMISize:            amiSize,
					OptionalVolumeSize: nodePool.NodeVolumeSize,
				}
				var activityOutput eksWorkflow.SelectVolumeSizeActivityOutput
				err = workflow.ExecuteActivity(ctx, eksWorkflow.SelectVolumeSizeActivityName, activityInput).Get(ctx, &activityOutput)
				if err != nil {
					_ = w.nodePoolStore.UpdateNodePoolStatus(
						context.Background(),
						input.OrganizationID,
						input.ClusterID,
						input.ClusterName,
						nodePool.Name,
						eks.NodePoolStatusError,
						fmt.Sprintf("Validation failed: selecting volume size failed: %s", err),
					)
					eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck

					return err
				}

				volumeSize = activityOutput.VolumeSize
			}

			activityInput := eksWorkflow.CreateAsgActivityInput{
				EKSActivityInput: commonActivityInput,
				ClusterID:        input.ClusterID,
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
				NodeVolumeSize:   volumeSize,
				NodeImage:        nodePool.NodeImage,
				NodeInstanceType: nodePool.NodeInstanceType,
				Labels:           nodePool.Labels,
				Tags:             input.Tags,
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

			effectiveImage := nodePool.NodeImage
			effectiveVolumeSize := nodePool.NodeVolumeSize
			if effectiveImage == "" ||
				effectiveVolumeSize == 0 { // Note: needing CF stack for original information for version.
				getCFStackInput := eksWorkflow.GetCFStackActivityInput{
					EKSActivityInput: commonActivityInput,
					StackName:        eksWorkflow.GenerateNodePoolStackName(input.ClusterName, nodePool.Name),
				}
				var getCFStackOutput eksWorkflow.GetCFStackActivityOutput
				err = workflow.ExecuteActivity(ctx, eksWorkflow.GetCFStackActivityName, getCFStackInput).Get(ctx, &getCFStackOutput)
				if err != nil {
					eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
					return err
				}

				var parameters struct {
					NodeImageID    string `mapstructure:"NodeImageId"`
					NodeVolumeSize int    `mapstructure:"NodeVolumeSize"`
				}
				err = sdkCloudFormation.ParseStackParameters(getCFStackOutput.Stack.Parameters, &parameters)
				if err != nil {
					eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
					return err
				}

				if effectiveImage == "" {
					effectiveImage = parameters.NodeImageID
				}

				if effectiveVolumeSize == 0 {
					effectiveVolumeSize = parameters.NodeVolumeSize
				}
			}

			var volumeSize int
			if nodePool.NodeVolumeSize > 0 {
				var amiSize int
				{
					activityInput := eksWorkflow.GetAMISizeActivityInput{
						EKSActivityInput: commonActivityInput,
						ImageID:          effectiveImage,
					}
					var activityOutput eksWorkflow.GetAMISizeActivityOutput
					err = workflow.ExecuteActivity(ctx, eksWorkflow.GetAMISizeActivityName, activityInput).Get(ctx, &activityOutput)
					if err != nil {
						eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
						return err
					}

					amiSize = activityOutput.AMISize
				}

				{
					activityInput := eksWorkflow.SelectVolumeSizeActivityInput{
						AMISize:            amiSize,
						OptionalVolumeSize: nodePool.NodeVolumeSize,
					}
					var activityOutput eksWorkflow.SelectVolumeSizeActivityOutput
					err = workflow.ExecuteActivity(ctx, eksWorkflow.SelectVolumeSizeActivityName, activityInput).Get(ctx, &activityOutput)
					if err != nil {
						eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
						return err
					}

					volumeSize = activityOutput.VolumeSize
					effectiveVolumeSize = volumeSize
				}
			}

			var nodePoolVersion string
			{
				activityInput := eksworkflow.CalculateNodePoolVersionActivityInput{
					Image:      effectiveImage,
					VolumeSize: effectiveVolumeSize,
				}

				activityOptions := ao
				activityOptions.StartToCloseTimeout = 30 * time.Second
				activityOptions.RetryPolicy = &cadence.RetryPolicy{
					InitialInterval:    10 * time.Second,
					BackoffCoefficient: 1.01,
					MaximumAttempts:    10,
					MaximumInterval:    10 * time.Minute,
				}

				var output eksworkflow.CalculateNodePoolVersionActivityOutput

				err = workflow.ExecuteActivity(
					workflow.WithActivityOptions(ctx, activityOptions),
					eksworkflow.CalculateNodePoolVersionActivityName,
					activityInput,
				).Get(ctx, &output)
				if err != nil {
					eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
					return err
				}

				nodePoolVersion = output.Version
			}

			activityInput := eksWorkflow.UpdateAsgActivityInput{
				EKSActivityInput: commonActivityInput,
				StackName:        eksWorkflow.GenerateNodePoolStackName(input.ClusterName, nodePool.Name),
				ScaleEnabled:     input.ScaleEnabled,
				Name:             nodePool.Name,
				Version:          nodePoolVersion,
				NodeSpotPrice:    nodePool.NodeSpotPrice,
				Autoscaling:      nodePool.Autoscaling,
				NodeMinCount:     nodePool.NodeMinCount,
				NodeMaxCount:     nodePool.NodeMaxCount,
				Count:            nodePool.Count,
				NodeVolumeSize:   volumeSize,
				NodeImage:        nodePool.NodeImage,
				NodeInstanceType: nodePool.NodeInstanceType,
				Labels:           nodePool.Labels,
				Tags:             input.Tags,
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
		// Note: created node pools are saved earlier to the database to be able
		// to set the stack ID at creation.
		nodePoolsToKeep := make(map[string]bool, len(nodePoolsToCreate))
		for _, nodePoolToCreate := range nodePoolsToCreate {
			nodePoolsToKeep[nodePoolToCreate.Name] = true
		}

		activityInput := eksWorkflow.SaveNodePoolsActivityInput{
			ClusterID:         input.ClusterID,
			NodePoolsToCreate: nil,
			NodePoolsToUpdate: nodePoolsToUpdate,
			NodePoolsToDelete: nodePoolsToDelete,
			NodePoolsToKeep:   nodePoolsToKeep,
		}

		err := workflow.ExecuteActivity(ctx, eksWorkflow.SaveNodePoolsActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
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
			err = errors.WrapIff(pkgCadence.UnwrapError(err), "%q activity failed", RunPostHookActivityName)
			eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, err.Error()) // nolint: errcheck
			return err
		}
	}

	_ = eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Running, pkgCluster.RunningMessage)
	return nil
}
