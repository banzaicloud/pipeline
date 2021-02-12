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
	"github.com/aws/aws-sdk-go/aws"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	pkgcadence "github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
	sdkcloudformation "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon/cloudformation"
)

// CreateNodePoolWorkflowName is the name of the EKS workflow creating a new
// node pool in a cluster.
const CreateNodePoolWorkflowName = "eks-create-node-pool"

// CreateNodePoolWorkflow defines a Cadence workflow encapsulating high level
// input-independent components required to create an EKS node pool.
type CreateNodePoolWorkflow struct{}

// CreateNodePoolWorkflowInput defines the input parameters of an EKS node pool
// creation.
type CreateNodePoolWorkflowInput struct {
	ClusterID         uint
	CreatorUserID     uint
	NodePool          eks.NewNodePool
	NodePoolSubnetIDs []string // Note: temporary while eks.NewNodePool has singular Subnet and ASH has plural.

	// Note: LegacyClusterAPI.CreateCluster installs and initializes the node
	// pool label set operator later, so the the node pool label set cannot be
	// created now. Once the installation happens, the currently available node
	// pools's label sets are created automatically before the cluster creation
	// finishes, so no additional operation is required aside from not creating
	// the node pool label set as part of the node pool creation.
	ShouldCreateNodePoolLabelSet bool

	// Note: LegacyClusterAPI.CreateCluster node pool creations store the entire
	// cluster descriptor object with the node pools included in the database at
	// a higher level, thus it should not be stored here, only checked, while
	// ClusterAPI.UpdateCluster and NodePoolAPI.CreateNodePool creations should
	// also store the node pool as it is not done previously.
	ShouldStoreNodePool bool

	// Note: LegacyClusterAPI.CreateCluster, ClusterAPI.UpdateCluster node pool
	// creations should not change the cluster status (CREATING/UPDATING),
	// because success could not yet mean RUNNING status and errors should be
	// handled by the higher level workflow, but NodePoolAPI.CreateNodePool node
	// pool creations should update the cluster status.
	ShouldUpdateClusterStatus bool
}

// NewCreateNodePoolWorkflow instantiates an EKS node pool creation workflow.
func NewCreateNodePoolWorkflow() *CreateNodePoolWorkflow {
	return &CreateNodePoolWorkflow{}
}

// Execute runs the workflow.
func (w CreateNodePoolWorkflow) Execute(ctx workflow.Context, input CreateNodePoolWorkflowInput) (err error) {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          15 * time.Second,
			BackoffCoefficient:       1.0,
			MaximumAttempts:          30,
			NonRetriableErrorReasons: []string{pkgcadence.ClientErrorReason, "cadenceInternal:Panic"},
		},
	}
	_ctx := ctx
	ctx = workflow.WithActivityOptions(ctx, ao)

	if input.ShouldUpdateClusterStatus {
		defer func() { // Note: update cluster status on error.
			if err != nil {
				_ = SetClusterStatus(_ctx, input.ClusterID, cluster.Warning, pkgcadence.UnwrapError(err).Error())
			}
		}()
	}

	eksClusters, err := listStoredEKSClusters(ctx, input.ClusterID)
	if err != nil {
		return err
	}
	eksCluster := eksClusters[input.ClusterID]

	eksActivityInput := EKSActivityInput{
		OrganizationID: eksCluster.Cluster.OrganizationID,
		SecretID:       eksCluster.Cluster.SecretID,
		Region:         eksCluster.Cluster.Location,
		ClusterName:    eksCluster.Cluster.Name,
	}

	if eksCluster.NodeInstanceRoleId == "" {
		// Note: in case store doesn't have the latest cluster state
		// (LegacyClusterAPI.CreateCluster with automatically created IAM roles).
		iamRoleStackName := generateStackNameForIam(eksCluster.Cluster.Name)
		var iamRoleOutputs struct {
			NodeInstanceRoleID string `mapstructure:"NodeInstanceRoleId"`
		}
		err = getCFStackOutputs(ctx, eksActivityInput, iamRoleStackName, &iamRoleOutputs)
		if err != nil {
			return err
		}

		eksCluster.NodeInstanceRoleId = iamRoleOutputs.NodeInstanceRoleID
	}

	if len(eksCluster.Subnets) == 0 ||
		eksCluster.Subnets[0].SubnetId == nil ||
		*eksCluster.Subnets[0].SubnetId == "" {
		// Note: in case store doesn't have the latest cluster state
		// (LegacyClusterAPI.CreateCluster with automatically created subnets).
		subnetStackNames, err := getClusterSubnetStackNames(ctx, eksActivityInput)
		if err != nil {
			return err
		}

		clusterSubnets := make([]*eksmodel.EKSSubnetModel, 0, len(eksCluster.Subnets))
		for subnetStackIndex, subnetStackName := range subnetStackNames {
			subnetStack, err := getCFStack(ctx, eksActivityInput, subnetStackName)
			if err != nil {
				return err
			}

			var subnetStackParameters struct {
				Cidr             string `mapstructure:"SubnetBlock"`
				AvailabilityZone string `mapstructure:"AvailabilityZoneName"`
			}
			err = sdkcloudformation.ParseStackParameters(subnetStack.Parameters, &subnetStackParameters)
			if err != nil {
				return errors.WrapWithDetails(
					err,
					"parsing subnet stack parameters failed",
					"stackName", subnetStackName,
					"parameters", subnetStack.Parameters,
				)
			}

			var subnetID string
			for _, subnetOutput := range subnetStack.Outputs {
				switch aws.StringValue(subnetOutput.OutputKey) {
				case "SubnetId":
					subnetID = aws.StringValue(subnetOutput.OutputValue)
				}
			}

			clusterSubnets = append(clusterSubnets, &eksmodel.EKSSubnetModel{
				ID:               uint(subnetStackIndex), // Note: not used.
				CreatedAt:        aws.TimeValue(subnetStack.CreationTime),
				EKSCluster:       eksCluster,
				ClusterID:        input.ClusterID,
				SubnetId:         aws.String(subnetID),
				Cidr:             aws.String(subnetStackParameters.Cidr),
				AvailabilityZone: aws.String(subnetStackParameters.AvailabilityZone),
			})
		}

		eksCluster.Subnets = clusterSubnets
	}

	if workflow.GetInfo(ctx).Attempt == 0 &&
		input.ShouldStoreNodePool {
		err = createStoredNodePool(
			ctx,
			eksCluster.Cluster.OrganizationID,
			input.ClusterID,
			eksCluster.Cluster.Name,
			input.CreatorUserID,
			input.NodePool,
		)
		if err != nil {
			return err
		}
	}

	defer func() { // Note: update the stored node pool status on error.
		if err != nil {
			_ = setNodePoolErrorStatus(
				ctx,
				eksCluster.Cluster.OrganizationID,
				input.ClusterID,
				eksCluster.Cluster.Name,
				input.NodePool.Name,
				err,
			)
		}
	}()

	amiSize, err := getAMISize(ctx, eksActivityInput, input.NodePool.Image)
	if err != nil {
		return err
	}

	volumeSize, err := selectVolumeSize(ctx, amiSize, input.NodePool.VolumeSize)
	if err != nil {
		return err
	}

	if input.ShouldCreateNodePoolLabelSet {
		err = createNodePoolLabelSetFromEKSNodePool(ctx, input.ClusterID, input.NodePool)
		if err != nil {
			return err
		}
	}

	vpcConfig, err := getVPCConfig(ctx, eksActivityInput, GenerateStackNameForCluster(eksCluster.Cluster.Name))
	if err != nil {
		return err
	}

	nodePoolVersion, err := calculateNodePoolVersion(
		ctx, input.NodePool.Image, input.NodePool.VolumeEncryption, input.NodePool.VolumeSize, input.NodePool.SecurityGroups)
	if err != nil {
		return err
	}

	err = createASG(
		ctx, eksActivityInput, eksCluster, vpcConfig, input.NodePool, input.NodePoolSubnetIDs, volumeSize, nodePoolVersion)
	if err != nil {
		return pkgcadence.WrapClientError(err)
	}

	if input.ShouldUpdateClusterStatus {
		err = SetClusterStatus(ctx, input.ClusterID, cluster.Running, cluster.RunningMessage)
		if err != nil {
			return err
		}
	}

	return nil
}

// Register registers the activity in the worker.
func (w CreateNodePoolWorkflow) Register(worker worker.Registry) {
	worker.RegisterWorkflowWithOptions(w.Execute, workflow.RegisterOptions{Name: CreateNodePoolWorkflowName})
}

// createNodePool creates a node pool.
//
// This is a convenience wrapper around the corresponding workflow.
func createNodePool(
	ctx workflow.Context,
	clusterID uint,
	creatorUserID uint,
	nodePool eks.NewNodePool,
	nodePoolSubnetIDs []string,
	shouldCreateNodePoolLabelSet bool,
	shouldStoreNodePool bool,
	shouldUpdateClusterStatus bool,
) error {
	return createNodePoolAsync(
		ctx,
		clusterID,
		creatorUserID,
		nodePool,
		nodePoolSubnetIDs,
		shouldCreateNodePoolLabelSet,
		shouldStoreNodePool,
		shouldUpdateClusterStatus,
	).Get(ctx, nil)
}

// createNodePoolAsync returns a future object for creating a node pool.
//
// This is a convenience wrapper around the corresponding workflow.
func createNodePoolAsync(
	ctx workflow.Context,
	clusterID uint,
	creatorUserID uint,
	nodePool eks.NewNodePool,
	nodePoolSubnetIDs []string,
	shouldCreateNodePoolLabelSet bool,
	shouldStoreNodePool bool,
	shouldUpdateClusterStatus bool,
) workflow.Future {
	return workflow.ExecuteChildWorkflow(ctx, CreateNodePoolWorkflowName, CreateNodePoolWorkflowInput{
		ClusterID:                    clusterID,
		NodePool:                     nodePool,
		NodePoolSubnetIDs:            nodePoolSubnetIDs,
		ShouldCreateNodePoolLabelSet: shouldCreateNodePoolLabelSet,
		ShouldStoreNodePool:          shouldStoreNodePool,
		ShouldUpdateClusterStatus:    shouldUpdateClusterStatus,
		CreatorUserID:                creatorUserID,
	})
}
