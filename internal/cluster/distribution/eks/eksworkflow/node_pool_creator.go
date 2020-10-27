// Copyright © 2020 Banzai Cloud
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

package eksworkflow

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/pkg/cadence"
	sdkAmazon "github.com/banzaicloud/pipeline/pkg/sdk/providers/amazon"
	"github.com/banzaicloud/pipeline/src/model"
)

type LegacyAWSSessionFactory interface {
	New(organizationID uint, secretID string, region string) (*session.Session, error)
}

// NewNodePoolCreator returns a new CreateNodePoolActivity.
func NewNodePoolCreator(
	db *gorm.DB,
	defaultVolumeSize int,
	eksNodePools eks.NodePoolStore,
	awsSessionFactory LegacyAWSSessionFactory,
) clusterworkflow.NodePoolCreator {
	return eksNodePoolCreator{
		db:                db,
		defaultVolumeSize: defaultVolumeSize,
		eksNodePools:      eksNodePools,
		awsSessionFactory: awsSessionFactory,
	}
}

type eksNodePoolCreator struct {
	db                *gorm.DB
	defaultVolumeSize int
	eksNodePools      eks.NodePoolStore
	awsSessionFactory LegacyAWSSessionFactory
}

func (c eksNodePoolCreator) CreateNodePool(
	ctx context.Context,
	userID uint,
	cl cluster.Cluster,
	rawNodePool cluster.NewRawNodePool,
) error {
	var nodePool eks.NewNodePool

	err := mapstructure.Decode(rawNodePool, &nodePool)
	if err != nil {
		return errors.Wrap(err, "failed to decode node pool")
	}

	var commonCluster model.ClusterModel

	err = c.db.
		Where(model.ClusterModel{ID: cl.ID}).
		First(&commonCluster).Error
	if gorm.IsRecordNotFoundError(err) {
		return cadence.NewClientError(errors.NewWithDetails(
			"cluster model is inconsistent",
			"clusterId", cl.ID,
		))
	}
	if err != nil {
		return errors.WrapWithDetails(
			err, "failed to get cluster info",
			"clusterId", cl.ID,
			"nodePoolName", nodePool.Name,
		)
	}

	var eksCluster eksmodel.EKSClusterModel

	err = c.db.
		Where(eksmodel.EKSClusterModel{ClusterID: cl.ID}).
		Preload("Subnets").
		Preload("Cluster").
		First(&eksCluster).Error
	if gorm.IsRecordNotFoundError(err) {
		return cadence.NewClientError(errors.NewWithDetails(
			"cluster model is inconsistent",
			"clusterId", cl.ID,
		))
	}
	if err != nil {
		return errors.WrapWithDetails(
			err, "failed to get cluster info",
			"clusterId", cl.ID,
			"nodePoolName", nodePool.Name,
		)
	}

	minSize := nodePool.Size
	maxSize := nodePool.Size + 1

	if nodePool.Autoscaling.Enabled {
		minSize = nodePool.Autoscaling.MinSize
		maxSize = nodePool.Autoscaling.MaxSize
	}

	commonActivityInput := awsworkflow.AWSCommonActivityInput{
		OrganizationID:            cl.OrganizationID,
		SecretID:                  cl.SecretID.ResourceID, // TODO: the underlying secret store is the legacy one
		Region:                    cl.Location,
		ClusterName:               cl.Name,
		AWSClientRequestTokenBase: sdkAmazon.NewNormalizedClientRequestToken(activity.GetInfo(ctx).WorkflowExecution.ID),
	}

	activityInformation := activity.GetInfo(ctx)
	if activityInformation.Attempt == 0 {
		err = c.eksNodePools.CreateNodePool(ctx, eksCluster.ID, userID, nodePool)
		if err != nil {
			return err
		}
	}

	var vpcActivityOutput *eksworkflow.GetVpcConfigActivityOutput
	{
		activityInput := eksworkflow.GetVpcConfigActivityInput{
			AWSCommonActivityInput: commonActivityInput,
			StackName:              eksworkflow.GenerateStackNameForCluster(cl.Name),
		}

		var err error

		vpcActivityOutput, err = eksworkflow.NewGetVpcConfigActivity(c.awsSessionFactory).Execute(ctx, activityInput)
		if err != nil {
			return err
		}
	}

	var amiSize int
	{
		activityOutput, err := eksworkflow.NewGetAMISizeActivity(
			c.awsSessionFactory,
			eksworkflow.NewEC2Factory(),
		).Execute(
			ctx,
			eksworkflow.GetAMISizeActivityInput{
				AWSCommonActivityInput: commonActivityInput,
				ImageID:                nodePool.Image,
			},
		)
		if err != nil {
			_ = c.eksNodePools.UpdateNodePoolStatus(
				ctx,
				cl.OrganizationID,
				cl.ID,
				cl.Name,
				nodePool.Name,
				eks.NodePoolStatusError,
				fmt.Sprintf("Validation failed: retrieving AMI size failed: %s", err),
			)

			return err
		}

		amiSize = activityOutput.AMISize
	}

	var volumeSize int
	{
		activityOutput, err := eksworkflow.NewSelectVolumeSizeActivity(
			c.defaultVolumeSize,
		).Execute(
			ctx,
			eksworkflow.SelectVolumeSizeActivityInput{
				AMISize:            amiSize,
				OptionalVolumeSize: nodePool.VolumeSize,
			},
		)
		if err != nil {
			_ = c.eksNodePools.UpdateNodePoolStatus(
				ctx,
				cl.OrganizationID,
				cl.ID,
				cl.Name,
				nodePool.Name,
				eks.NodePoolStatusError,
				fmt.Sprintf("Validation failed: selecting volume size failed: %s", err),
			)

			return err
		}

		volumeSize = activityOutput.VolumeSize
	}

	subinput := eksworkflow.CreateAsgActivityInput{
		AWSCommonActivityInput: commonActivityInput,
		ClusterID:              cl.ID,
		StackName:              eksworkflow.GenerateNodePoolStackName(cl.Name, nodePool.Name),

		ScaleEnabled: commonCluster.ScaleOptions.Enabled,

		Subnets: []eksworkflow.Subnet{
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
		subinput.SSHKeyName = eksworkflow.GenerateSSHKeyNameForCluster(cl.Name)
	}

	nodePoolTemplate, err := eksworkflow.GetNodePoolTemplate()
	if err != nil {
		return errors.WrapIf(err, "failed to get CloudFormation template for node pools")
	}

	_, err = eksworkflow.NewCreateAsgActivity(
		c.awsSessionFactory, nodePoolTemplate, c.eksNodePools,
	).Execute(ctx, subinput)
	if err != nil {
		return cadence.WrapClientError(err)
	}

	return nil
}
