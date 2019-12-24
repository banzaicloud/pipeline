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
	"github.com/banzaicloud/pipeline/internal/cluster/distribution"
	eksworkflow "github.com/banzaicloud/pipeline/internal/providers/amazon/eks/workflow"
	"github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/model"
)

const CreateNodePoolActivityName = "create-node-pool"

type CreateNodePoolActivity struct {
	clusters          cluster.Store
	db                *gorm.DB
	nodePools         cluster.NodePoolStore
	eksNodePools      distribution.EKSNodePoolStore
	awsSessionFactory AWSSessionFactory
}

// NewCreateNodePoolActivity returns a new CreateNodePoolActivity.
func NewCreateNodePoolActivity(
	clusters cluster.Store,
	db *gorm.DB,
	nodePools cluster.NodePoolStore,
	eksNodePools distribution.EKSNodePoolStore,
	awsSessionFactory AWSSessionFactory,
) CreateNodePoolActivity {
	return CreateNodePoolActivity{
		clusters:          clusters,
		db:                db,
		nodePools:         nodePools,
		eksNodePools:      eksNodePools,
		awsSessionFactory: awsSessionFactory,
	}
}

type CreateNodePoolActivityInput struct {
	ClusterID   uint
	UserID      uint
	NodePool    cluster.NewNodePool
	RawNodePool cluster.NewRawNodePool
}

func (a CreateNodePoolActivity) Execute(ctx context.Context, input CreateNodePoolActivityInput) error {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return cadence.WrapClientError(err)
	}

	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		var nodePool distribution.NewEKSNodePool

		err := mapstructure.Decode(input.RawNodePool, &nodePool)
		if err != nil {
			return errors.Wrap(err, "failed to decode node pool")
		}

		var commonCluster model.ClusterModel

		err = a.db.
			Where(model.ClusterModel{ID: c.ID}).
			First(&commonCluster).Error
		if gorm.IsRecordNotFoundError(err) {
			return errors.NewWithDetails(
				"cluster model is inconsistent",
				"clusterId", c.ID,
			)
		}
		if err != nil {
			return errors.WrapWithDetails(
				err, "failed to get cluster info",
				"clusterId", c.ID,
				"nodePoolName", input.NodePool.Name,
			)
		}

		var eksCluster model.EKSClusterModel

		err = a.db.
			Where(model.EKSClusterModel{ClusterID: c.ID}).
			First(&eksCluster).Error
		if gorm.IsRecordNotFoundError(err) {
			return errors.NewWithDetails(
				"cluster model is inconsistent",
				"clusterId", c.ID,
			)
		}
		if err != nil {
			return errors.WrapWithDetails(
				err, "failed to get cluster info",
				"clusterId", c.ID,
				"nodePoolName", input.NodePool.Name,
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
			SecretID:                  c.SecretID.String(),
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

			vpcActivityOutput, err = eksworkflow.NewGetVpcConfigActivity(a.awsSessionFactory).Execute(ctx, input)
			if err != nil {
				return err
			}
		}

		var subnet eksworkflow.Subnet

		if nodePool.Subnet.SubnetId != "" {
			for _, s := range eksCluster.Subnets {
				if s.SubnetId != nil && *s.SubnetId == nodePool.Subnet.SubnetId {
					subnet.SubnetID = nodePool.Subnet.SubnetId
					subnet.Cidr = *s.Cidr
					subnet.AvailabilityZone = *s.AvailabilityZone
				}
			}
		} else if nodePool.Subnet.Cidr != "" && nodePool.Subnet.AvailabilityZone != "" {
			for _, s := range eksCluster.Subnets {
				if s.Cidr != nil && *s.Cidr == nodePool.Subnet.Cidr && s.AvailabilityZone != nil && *s.AvailabilityZone == nodePool.Subnet.AvailabilityZone {
					subnet.SubnetID = *s.SubnetId
					subnet.Cidr = nodePool.Subnet.Cidr
					subnet.AvailabilityZone = nodePool.Subnet.AvailabilityZone
				}
			}
		}

		subinput := eksworkflow.CreateAsgActivityInput{
			EKSActivityInput: eksworkflow.EKSActivityInput{
				OrganizationID:            c.OrganizationID,
				SecretID:                  c.SecretID.String(),
				Region:                    c.Location,
				ClusterName:               c.Name,
				AWSClientRequestTokenBase: c.UID,
			},
			StackName: eksworkflow.GenerateNodePoolStackName(c.Name, input.NodePool.Name),

			ScaleEnabled: commonCluster.ScaleOptions.Enabled,
			SSHKeyName:   eksworkflow.GenerateSSHKeyNameForCluster(c.Name),

			Subnets: []eksworkflow.Subnet{
				subnet,
			},

			VpcID:               vpcActivityOutput.VpcID,
			SecurityGroupID:     vpcActivityOutput.SecurityGroupID,
			NodeSecurityGroupID: vpcActivityOutput.NodeSecurityGroupID,
			NodeInstanceRoleID:  eksCluster.NodeInstanceRoleId,

			Name:             input.NodePool.Name,
			NodeSpotPrice:    nodePool.SpotPrice,
			Autoscaling:      nodePool.Autoscaling.Enabled,
			NodeMinCount:     minSize,
			NodeMaxCount:     maxSize,
			Count:            nodePool.Size,
			NodeImage:        nodePool.Image,
			NodeInstanceType: nodePool.InstanceType,
			Labels:           input.NodePool.Labels,
		}

		nodePoolTemplate, err := eksworkflow.GetNodePoolTemplate()
		if err != nil {
			return errors.WrapIf(err, "failed to get CloudFormation template for node pools")
		}

		_, err = eksworkflow.NewCreateAsgActivity(a.awsSessionFactory, nodePoolTemplate).Execute(ctx, subinput)
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
