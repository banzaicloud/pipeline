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
	"sort"
	"strconv"
	"strings"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	eksWorkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
	sdkCloudFormation "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon/cloudformation"
	"github.com/banzaicloud/pipeline/pkg/sdk/semver"
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
		OrganizationID: input.OrganizationID,
		SecretID:       input.SecretID,
		Region:         input.Region,
		ClusterName:    input.ClusterName,
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

	var createNodePoolsFuture workflow.Future
	{ // Note: creating new node pools.
		newNodePools := make(map[string]eks.NewNodePool, len(input.NewNodePools))
		for _, newNodePool := range input.NewNodePools {
			newNodePools[newNodePool.Name] = newNodePool
		}

		log.Info("node pools will be created")

		activityInput := eksWorkflow.CreateNodePoolsWorkflowInput{
			ClusterID:                    input.ClusterID,
			CreatorUserID:                input.UpdaterUserID,
			NodePools:                    newNodePools,
			NodePoolSubnetIDs:            input.NewNodePoolSubnetIDs,
			ShouldCreateNodePoolLabelSet: false, // Note: node pool labels are updated in this workflow. // TODO: update when UpdateNodePoolWorkflow is refactored.
			ShouldStoreNodePool:          true,  // Note: using the CreateNodePoolWorkflow's store logic instead of the persistence logic here.
			ShouldUpdateClusterStatus:    false, // Note: the cluster update workflow sets the cluster status in waitForActivities().
		}
		ctx = workflow.WithActivityOptions(ctx, aoWithHeartBeat)

		createNodePoolsFuture = workflow.ExecuteChildWorkflow(
			ctx,
			eksWorkflow.CreateNodePoolsWorkflowName,
			activityInput,
		)
	}

	nodePoolsToUpdate := make(map[string]eksWorkflow.AutoscaleGroup, len(input.UpdatedNodePools))
	updateNodePoolFutures := make([]workflow.Future, 0, len(input.UpdatedNodePools))
	for _, updatedNodePool := range input.UpdatedNodePools {
		log := logger.With("nodePool", updatedNodePool.Name)

		if !updatedNodePool.Create && !updatedNodePool.Delete {
			// update nodePool
			log.Info("node pool will be updated")
			nodePoolsToUpdate[updatedNodePool.Name] = updatedNodePool

			var currentTemplateVersion semver.Version
			effectiveImage := updatedNodePool.NodeImage

			effectiveVolumes := updatedNodePool.Volumes
			if effectiveVolumes == nil {
				effectiveVolumes = &eks.NodePoolVolumes{}
			}
			if effectiveVolumes.InstanceRoot == nil {
				effectiveVolumes.InstanceRoot = &eks.NodePoolVolume{}
			}
			if effectiveVolumes.KubeletRoot == nil {
				effectiveVolumes.KubeletRoot = &eks.NodePoolVolume{}
			}

			effectiveSecurityGroups := updatedNodePool.SecurityGroups
			{ // Note: needing CF stack for template version and possibly node pool version.
				getCFStackInput := eksWorkflow.GetCFStackActivityInput{
					EKSActivityInput: commonActivityInput,
					StackName:        eksWorkflow.GenerateNodePoolStackName(input.ClusterName, updatedNodePool.Name),
				}
				var getCFStackOutput eksWorkflow.GetCFStackActivityOutput
				err = workflow.ExecuteActivity(ctx, eksWorkflow.GetCFStackActivityName, getCFStackInput).Get(ctx, &getCFStackOutput)
				if err != nil {
					return err
				}

				var parameters struct {
					CustomNodeSecurityGroups           string         `mapstructure:"CustomNodeSecurityGroups,omitempty"` // Note: CustomNodeSecurityGroups is only available from template version 2.0.0.
					NodeImageID                        string         `mapstructure:"NodeImageId"`
					NodeVolumeStorage                  string         `mapstructure:"NodeVolumeStorage,omitempty"`           // Note: NodeVolumeStorage is only available from template version 2.5.0.
					NodeVolumeEncryptionEnabled        string         `mapstructure:"NodeVolumeEncryptionEnabled,omitempty"` // Note: NodeVolumeEncryptionEnabled is only available from template version 2.1.0.
					NodeVolumeEncryptionKeyARN         string         `mapstructure:"NodeVolumeEncryptionKeyARN,omitempty"`  // Note: NodeVolumeEncryptionKeyARN is only available from template version 2.1.0.
					NodeVolumeSize                     int            `mapstructure:"NodeVolumeSize"`
					KubeletRootVolumeStorage           string         `mapstructure:"KubeletRootVolumeStorage,omitempty"`           // Note: KubeletRootVolumeStorage is only available from template version 2.5.0.
					KubeletRootVolumeEncryptionEnabled string         `mapstructure:"KubeletRootVolumeEncryptionEnabled,omitempty"` // Note: KubeletRootVolumeEncryptionEnabled is only available from template version 2.5.0.
					KubeletRootVolumeEncryptionKeyARN  string         `mapstructure:"KubeletRootVolumeEncryptionKeyARN,omitempty"`  // Note: KubeletRootVolumeEncryptionKeyARN is only available from template version 2.5.0.
					KubeletRootVolumeSize              int            `mapstructure:"KubeletRootVolumeSize,omitempty"`              // Note: KubeletRootVolumeSize is only available from template version 2.5.0.
					TemplateVersion                    semver.Version `mapstructure:"TemplateVersion,omitempty"`                    // Note: TemplateVersion is only available from template version 2.0.0.
				}
				err = sdkCloudFormation.ParseStackParameters(getCFStackOutput.Stack.Parameters, &parameters)
				if err != nil {
					eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
					return err
				}

				currentTemplateVersion = parameters.TemplateVersion

				if effectiveImage == "" {
					effectiveImage = parameters.NodeImageID
				}

				if effectiveSecurityGroups == nil &&
					parameters.CustomNodeSecurityGroups != "" {
					effectiveSecurityGroups = strings.Split(parameters.CustomNodeSecurityGroups, ",")
					sort.Strings(effectiveSecurityGroups)
				}

				if effectiveVolumes.InstanceRoot.Storage == "" {
					effectiveVolumes.InstanceRoot.Storage = parameters.NodeVolumeStorage
					// set default ebs value for InstanceRoot.Storage for old templates
					if currentTemplateVersion.IsLessThan("2.5.0") {
						effectiveVolumes.InstanceRoot.Storage = eks.EBS_STORAGE
					}
				}
				// load EBS volume related params only in case storage == ebs
				if eks.EBS_STORAGE == effectiveVolumes.InstanceRoot.Storage {
					if effectiveVolumes.InstanceRoot.Encryption == nil &&
						parameters.NodeVolumeEncryptionEnabled != "" {
						isNodeVolumeEncryptionEnabled, err := strconv.ParseBool(parameters.NodeVolumeEncryptionEnabled)
						if err != nil {
							return errors.WrapIf(err, "invalid node volume encryption enabled value")
						}

						effectiveVolumes.InstanceRoot.Encryption = &eks.NodePoolVolumeEncryption{
							Enabled:          isNodeVolumeEncryptionEnabled,
							EncryptionKeyARN: parameters.NodeVolumeEncryptionKeyARN,
						}
					}

					if effectiveVolumes.InstanceRoot.Size == 0 {
						effectiveVolumes.InstanceRoot.Size = parameters.NodeVolumeSize
					}
					if effectiveVolumes.InstanceRoot.Type == "" {
						effectiveVolumes.InstanceRoot.Type = "gp3"
					}
				} else if eks.INSTANCE_STORE_STORAGE == effectiveVolumes.InstanceRoot.Storage {
					effectiveVolumes.InstanceRoot.Encryption = nil
					// can not be set to empty string
					effectiveVolumes.InstanceRoot.Type = "gp3"
					effectiveVolumes.InstanceRoot.Size = 0
				}

				if effectiveVolumes.KubeletRoot.Storage == "" {
					// set default none value for KubeletRoot.Storage for old templates
					if currentTemplateVersion.IsLessThan("2.5.0") {
						effectiveVolumes.KubeletRoot.Storage = eks.NONE_STORAGE
					} else {
						effectiveVolumes.KubeletRoot.Storage = parameters.KubeletRootVolumeStorage
					}
				}
				// load EBS volume related params only in case storage == ebs
				if eks.EBS_STORAGE == effectiveVolumes.KubeletRoot.Storage {
					if effectiveVolumes.KubeletRoot.Encryption == nil &&
						parameters.KubeletRootVolumeEncryptionEnabled != "" {
						isVolumeEncryptionEnabled, err := strconv.ParseBool(parameters.KubeletRootVolumeEncryptionEnabled)
						if err != nil {
							return errors.WrapIf(err, "invalid kubelet root volume encryption enabled value")
						}

						effectiveVolumes.KubeletRoot.Encryption = &eks.NodePoolVolumeEncryption{
							Enabled:          isVolumeEncryptionEnabled,
							EncryptionKeyARN: parameters.KubeletRootVolumeEncryptionKeyARN,
						}
					}

					if effectiveVolumes.KubeletRoot.Size == 0 {
						effectiveVolumes.KubeletRoot.Size = parameters.KubeletRootVolumeSize
					}
					// set default size in case it's still 0
					if effectiveVolumes.KubeletRoot.Size == 0 {
						effectiveVolumes.KubeletRoot.Size = 50
					}
					if effectiveVolumes.KubeletRoot.Type == "" {
						effectiveVolumes.KubeletRoot.Type = "gp3"
					}
				} else if eks.INSTANCE_STORE_STORAGE == effectiveVolumes.KubeletRoot.Storage ||
					eks.NONE_STORAGE == effectiveVolumes.KubeletRoot.Storage {
					effectiveVolumes.KubeletRoot.Encryption = nil
					effectiveVolumes.KubeletRoot.Type = ""
					effectiveVolumes.KubeletRoot.Size = 0
				}
			}

			var volumeSize int
			if updatedNodePool.Volumes != nil && updatedNodePool.Volumes.InstanceRoot != nil && updatedNodePool.Volumes.InstanceRoot.Size > 0 {
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
						OptionalVolumeSize: updatedNodePool.Volumes.InstanceRoot.Size,
					}
					var activityOutput eksWorkflow.SelectVolumeSizeActivityOutput
					err = workflow.ExecuteActivity(ctx, eksWorkflow.SelectVolumeSizeActivityName, activityInput).Get(ctx, &activityOutput)
					if err != nil {
						eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
						return err
					}

					volumeSize = activityOutput.VolumeSize
					effectiveVolumes.InstanceRoot.Size = volumeSize
				}
			}

			var nodePoolVersion string
			{
				activityInput := eksWorkflow.CalculateNodePoolVersionActivityInput{
					Image:                effectiveImage,
					Volumes:              *effectiveVolumes,
					CustomSecurityGroups: effectiveSecurityGroups,
				}

				activityOptions := ao
				activityOptions.StartToCloseTimeout = 30 * time.Second
				activityOptions.RetryPolicy = &cadence.RetryPolicy{
					InitialInterval:    10 * time.Second,
					BackoffCoefficient: 1.01,
					MaximumAttempts:    10,
					MaximumInterval:    10 * time.Minute,
				}

				var output eksWorkflow.CalculateNodePoolVersionActivityOutput

				err = workflow.ExecuteActivity(
					workflow.WithActivityOptions(ctx, activityOptions),
					eksWorkflow.CalculateNodePoolVersionActivityName,
					activityInput,
				).Get(ctx, &output)
				if err != nil {
					eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Warning, pkgCadence.UnwrapError(err).Error()) // nolint: errcheck
					return err
				}

				nodePoolVersion = output.Version
			}

			activityInput := eksWorkflow.UpdateAsgActivityInput{
				EKSActivityInput:       commonActivityInput,
				StackName:              eksWorkflow.GenerateNodePoolStackName(input.ClusterName, updatedNodePool.Name),
				Name:                   updatedNodePool.Name,
				Version:                nodePoolVersion,
				NodeSpotPrice:          updatedNodePool.NodeSpotPrice,
				Autoscaling:            updatedNodePool.Autoscaling,
				NodeMinCount:           updatedNodePool.NodeMinCount,
				NodeMaxCount:           updatedNodePool.NodeMaxCount,
				Count:                  updatedNodePool.Count,
				NodeVolumes:            *effectiveVolumes,
				NodeImage:              updatedNodePool.NodeImage,
				NodeInstanceType:       updatedNodePool.NodeInstanceType,
				SecurityGroups:         updatedNodePool.SecurityGroups,
				Labels:                 updatedNodePool.Labels,
				Tags:                   input.Tags,
				CurrentTemplateVersion: currentTemplateVersion,
			}
			ctx = workflow.WithActivityOptions(ctx, aoWithHeartBeat)
			f := workflow.ExecuteActivity(ctx, eksWorkflow.UpdateAsgActivityName, activityInput)
			updateNodePoolFutures = append(updateNodePoolFutures, f)
		}
	}

	// wait for AutoScalingGroups to be created & updated
	err = waitForActivities(append([]workflow.Future{createNodePoolsFuture}, updateNodePoolFutures...), ctx, input.ClusterID)
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

	_ = eksWorkflow.SetClusterStatus(ctx, input.ClusterID, pkgCluster.Running, pkgCluster.RunningMessage)
	return nil
}
