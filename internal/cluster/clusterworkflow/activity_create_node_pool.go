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

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/model"
)

const CreateNodePoolActivityName = "create-node-pool"

type CreateNodePoolActivity struct {
	clusters                 cluster.Store
	db                       *gorm.DB
	nodePools                cluster.NodePoolStore
	eksNodePools             eks.NodePoolStore
	awsSessionFactory        AWSSessionFactory
	cloudFormationAPIFactory eksworkflow.CloudFormationAPIFactory
}

// NewCreateNodePoolActivity returns a new CreateNodePoolActivity.
func NewCreateNodePoolActivity(
	clusters cluster.Store,
	db *gorm.DB,
	nodePools cluster.NodePoolStore,
	eksNodePools eks.NodePoolStore,
	awsSessionFactory AWSSessionFactory,
	cloudFormationAPIFactory eksworkflow.CloudFormationAPIFactory,
) CreateNodePoolActivity {
	return CreateNodePoolActivity{
		clusters:                 clusters,
		db:                       db,
		nodePools:                nodePools,
		eksNodePools:             eksNodePools,
		awsSessionFactory:        awsSessionFactory,
		cloudFormationAPIFactory: cloudFormationAPIFactory,
	}
}

type CreateNodePoolActivityInput struct {
	ClusterID   uint
	UserID      uint
	RawNodePool cluster.NewRawNodePool
}

func (a CreateNodePoolActivity) Execute(ctx context.Context, input CreateNodePoolActivityInput) error {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return cadence.WrapClientError(err)
	}

	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		var nodePool eks.NewNodePool

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

		var eksCluster eksmodel.EKSClusterModel

		err = a.db.
			Where(eksmodel.EKSClusterModel{ClusterID: c.ID}).
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

		commonActivityInput := eksworkflow.EKSActivityInput{
			OrganizationID:            c.OrganizationID,
			SecretID:                  c.SecretID.ResourceID, // TODO: the underlying secret store is the legacy one
			Region:                    c.Location,
			ClusterName:               c.Name,
			AWSClientRequestTokenBase: c.UID,
		}

		var vpcActivityOutput *eksworkflow.GetVpcConfigActivityOutput
		{
			input := eksworkflow.GetVpcConfigActivityInput{
				EKSActivityInput: commonActivityInput,
				StackName:        eksworkflow.GenerateStackNameForCluster(c.Name),
			}

			var err error

			vpcActivityOutput, err = eksworkflow.NewGetVpcConfigActivity(
				a.awsSessionFactory,
				a.cloudFormationAPIFactory,
			).Execute(ctx, input)
			if err != nil {
				return err
			}
		}

		subinput := eksworkflow.CreateAsgActivityInput{
			EKSActivityInput: commonActivityInput,
			StackName:        eksworkflow.GenerateNodePoolStackName(c.Name, nodePool.Name),

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
			NodeImage:        nodePool.Image,
			NodeInstanceType: nodePool.InstanceType,
			Labels:           nodePool.Labels,
			Tags:             eksCluster.Cluster.Tags,
		}

		eksConfig := global.Config.Distribution.EKS
		if eksConfig.SSH.Generate {
			subinput.SSHKeyName = eksworkflow.GenerateSSHKeyNameForCluster(c.Name)
		}

		nodePoolTemplate, err := eksworkflow.GetNodePoolTemplate()
		if err != nil {
			return errors.WrapIf(err, "failed to get CloudFormation template for node pools")
		}

		_, err = eksworkflow.NewCreateAsgActivity(a.awsSessionFactory, a.cloudFormationAPIFactory, nodePoolTemplate).Execute(ctx, subinput)
		if err != nil {
			return cadence.WrapClientError(err)
		}

		err = a.eksNodePools.CreateNodePool(ctx, eksCluster.ID, input.UserID, nodePool)
		if err != nil {
			return err
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
