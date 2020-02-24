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

package eks

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

// Service provides an interface to EKS clusters.
type Service interface {
	// UpdateNodePool updates an existing node pool in a cluster.
	//
	// This method accepts a partial body representation.
	UpdateNodePool(ctx context.Context, clusterID uint, nodePoolName string, nodePoolUpdate NodePoolUpdate) (string, error)
}

// NodePoolUpdate describes a node pool update request.
//
// A node pool update contains a partial representation of the node pool resource,
// updating only the changed values.
type NodePoolUpdate struct {
	Image string `mapstructure:"image"`
}

// NewService returns a new Service instance.
func NewService(
	genericClusters cluster.Store,
	nodePools NodePoolStore,
	nodePoolManager NodePoolManager,
) Service {
	return service{
		genericClusters: genericClusters,
		nodePools:       nodePools,
		nodePoolManager: nodePoolManager,
	}
}

type service struct {
	genericClusters cluster.Store
	nodePools       NodePoolStore
	nodePoolManager NodePoolManager
}

// NodePoolManager is responsible for managing node pools.
type NodePoolManager interface {
	// UpdateNodePool updates an existing node pool in a cluster.
	UpdateNodePool(ctx context.Context, c cluster.Cluster, nodePoolName string, nodePoolUpdate NodePoolUpdate) (string, error)
}

func (s service) UpdateNodePool(
	ctx context.Context,
	clusterID uint,
	nodePoolName string,
	nodePoolUpdate NodePoolUpdate,
) (string, error) {
	// TODO: check if node pool exists

	c, err := s.genericClusters.GetCluster(ctx, clusterID)
	if err != nil {
		return "", err
	}

	err = s.genericClusters.SetStatus(ctx, clusterID, cluster.Updating, "updating node pool")
	if err != nil {
		return "", err
	}

	return s.nodePoolManager.UpdateNodePool(ctx, c, nodePoolName, nodePoolUpdate)
}
