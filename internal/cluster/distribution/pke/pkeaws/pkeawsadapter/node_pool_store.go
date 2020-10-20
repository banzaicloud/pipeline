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

	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke"
)

type nodePoolStore struct {
	db *gorm.DB
}

// NewNodePoolStore returns a new eks.NodePoolStore
// that provides an interface to EKS node pool persistence.
func NewNodePoolStore(db *gorm.DB) pke.NodePoolStore {
	return nodePoolStore{
		db: db,
	}
}

func (s nodePoolStore) CreateNodePool(
	_ context.Context,
	clusterID uint,
	createdBy uint,
	nodePool pke.NewNodePool,
) error {
	panic("not implemented")
}

func (s nodePoolStore) DeleteNodePool(
	ctx context.Context, organizationID, clusterID uint, clusterName string, nodePoolName string,
) error {
	panic("not implemented")
}

// ListNodePools retrieves the node pools for the cluster specified by its
// cluster ID.
func (s nodePoolStore) ListNodePools(
	ctx context.Context,
	organizationID uint,
	clusterID uint,
	clusterName string,
) (existingNodePools map[string]pke.ExistingNodePool, err error) {
	panic("not implemented")
}
