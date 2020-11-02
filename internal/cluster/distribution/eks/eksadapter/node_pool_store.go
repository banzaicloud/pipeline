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

package eksadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
)

type nodePoolStore struct {
	db *gorm.DB
}

// NewNodePoolStore returns a new eks.NodePoolStore
// that provides an interface to EKS node pool persistence.
func NewNodePoolStore(db *gorm.DB) eks.NodePoolStore {
	return nodePoolStore{
		db: db,
	}
}

// CreateNodePool saves a new node pool.
//
// Implements the eks.NodePoolStore interface.
func (s nodePoolStore) CreateNodePool(
	_ context.Context,
	organizationID uint,
	clusterID uint,
	clusterName string,
	createdBy uint,
	nodePool eks.NewNodePool,
) (err error) {
	var eksCluster eksmodel.EKSClusterModel
	err = s.db.
		Where(eksmodel.EKSClusterModel{ClusterID: clusterID}).
		First(&eksCluster).Error
	if err != nil && gorm.IsRecordNotFoundError(err) {
		return cluster.NotFoundError{
			OrganizationID: organizationID,
			ClusterID:      clusterID,
			ClusterName:    clusterName,
		}
	} else if err != nil {
		return errors.WrapWithDetails(err, "fetching cluster from database failed",
			"organizationId", organizationID,
			"clusterId", clusterID,
			"clusterName", clusterName,
		)
	}

	nodePoolModel := &eksmodel.AmazonNodePoolsModel{
		ClusterID:        eksCluster.ID,
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
		Status:           eks.NodePoolStatusCreating,
		StatusMessage:    "",
		// NodeVolumeSize:   nodePool.VolumeSize, // Note: not stored in DB.
		// Labels:           nodePool.Labels, // Note: not stored in DB.
	}

	err = s.db.Save(nodePoolModel).Error
	if err != nil {
		return errors.WrapWithDetails(err, "creating node pool in database failed",
			"organizationId", organizationID,
			"clusterId", clusterID,
			"clusterName", clusterName,
			"nodePoolName", nodePool.Name,
		)
	}

	return nil
}

func (s nodePoolStore) DeleteNodePool(
	ctx context.Context, organizationID, clusterID uint, clusterName string, nodePoolName string,
) error {
	var eksCluster eksmodel.EKSClusterModel
	err := s.db.
		Where(eksmodel.EKSClusterModel{ClusterID: clusterID}).
		First(&eksCluster).Error
	if err != nil && gorm.IsRecordNotFoundError(err) {
		return cluster.NotFoundError{
			OrganizationID: organizationID,
			ClusterID:      clusterID,
			ClusterName:    clusterName,
		}
	} else if err != nil {
		return errors.WrapWithDetails(err, "fetching cluster from database failed",
			"organizationId", organizationID,
			"clusterId", clusterID,
			"clusterName", clusterName,
		)
	}

	err = s.db.
		Where(eksmodel.AmazonNodePoolsModel{ClusterID: eksCluster.ID, Name: nodePoolName}).
		Delete(eksmodel.AmazonNodePoolsModel{}).Error
	if err != nil {
		return errors.WrapWithDetails(err, "deleting node pool from database failed",
			"organizationId", organizationID,
			"clusterId", clusterID,
			"clusterName", clusterName,
			"nodePoolName", nodePoolName,
		)
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
	var eksCluster eksmodel.EKSClusterModel
	err = s.db.
		Where(eksmodel.EKSClusterModel{ClusterID: clusterID}).
		Preload("NodePools").
		First(&eksCluster).Error
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

	var nodePoolModel *eksmodel.AmazonNodePoolsModel
	for _, clusterNodePoolModel := range eksCluster.NodePools {
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
		nodePoolModel.Status = eks.NodePoolStatusEmpty
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
	nodePoolStatus eks.NodePoolStatus,
	nodePoolStatusMessage string,
) (err error) {
	var eksCluster eksmodel.EKSClusterModel
	err = s.db.
		Where(eksmodel.EKSClusterModel{ClusterID: clusterID}).
		Preload("NodePools").
		First(&eksCluster).Error
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

	var nodePoolModel *eksmodel.AmazonNodePoolsModel
	for _, clusterNodePoolModel := range eksCluster.NodePools {
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
) (existingNodePools map[string]eks.ExistingNodePool, err error) {
	var eksCluster eksmodel.EKSClusterModel
	err = s.db.
		Where(eksmodel.EKSClusterModel{ClusterID: clusterID}).
		Preload("NodePools").
		First(&eksCluster).Error
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

	existingNodePools = make(map[string]eks.ExistingNodePool, len(eksCluster.NodePools))
	for _, nodePoolModel := range eksCluster.NodePools {
		existingNodePools[nodePoolModel.Name] = eks.ExistingNodePool{
			Name:          nodePoolModel.Name,
			StackID:       nodePoolModel.StackID,
			Status:        nodePoolModel.Status,
			StatusMessage: nodePoolModel.StatusMessage,
		}
	}

	return existingNodePools, nil
}
