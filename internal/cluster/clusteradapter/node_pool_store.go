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

package clusteradapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/model"
)

// NodePoolStore provides an interface to node pool persistence.
type NodePoolStore struct {
	db       *gorm.DB
	clusters cluster.Store
}

// NewNodePoolStore returns a new NodePoolStore.
func NewNodePoolStore(db *gorm.DB, clusters cluster.Store) NodePoolStore {
	return NodePoolStore{
		db:       db,
		clusters: clusters,
	}
}

// NodePoolExists checks if a node pool exists.
func (s NodePoolStore) NodePoolExists(ctx context.Context, clusterID uint, name string) (bool, error) {
	c, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return false, err
	}

	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		var eksCluster model.EKSClusterModel

		err := s.db.
			Where(model.EKSClusterModel{ClusterID: clusterID}).
			Preload("NodePools", "name = ?", name).
			First(&eksCluster).Error
		if gorm.IsRecordNotFoundError(err) {
			return false, errors.NewWithDetails(
				"cluster model is inconsistent",
				"clusterId", clusterID,
			)
		}
		if err != nil {
			return false, errors.WrapWithDetails(
				err, "failed to check if node pool exists",
				"clusterId", clusterID,
				"nodePoolName", name,
			)
		}

		if len(eksCluster.NodePools) == 0 {
			return false, nil
		}

	default:
		return false, errors.WithStack(cluster.NotSupportedDistributionError{
			ID:           c.ID,
			Cloud:        c.Cloud,
			Distribution: c.Distribution,

			Message: "the node pool API does not support this distribution yet",
		})
	}

	return true, nil
}

// DeleteNodePool deletes a node pool.
func (s NodePoolStore) DeleteNodePool(ctx context.Context, clusterID uint, name string) error {
	c, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		var eksCluster model.EKSClusterModel

		err := s.db.Where(model.EKSClusterModel{ClusterID: clusterID}).First(&eksCluster).Error
		if gorm.IsRecordNotFoundError(err) {
			return errors.NewWithDetails(
				"cluster model is inconsistent",
				"clusterId", clusterID,
			)
		}
		if err != nil {
			return errors.WrapWithDetails(
				err, "failed to delete node pool",
				"clusterId", clusterID,
				"nodePoolName", name,
			)
		}

		err = s.db.Where(model.AmazonNodePoolsModel{ClusterID: eksCluster.ID, Name: name}).Delete(model.AmazonNodePoolsModel{}).Error
		if err != nil {
			return errors.WrapWithDetails(
				err, "failed to delete node pool",
				"clusterId", clusterID,
				"nodePoolName", name,
			)
		}

	default:
		return errors.WithStack(cluster.NotSupportedDistributionError{
			ID:           c.ID,
			Cloud:        c.Cloud,
			Distribution: c.Distribution,

			Message: "the node pool API does not support this distribution yet",
		})
	}

	return nil
}
