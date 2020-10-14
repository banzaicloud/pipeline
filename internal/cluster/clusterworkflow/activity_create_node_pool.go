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

package clusterworkflow

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/awscommon"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/awscommon/awscommonmodel"
	awscommonworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/awscommon/awscommonproviders/workflow"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/providers"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
	"github.com/banzaicloud/pipeline/src/model"
)

const CreateNodePoolActivityName = "create-node-pool"

type CreateNodePoolActivity struct {
	clusters          cluster.Store
	db                *gorm.DB
	defaultVolumeSize int
	nodePools         cluster.NodePoolStore
	eksNodePools      awscommon.NodePoolStore
	awsSessionFactory AWSSessionFactory
}

// NewCreateNodePoolActivity returns a new CreateNodePoolActivity.
func NewCreateNodePoolActivity(
	clusters cluster.Store,
	db *gorm.DB,
	defaultVolumeSize int,
	nodePools cluster.NodePoolStore,
	eksNodePools awscommon.NodePoolStore,
	awsSessionFactory AWSSessionFactory,
) CreateNodePoolActivity {
	return CreateNodePoolActivity{
		clusters:          clusters,
		db:                db,
		defaultVolumeSize: defaultVolumeSize,
		nodePools:         nodePools,
		eksNodePools:      eksNodePools,
		awsSessionFactory: awsSessionFactory,
	}
}

type CreateNodePoolActivityInput struct {
	ClusterID   uint
	UserID      uint
	RawNodePool cluster.NewRawNodePool
}

func (a CreateNodePoolActivity) Execute(ctx context.Context, input CreateNodePoolActivityInput) error {
	activityInformation := activity.GetInfo(ctx)

	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return cadence.WrapClientError(err)
	}

	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		var nodePool awscommon.NewNodePool

		err := mapstructure.Decode(input.RawNodePool, &nodePool)
		if err != nil {
			return errors.Wrap(err, "failed to decode node pool")
		}

		var commonCluster model.ClusterModel

		err = a.db.
			Where(model.ClusterModel{ID: c.ID}).
			First(&commonCluster).Error
		if gorm.IsRecordNotFoundError(err) {
			return cadence.NewClientError(errors.NewWithDetails(
				"cluster model is inconsistent",
				"clusterId", c.ID,
			))
		}
		if err != nil {
			return errors.WrapWithDetails(
				err, "failed to get cluster info",
				"clusterId", c.ID,
				"nodePoolName", nodePool.Name,
			)
		}

		var eksCluster awscommonmodel.AWSCommonClusterModel

		err = a.db.
			Where(awscommonmodel.AWSCommonClusterModel{ClusterID: c.ID}).
			Preload("Subnets").
			Preload("Cluster").
			First(&eksCluster).Error
		if gorm.IsRecordNotFoundError(err) {
			return cadence.NewClientError(errors.NewWithDetails(
				"cluster model is inconsistent",
				"clusterId", c.ID,
			))
		}
		if err != nil {
			return errors.WrapWithDetails(
				err, "failed to get cluster info",
				"clusterId", c.ID,
				"nodePoolName", nodePool.Name,
			)
		}

		minSize := nodePool.Size
		maxSize := nodePool.Size + 1

		if nodePool.Autoscaling.Enabled {
			minSize = nodePool.Autoscaling.MinSize
			maxSize = nodePool.Autoscaling.MaxSize
		}

		commonActivityInput := awscommonworkflow.AWSCommonActivityInput{
			OrganizationID:            c.OrganizationID,
			SecretID:                  c.SecretID.ResourceID, // TODO: the underlying secret store is the legacy one
			Region:                    c.Location,
			ClusterName:               c.Name,
			AWSClientRequestTokenBase: sdkAmazon.NewNormalizedClientRequestToken(activity.GetInfo(ctx).WorkflowExecution.ID),
		}

		if activityInformation.Attempt == 0 {
			err = a.eksNodePools.CreateNodePool(ctx, eksCluster.ID, input.UserID, nodePool)
			if err != nil {
				return err
			}
		}

		var vpcActivityOutput *eksworkflow.GetVpcConfigActivityOutput
		{
			activityInput := eksworkflow.GetVpcConfigActivityInput{
				AWSCommonActivityInput: commonActivityInput,
				StackName:              awscommonworkflow.GenerateStackNameForCluster(c.Name),
			}

			var err error

			vpcActivityOutput, err = eksworkflow.NewGetVpcConfigActivity(a.awsSessionFactory).Execute(ctx, activityInput)
			if err != nil {
				return err
			}
		}

		var amiSize int
		{
			activityOutput, err := eksworkflow.NewGetAMISizeActivity(
				a.awsSessionFactory,
				eksworkflow.NewEC2Factory(),
			).Execute(
				ctx,
				eksworkflow.GetAMISizeActivityInput{
					AWSCommonActivityInput: commonActivityInput,
					ImageID:                nodePool.Image,
				},
			)
			if err != nil {
				_ = a.eksNodePools.UpdateNodePoolStatus(
					ctx,
					c.OrganizationID,
					c.ID,
					c.Name,
					nodePool.Name,
					awscommon.NodePoolStatusError,
					fmt.Sprintf("Validation failed: retrieving AMI size failed: %s", err),
				)

				return err
			}

			amiSize = activityOutput.AMISize
		}

		var volumeSize int
		{
			activityOutput, err := eksworkflow.NewSelectVolumeSizeActivity(
				a.defaultVolumeSize,
			).Execute(
				ctx,
				eksworkflow.SelectVolumeSizeActivityInput{
					AMISize:            amiSize,
					OptionalVolumeSize: nodePool.VolumeSize,
				},
			)
			if err != nil {
				_ = a.eksNodePools.UpdateNodePoolStatus(
					ctx,
					c.OrganizationID,
					c.ID,
					c.Name,
					nodePool.Name,
					awscommon.NodePoolStatusError,
					fmt.Sprintf("Validation failed: selecting volume size failed: %s", err),
				)

				return err
			}

			volumeSize = activityOutput.VolumeSize
		}

		subinput := eksworkflow.CreateAsgActivityInput{
			AWSCommonActivityInput: commonActivityInput,
			ClusterID:              input.ClusterID,
			StackName:              awscommonworkflow.GenerateNodePoolStackName(c.Name, nodePool.Name),

			ScaleEnabled: commonCluster.ScaleOptions.Enabled,

			Subnets: []awscommonworkflow.Subnet{
				{
					SubnetID: nodePool.SubnetID,
				},
			},

			VpcID:               vpcActivityOutput.VpcID,
			SecurityGroupID:     vpcActivityOutput.SecurityGroupID,
			NodeSecurityGroupID: vpcActivityOutput.NodeSecurityGroupID,
			NodeInstanceRoleID:  eksCluster.NodeInstanceRoleId,

			Name:             nodePool.Name,
			NodeSpotPrice:    nodePool.SpotPrice,
			Autoscaling:      nodePool.Autoscaling.Enabled,
			NodeMinCount:     minSize,
			NodeMaxCount:     maxSize,
			Count:            nodePool.Size,
			NodeVolumeSize:   volumeSize,
			NodeImage:        nodePool.Image,
			NodeInstanceType: nodePool.InstanceType,
			Labels:           nodePool.Labels,
			Tags:             eksCluster.Cluster.Tags,
		}

		eksConfig := global.Config.Distribution.EKS
		if eksConfig.SSH.Generate {
			subinput.SSHKeyName = awscommonworkflow.GenerateSSHKeyNameForCluster(c.Name)
		}

		nodePoolTemplate, err := eksworkflow.GetNodePoolTemplate()
		if err != nil {
			return errors.WrapIf(err, "failed to get CloudFormation template for node pools")
		}

		_, err = eksworkflow.NewCreateAsgActivity(
			a.awsSessionFactory, nodePoolTemplate, a.eksNodePools,
		).Execute(ctx, subinput)
		if err != nil {
			return cadence.WrapClientError(err)
		}
	default:
		return cadence.WrapClientError(errors.WithStack(cluster.NotSupportedDistributionError{
			ID:           c.ID,
			Cloud:        c.Cloud,
			Distribution: c.Distribution,

			Message: "the node pool API does not support this distribution yet",
		}))
	}

	return nil
}
