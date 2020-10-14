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

package awscommonadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/awscommon"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/awscommon/awscommonmodel"
)

type nodePoolStore struct {
	db *gorm.DB
}

// NewNodePoolStore returns a new AWSCommon.NodePoolStore
// that provides an interface to AWS node pool persistence.
func NewNodePoolStore(db *gorm.DB) awscommon.NodePoolStore {
	return nodePoolStore{
		db: db,
	}
}

func (s nodePoolStore) CreateNodePool(
	_ context.Context,
	clusterID uint,
	createdBy uint,
	nodePool awscommon.NewNodePool,
) error {
	nodePoolModel := &awscommonmodel.AmazonNodePoolsModel{
		ClusterID:        clusterID,
		CreatedBy:        createdBy,
		Name:             nodePool.Name,
		StackID:          "",
		NodeInstanceType: nodePool.InstanceType,
		NodeImage:        nodePool.Image,
		NodeSpotPrice:    nodePool.SpotPrice,
		Autoscaling:      nodePool.Autoscaling.Enabled,
		NodeMinCount:     nodePool.Autoscaling.MinSize,
		NodeMaxCount:     nodePool.Autoscaling.MaxSize,
		Count:            nodePool.Size,
		Status:           awscommon.NodePoolStatusCreating,
		StatusMessage:    "",
		// NodeVolumeSize:   nodePool.VolumeSize, // Note: not stored in DB.
		// Labels:           nodePool.Labels, // Note: not stored in DB.
	}

	err := s.db.Save(nodePoolModel).Error
	if err != nil {
		return errors.Wrap(err, "failed to save node pool")
	}

	return nil
}

// UpdateNodePoolStackID sets the stack ID in the node pool storage to the
// specified value.
func (s nodePoolStore) UpdateNodePoolStackID(
	ctx context.Context,
	organizationID uint,
	clusterID uint,
	clusterName string,
	nodePoolName string,
	nodePoolStackID string,
) (err error) {
	var pkeAwsCluster awscommonmodel.AWSCommonClusterModel
	err = s.db.
		Where(awscommonmodel.AWSCommonClusterModel{ClusterID: clusterID}).
		Preload("NodePools").
		First(&pkeAwsCluster).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return cluster.NotFoundError{
				OrganizationID: organizationID,
				ClusterID:      clusterID,
				ClusterName:    clusterName,
			}
		}

		return errors.WrapWithDetails(err, "fetching cluster from database failed",
			"organizationId", organizationID,
			"clusterId", clusterID,
			"clusterName", clusterName,
			"nodePool", nodePoolName,
		)
	}

	var nodePoolModel *awscommonmodel.AmazonNodePoolsModel
	for _, clusterNodePoolModel := range pkeAwsCluster.NodePools {
		if nodePoolName == clusterNodePoolModel.Name {
			nodePoolModel = clusterNodePoolModel
			break
		}
	}
	if nodePoolModel == nil {
		return cluster.NodePoolNotFoundError{
			ClusterID: clusterID,
			NodePool:  nodePoolName,
		}
	}

	nodePoolModel.StackID = nodePoolStackID

	if nodePoolStackID != "" { // Note: using CF stack status from now on as long as it exists.
		nodePoolModel.Status = awscommon.NodePoolStatusEmpty
		nodePoolModel.StatusMessage = ""
	}

	err = s.db.Save(nodePoolModel).Error
	if err != nil {
		return errors.WrapWithDetails(err, "updating node pool in database failed",
			"organizationId", organizationID,
			"clusterId", clusterID,
			"clusterName", clusterName,
			"nodePoolName", nodePoolName,
		)
	}

	return nil
}

// UpdateNodePoolStackID sets the status and status message in the node pool
// storage to the specified value.
func (s nodePoolStore) UpdateNodePoolStatus(
	ctx context.Context,
	organizationID uint,
	clusterID uint,
	clusterName string,
	nodePoolName string,
	nodePoolStatus awscommon.NodePoolStatus,
	nodePoolStatusMessage string,
) (err error) {
	var pkeCluster awscommonmodel.AWSCommonClusterModel
	err = s.db.
		Where(awscommonmodel.AWSCommonClusterModel{ClusterID: clusterID}).
		Preload("NodePools").
		First(&pkeCluster).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return cluster.NotFoundError{
				OrganizationID: organizationID,
				ClusterID:      clusterID,
				ClusterName:    clusterName,
			}
		}

		return errors.WrapWithDetails(err, "fetching cluster from database failed",
			"organizationId", organizationID,
			"clusterId", clusterID,
			"clusterName", clusterName,
			"nodePoolName", nodePoolName,
		)
	}

	var nodePoolModel *awscommonmodel.AmazonNodePoolsModel
	for _, clusterNodePoolModel := range pkeCluster.NodePools {
		if nodePoolName == clusterNodePoolModel.Name {
			nodePoolModel = clusterNodePoolModel
			break
		}
	}
	if nodePoolModel == nil {
		return cluster.NodePoolNotFoundError{
			ClusterID: clusterID,
			NodePool:  nodePoolName,
		}
	}

	nodePoolModel.Status = nodePoolStatus
	nodePoolModel.StatusMessage = nodePoolStatusMessage

	err = s.db.Save(nodePoolModel).Error
	if err != nil {
		return errors.WrapWithDetails(err, "updating node pool in database failed",
			"organizationId", organizationID,
			"clusterId", clusterID,
			"clusterName", clusterName,
			"nodePoolName", nodePoolName,
		)
	}

	return nil
}

// ListNodePools retrieves the node pools for the cluster specified by its
// cluster ID.
func (s nodePoolStore) ListNodePools(
	ctx context.Context,
	organizationID uint,
	clusterID uint,
	clusterName string,
) (existingNodePools map[string]awscommon.ExistingNodePool, err error) {
	var pkeCluster awscommonmodel.AWSCommonClusterModel
	err = s.db.
		Where(awscommonmodel.AWSCommonClusterModel{ClusterID: clusterID}).
		Preload("NodePools").
		First(&pkeCluster).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, cluster.NotFoundError{
				OrganizationID: organizationID,
				ClusterID:      clusterID,
				ClusterName:    clusterName,
			}
		}

		return nil, errors.WrapWithDetails(err, "fetching node pools from database failed",
			"organizationId", organizationID,
			"clusterId", clusterID,
			"clusterName", clusterName,
		)
	}

	existingNodePools = make(map[string]awscommon.ExistingNodePool, len(pkeCluster.NodePools))
	for _, nodePoolModel := range pkeCluster.NodePools {
		existingNodePools[nodePoolModel.Name] = awscommon.ExistingNodePool{
			Name:          nodePoolModel.Name,
			StackID:       nodePoolModel.StackID,
			Status:        nodePoolModel.Status,
			StatusMessage: nodePoolModel.StatusMessage,
		}
	}

	return existingNodePools, nil
}
