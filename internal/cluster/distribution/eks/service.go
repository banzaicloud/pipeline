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

	Options NodePoolUpdateOptions `mapstructure:"options"`
}

type NodePoolUpdateOptions struct {

	// Maximum number of extra nodes that can be created during the update.
	MaxSurge int `mapstructure:"maxSurge"`

	// Maximum number of nodes that can be updated simultaneously.
	MaxBatchSize int `mapstructure:"maxBatchSize"`

	// Maximum number of nodes that can be unavailable during the update.
	MaxUnavailable int `mapstructure:"maxUnavailable"`

	// Kubernetes node drain specific options.
	Drain NodePoolUpdateDrainOptions `mapstructure:"drain"`
}

type NodePoolUpdateDrainOptions struct {
	Timeout int `mapstructure:"timeout"`

	FailOnError bool `mapstructure:"failOnError"`

	PodSelector string `mapstructure:"podSelector"`
}

// NodePool encapsulates information about a cluster node pool.
type NodePool struct {
	Name         string            `mapstructure:"name"`
	Labels       map[string]string `mapstructure:"labels"`
	Size         int               `mapstructure:"size"`
	Autoscaling  Autoscaling       `mapstructure:"autoscaling"`
	InstanceType string            `mapstructure:"instanceType"`
	Image        string            `mapstructure:"image"`
	SpotPrice    string            `mapstructure:"spotPrice"`
}

// Autoscaling describes the EKS node pool's autoscaling settings.
type Autoscaling struct {
	Enabled bool `mapstructure:"enabled"`
	MinSize int  `mapstructure:"minSize"`
	MaxSize int  `mapstructure:"maxSize"`
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

	// ListNodePools lists node pools from a cluster.
	ListNodePools(ctx context.Context, c cluster.Cluster, nodePoolNames []string) ([]NodePool, error)
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
