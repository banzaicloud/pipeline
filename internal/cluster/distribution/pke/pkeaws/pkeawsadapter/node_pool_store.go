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

package pkeawsadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
	pkeprovider "github.com/banzaicloud/pipeline/internal/providers/pke"
)

type nodePoolStore struct {
	db *gorm.DB
}

// NewNodePoolStore returns a new pke.NodePoolStore
// that provides an interface to pke node pool persistence.
func NewNodePoolStore(db *gorm.DB) pke.NodePoolStore {
	return nodePoolStore{
		db: db,
	}
}

func (s nodePoolStore) DeleteNodePool(
	ctx context.Context, organizationID, clusterID uint, clusterName string, nodePoolName string,
) error {
	var pkeAWSCluster pkeprovider.EC2PKEClusterModel
	err := s.db.
		Where(pkeprovider.EC2PKEClusterModel{ClusterID: clusterID}).
		First(&pkeAWSCluster).Error
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
		Where(pkeprovider.NodePool{ClusterID: pkeAWSCluster.ID, Name: nodePoolName}).
		Delete(pkeprovider.NodePool{}).Error
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

// ListNodePools retrieves the node pools for the cluster specified by its
// cluster ID.
func (s nodePoolStore) ListNodePools(
	ctx context.Context,
	organizationID uint,
	clusterID uint,
	clusterName string,
) (existingNodePools map[string]pke.ExistingNodePool, err error) {
	var pkeAWSCluster pkeprovider.EC2PKEClusterModel
	err = s.db.
		Where(pkeprovider.EC2PKEClusterModel{ClusterID: clusterID}).
		Preload("NodePools").
		First(&pkeAWSCluster).Error
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

	existingNodePools = make(map[string]pke.ExistingNodePool, len(pkeAWSCluster.NodePools))
	for _, nodePoolModel := range pkeAWSCluster.NodePools {
		existingNodePools[nodePoolModel.Name] = pke.ExistingNodePool{
			Name: nodePoolModel.Name,
		}
	}

	return existingNodePools, nil
}
