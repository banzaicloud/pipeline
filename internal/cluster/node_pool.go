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

package cluster

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/pkg/providers"
)

// NodePoolService provides an interface to node pools.
//go:generate mga gen kit endpoint --outdir clusterdriver --outfile node_pool_endpoint_gen.go --with-oc --base-name NodePool NodePoolService
//go:generate mockery -name NodePoolService -inpkg
type NodePoolService interface {
	// DeleteNodePool deletes a node pool from a cluster.
	DeleteNodePool(ctx context.Context, clusterID uint, name string) (bool, error)
}

type nodePoolService struct {
	clusters  Store
	nodePools NodePoolStore
	manager   NodePoolManager
}

// NodePoolStore provides an interface to node pool persistence.
type NodePoolStore interface {
	// NodePoolExists checks if a node pool exists.
	NodePoolExists(ctx context.Context, clusterID uint, name string) (bool, error)

	// DeleteNodePool deletes a node pool.
	DeleteNodePool(ctx context.Context, clusterID uint, name string) error
}

// NodePoolManager manages node pool infrastructure.
type NodePoolManager interface {
	// DeleteNodePool deletes a node pool from a cluster.
	DeleteNodePool(ctx context.Context, clusterID uint, name string) error
}

// NewNodePoolService returns a new NodePoolService.
func NewNodePoolService(clusters Store, nodePools NodePoolStore, manager NodePoolManager) NodePoolService {
	return nodePoolService{
		clusters:  clusters,
		nodePools: nodePools,
		manager:   manager,
	}
}

func (s nodePoolService) DeleteNodePool(ctx context.Context, clusterID uint, name string) (bool, error) {
	cluster, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return false, err
	}

	if err := s.supported(cluster); err != nil {
		return false, err
	}

	if cluster.Status != Running && cluster.Status != Warning {
		return false, errors.WithStack(NotReadyError{ID: clusterID})
	}

	exists, err := s.nodePools.NodePoolExists(ctx, clusterID, name)
	if err != nil {
		return false, err
	}

	// Already deleted
	if !exists {
		return true, nil
	}

	err = s.clusters.SetStatus(ctx, clusterID, Updating, "deleting node pool")
	if err != nil {
		return false, err
	}

	err = s.manager.DeleteNodePool(ctx, clusterID, name)
	if err != nil {
		return false, err
	}

	return false, nil
}

func (s nodePoolService) supported(cluster Cluster) error {
	switch {
	case cluster.Cloud == providers.Amazon && cluster.Distribution == "eks":
		return nil
	}

	return errors.WithStack(NotSupportedDistributionError{
		ID:           cluster.ID,
		Cloud:        cluster.Cloud,
		Distribution: cluster.Distribution,

		Message: "the node pool API does not support this distribution yet",
	})
}
