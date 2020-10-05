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
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	"github.com/banzaicloud/pipeline/internal/providers/pke"
	"github.com/banzaicloud/pipeline/pkg/providers"
)

type nodePoolStore struct {
	db       *gorm.DB
	clusters cluster.Store
}

// NewNodePoolStore returns a new cluster.NodePoolStore
// that persists node pools into the database using Gorm.
func NewNodePoolStore(db *gorm.DB, clusters cluster.Store) cluster.NodePoolStore {
	return nodePoolStore{
		db:       db,
		clusters: clusters,
	}
}

func (s nodePoolStore) NodePoolExists(ctx context.Context, clusterID uint, name string) (isExisting bool, storedName string, err error) {
	c, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return false, "", err
	}

	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		var eksCluster eksmodel.EKSClusterModel

		err := s.db.
			Where(eksmodel.EKSClusterModel{ClusterID: clusterID}).
			Preload("NodePools", "name = ?", name).
			First(&eksCluster).Error
		if gorm.IsRecordNotFoundError(err) {
			return false, "", errors.NewWithDetails(
				"cluster model is inconsistent",
				"clusterId", clusterID,
			)
		}
		if err != nil {
			return false, "", errors.WrapWithDetails(
				err, "failed to check if node pool exists",
				"clusterId", clusterID,
				"nodePoolName", name,
			)
		}

		if len(eksCluster.NodePools) == 0 {
			return false, "", nil
		}

		storedName = eksCluster.NodePools[0].Name

	case c.Cloud == providers.Amazon && c.Distribution == "pke":
		var pkeCluster pke.EC2PKEClusterModel

		err := s.db.
			Where(pke.EC2PKEClusterModel{ClusterID: clusterID}).
			Preload("NodePools", "name = ?", name).
			First(&pkeCluster).Error
		if gorm.IsRecordNotFoundError(err) {
			return false, "", errors.NewWithDetails(
				"cluster model is inconsistent",
				"clusterId", clusterID,
			)
		}
		if err != nil {
			return false, "", errors.WrapWithDetails(
				err, "failed to check if node pool exists",
				"clusterId", clusterID,
				"nodePoolName", name,
			)
		}

		if len(pkeCluster.NodePools) == 0 {
			return false, "", nil
		}

		storedName = pkeCluster.NodePools[0].Name
	default:
		return false, "", errors.WithStack(cluster.NotSupportedDistributionError{
			ID:           c.ID,
			Cloud:        c.Cloud,
			Distribution: c.Distribution,

			Message: "the node pool API does not support this distribution yet",
		})
	}

	return true, storedName, nil
}

func (s nodePoolStore) DeleteNodePool(ctx context.Context, clusterID uint, name string) error {
	c, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	switch {
	case c.Cloud == providers.Amazon && c.Distribution == "eks":
		var eksCluster eksmodel.EKSClusterModel

		err := s.db.Where(eksmodel.EKSClusterModel{ClusterID: clusterID}).First(&eksCluster).Error
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

		err = s.db.Where(eksmodel.AmazonNodePoolsModel{ClusterID: eksCluster.ID, Name: name}).Delete(eksmodel.AmazonNodePoolsModel{}).Error
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
