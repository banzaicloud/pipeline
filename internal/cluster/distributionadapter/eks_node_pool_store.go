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

package distributionadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution"
	"github.com/banzaicloud/pipeline/src/model"
)

// NodePoolStore provides an interface to EKS node pool persistence.
type EKSNodePoolStore struct {
	db *gorm.DB
}

// NewEKSNodePoolStore returns a new EKSNodePoolStore.
func NewEKSNodePoolStore(db *gorm.DB) EKSNodePoolStore {
	return EKSNodePoolStore{
		db: db,
	}
}

// CreateNodePool saves a new node pool.
func (s EKSNodePoolStore) CreateNodePool(
	_ context.Context,
	clusterID uint,
	createdBy uint,
	nodePool distribution.NewEKSNodePool,
) error {
	nodePoolModel := &model.AmazonNodePoolsModel{
		ClusterID:        clusterID,
		CreatedBy:        createdBy,
		Name:             nodePool.Name,
		NodeInstanceType: nodePool.InstanceType,
		NodeImage:        nodePool.Image,
		NodeSpotPrice:    nodePool.SpotPrice,
		Autoscaling:      nodePool.Autoscaling.Enabled,
		NodeMinCount:     nodePool.Autoscaling.MinSize,
		NodeMaxCount:     nodePool.Autoscaling.MaxSize,
		Count:            nodePool.Size,
	}

	err := s.db.Save(nodePoolModel).Error
	if err != nil {
		return errors.Wrap(err, "failed to save node pool")
	}

	return nil
}
