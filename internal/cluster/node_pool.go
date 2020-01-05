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
	"strconv"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/pkg/providers"
)

// NodePool is a common interface for all distribution node pools.
type NodePool interface {
	// GetName returns the node pool name.
	GetName() string

	// GetInstanceType returns the node pool instance type.
	GetInstanceType() string

	// IsOnDemand determines whether the machines in the node pool are on demand or spot/preemtible instances.
	IsOnDemand() bool

	// GetLabels returns labels that are/should be applied to every node in the pool.
	GetLabels() map[string]string
}

// NodePoolService provides an interface to node pools.
//go:generate mga gen kit endpoint --outdir clusterdriver --outfile node_pool_endpoint_gen.go --with-oc --base-name NodePool NodePoolService
//go:generate mga gen mockery --name NodePoolService --inpkg
type NodePoolService interface {
	// CreateNodePool creates a new node pool in a cluster.
	CreateNodePool(ctx context.Context, clusterID uint, rawNodePool NewRawNodePool) error

	// DeleteNodePool deletes a node pool from a cluster.
	DeleteNodePool(ctx context.Context, clusterID uint, name string) (bool, error)
}

// NewRawNodePool is an unstructured, distribution specific descriptor for a new node pool.
type NewRawNodePool map[string]interface{}

// GetName returns the node pool name.
func (n NewRawNodePool) GetName() string {
	name, ok := n["name"].(string)
	if !ok {
		return ""
	}

	return name
}

// GetInstanceType returns the node pool instance type.
func (n NewRawNodePool) GetInstanceType() string {
	instanceType, ok := n["instanceType"].(string)
	if !ok {
		return ""
	}

	return instanceType
}

// IsOnDemand determines whether the machines in the node pool are on demand or spot/preemtible instances.
func (n NewRawNodePool) IsOnDemand() bool {
	if spotPrice, ok := n["spotPrice"].(string); ok {
		if price, err := strconv.ParseFloat(spotPrice, 64); err == nil {
			return price <= 0.0
		}
	}

	if preemptible, ok := n["preemptible"].(bool); ok {
		return !preemptible
	}

	return true
}

// GetLabels returns labels that are/should be applied to every node in the pool.
func (n NewRawNodePool) GetLabels() map[string]string {
	var labels map[string]string

	l, ok := n["labels"]
	if !ok {
		return map[string]string{}
	}

	err := mapstructure.Decode(l, &labels)
	if err != nil {
		return map[string]string{}
	}

	return labels
}

type nodePoolService struct {
	clusters  Store
	nodePools NodePoolStore
	validator NodePoolValidator
	processor NodePoolProcessor
	manager   NodePoolManager
}

// NodePoolAlreadyExistsError is returned when a node pool already exists.
type NodePoolAlreadyExistsError struct {
	ClusterID uint
	NodePool  string
}

// Error implements the error interface.
func (NodePoolAlreadyExistsError) Error() string {
	return "node pool already exists"
}

// Details returns error details.
func (e NodePoolAlreadyExistsError) Details() []interface{} {
	return []interface{}{"clusterId", e.ClusterID, "nodePool", e.NodePool}
}

// NotFound tells a client that this error is related to a conflicting request.
// Can be used to translate the error to status codes for example.
func (NodePoolAlreadyExistsError) Conflict() bool {
	return true
}

// IsBusinessError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
// Deprecated: use ClientError instead.
func (NodePoolAlreadyExistsError) IsBusinessError() bool {
	return true
}

// ClientError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (NodePoolAlreadyExistsError) ClientError() bool {
	return true
}

// NodePoolStore provides an interface to node pool persistence.
type NodePoolStore interface {
	// NodePoolExists checks if a node pool exists.
	NodePoolExists(ctx context.Context, clusterID uint, name string) (bool, error)

	// DeleteNodePool deletes a node pool.
	DeleteNodePool(ctx context.Context, clusterID uint, name string) error
}

// NodePoolValidator validates a node pool descriptor.
type NodePoolValidator interface {
	// ValidateNew validates a new node pool descriptor.
	ValidateNew(ctx context.Context, cluster Cluster, rawNodePool NewRawNodePool) error
}

// NodePoolProcessor processes a node pool descriptor.
type NodePoolProcessor interface {
	// ProcessNew processes a new node pool descriptor.
	ProcessNew(ctx context.Context, cluster Cluster, rawNodePool NewRawNodePool) (NewRawNodePool, error)
}

// NodePoolManager manages node pool infrastructure.
type NodePoolManager interface {
	// CreateNodePool creates a new node pool in a cluster.
	CreateNodePool(ctx context.Context, clusterID uint, rawNodePool NewRawNodePool) error

	// DeleteNodePool deletes a node pool from a cluster.
	DeleteNodePool(ctx context.Context, clusterID uint, name string) error
}

// NewNodePoolService returns a new NodePoolService.
func NewNodePoolService(
	clusters Store,
	nodePools NodePoolStore,
	validator NodePoolValidator,
	processor NodePoolProcessor,
	manager NodePoolManager,
) NodePoolService {
	return nodePoolService{
		clusters:  clusters,
		nodePools: nodePools,
		validator: validator,
		processor: processor,
		manager:   manager,
	}
}

func (s nodePoolService) CreateNodePool(
	ctx context.Context,
	clusterID uint,
	rawNodePool NewRawNodePool,
) error {
	cluster, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := s.checkCluster(cluster); err != nil {
		return err
	}

	if err := s.validator.ValidateNew(ctx, cluster, rawNodePool); err != nil {
		return err
	}

	exists, err := s.nodePools.NodePoolExists(ctx, clusterID, rawNodePool.GetName())
	if err != nil {
		return err
	}

	if exists {
		return errors.WithStack(NodePoolAlreadyExistsError{
			ClusterID: clusterID,
			NodePool:  rawNodePool.GetName(),
		})
	}

	rawNodePool, err = s.processor.ProcessNew(ctx, cluster, rawNodePool)
	if err != nil {
		return err
	}

	err = s.clusters.SetStatus(ctx, clusterID, Updating, "creating node pool")
	if err != nil {
		return err
	}

	err = s.manager.CreateNodePool(ctx, clusterID, rawNodePool)
	if err != nil {
		return err
	}

	return nil
}

func (s nodePoolService) DeleteNodePool(ctx context.Context, clusterID uint, name string) (bool, error) {
	cluster, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return false, err
	}

	if err := s.checkCluster(cluster); err != nil {
		return false, err
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

func (s nodePoolService) checkCluster(cluster Cluster) error {
	if err := s.supported(cluster); err != nil {
		return err
	}

	if cluster.Status != Running && cluster.Status != Warning {
		return errors.WithStack(NotReadyError{ID: cluster.ID})
	}

	return nil
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
