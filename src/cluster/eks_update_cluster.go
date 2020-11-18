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

	ClusterID     uint
	ClusterName   string
	ScaleEnabled  bool
	Tags          map[string]string
	UpdaterUserID uint

	DeletableNodePoolNames []string
	NewNodePools           []eks.NewNodePool
	NewNodePoolSubnetIDs   map[string][]string
	NodePoolLabels         map[string]map[string]string // TODO: remove when UpdateNodePoolWorkflow is refactored.
	UpdatedNodePools       []eksWorkflow.AutoscaleGroup
}

type EKSUpdateClusterWorkflow struct {
	nodePoolStore eks.NodePoolStore
}

func NewEKSUpdateClusterWorkflow(nodePoolStore eks.NodePoolStore) (eksUpdateClusterWorkflow *EKSUpdateClusterWorkflow) {
	return &EKSUpdateClusterWorkflow{
		nodePoolStore: nodePoolStore,
	}
}

func waitForActivities(futures []workflow.Future, ctx workflow.Context, clusterID uint) error {
	errs := make([]error, len(futures))
	for i, future := range futures {
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
	//
	// TODO: update when UpdateNodePoolWorkflow is refactored. The plan is to
	// update the node pool labels as part of the UpdateNodePoolWorkflow (as it
	// is with CreateNodePoolWorkflow) and thus this becomes obsolete (requires
	// field update at CreateNodePoolWorkflow call as well).
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

	// first delete node pools
	deleteNodePoolFutures := make([]workflow.Future, 0, len(input.DeletableNodePoolNames))
	for _, nodePoolName := range input.DeletableNodePoolNames {
		log := logger.With("nodePool", nodePoolName)

		log.Info("node pool will be deleted")

		activityInput := eksWorkflow.DeleteNodePoolWorkflowInput{
			ClusterID:                 input.ClusterID,
			ClusterName:               input.ClusterName,
			NodePoolName:              nodePoolName,
			OrganizationID:            input.OrganizationID,
			Region:                    input.Region,
			SecretID:                  input.SecretID,
			ShouldUpdateClusterStatus: false,
		}
		ctx = workflow.WithActivityOptions(ctx, aoWithHeartBeat)
		deleteNodePoolFuture := workflow.ExecuteChildWorkflow(ctx, eksWorkflow.DeleteNodePoolWorkflowName, activityInput)
		deleteNodePoolFutures = append(deleteNodePoolFutures, deleteNodePoolFuture)
	}

	// wait for AutoScalingGroups to be deleted
	err := waitForActivities(deleteNodePoolFutures, ctx, input.ClusterID)
	if err != nil {
		return err
	}

	createNodePoolFutures := make([]workflow.Future, 0, len(input.NewNodePools))
	for _, newNodePool := range input.NewNodePools {
		log.Info("node pool will be created")

		activityInput := eksWorkflow.CreateNodePoolWorkflowInput{
			ClusterID:                    input.ClusterID,
			CreatorUserID:                input.UpdaterUserID,
			NodePool:                     newNodePool,
			NodePoolSubnetIDs:            input.NewNodePoolSubnetIDs[newNodePool.Name],
			ShouldCreateNodePoolLabelSet: false, // TODO: update when UpdateNodePoolWorkflow is refactored.
			ShouldStoreNodePool:          true,
			ShouldUpdateClusterStatus:    false,
		}
		ctx = workflow.WithActivityOptions(ctx, aoWithHeartBeat)

		createNodePoolFuture := workflow.ExecuteChildWorkflow(
			ctx,
			eksWorkflow.CreateNodePoolWorkflowName,
			activityInput,
		)
		createNodePoolFutures = append(createNodePoolFutures, createNodePoolFuture)
	}

	nodePoolsToUpdate := make(map[string]eksWorkflow.AutoscaleGroup, len(input.UpdatedNodePools))
	updateNodePoolFutures := make([]workflow.Future, 0, len(input.UpdatedNodePools))
	for _, updatedNodePool := range input.UpdatedNodePools {
		log := logger.With("nodePool", updatedNodePool.Name)

		if !updatedNodePool.Create && !updatedNodePool.Delete {
			// update nodePool
			log.Info("node pool will be updated")
			nodePoolsToUpdate[updatedNodePool.Name] = updatedNodePool

			effectiveImage := updatedNodePool.NodeImage
			effectiveVolumeSize := updatedNodePool.NodeVolumeSize
			if effectiveImage == "" ||
				effectiveVolumeSize == 0 { // Note: needing CF stack for original information for version.
				getCFStackInput := eksWorkflow.GetCFStackActivityInput{
					EKSActivityInput: commonActivityInput,
					StackName:        eksWorkflow.GenerateNodePoolStackName(input.ClusterName, updatedNodePool.Name),
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
			if updatedNodePool.NodeVolumeSize > 0 {
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
						OptionalVolumeSize: updatedNodePool.NodeVolumeSize,
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
				StackName:        eksWorkflow.GenerateNodePoolStackName(input.ClusterName, updatedNodePool.Name),
				ScaleEnabled:     input.ScaleEnabled,
				Name:             updatedNodePool.Name,
				Version:          nodePoolVersion,
				NodeSpotPrice:    updatedNodePool.NodeSpotPrice,
				Autoscaling:      updatedNodePool.Autoscaling,
				NodeMinCount:     updatedNodePool.NodeMinCount,
				NodeMaxCount:     updatedNodePool.NodeMaxCount,
				Count:            updatedNodePool.Count,
				NodeVolumeSize:   volumeSize,
				NodeImage:        updatedNodePool.NodeImage,
				NodeInstanceType: updatedNodePool.NodeInstanceType,
				Labels:           updatedNodePool.Labels,
				Tags:             input.Tags,
			}
			ctx = workflow.WithActivityOptions(ctx, aoWithHeartBeat)
			f := workflow.ExecuteActivity(ctx, eksWorkflow.UpdateAsgActivityName, activityInput)
			updateNodePoolFutures = append(updateNodePoolFutures, f)
		}
	}

	// wait for AutoScalingGroups to be created & updated
	err = waitForActivities(append(createNodePoolFutures, updateNodePoolFutures...), ctx, input.ClusterID)
	if err != nil {
		return err
	}

	// Update node pools
	{
		// Note: created and deleted  node pools are saved earlier to the
		// database to be able to set the stack ID at creation and because the
		// new node pool workflows are designed to do the complete processes.
		nodePoolsToKeep := make(map[string]bool, len(input.NewNodePools))
		for _, newNodePool := range input.NewNodePools {
			nodePoolsToKeep[newNodePool.Name] = true
		}

		activityInput := eksWorkflow.SaveNodePoolsActivityInput{
			ClusterID:         input.ClusterID,
			NodePoolsToCreate: nil,
			NodePoolsToUpdate: nodePoolsToUpdate,
			NodePoolsToDelete: nil,
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
