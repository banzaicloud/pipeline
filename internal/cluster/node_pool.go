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
	"fmt"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/banzaicloud/pipeline/pkg/providers"
)

// NodePoolService provides an interface to node pools.
//go:generate mga gen kit endpoint --outdir clusterdriver --outfile node_pool_endpoint_gen.go --with-oc --base-name NodePool NodePoolService
//go:generate mga gen mockery --name NodePoolService --inpkg
type NodePoolService interface {
	// CreateNodePool creates a new node pool in a cluster.
	CreateNodePool(ctx context.Context, clusterID uint, rawNodePool NewRawNodePool) error

	// DeleteNodePool deletes a node pool from a cluster.
	DeleteNodePool(ctx context.Context, clusterID uint, name string) (bool, error)
}

// NewNodePool contains generic parameters of a new node pool.
type NewNodePool struct {
	Name   string            `mapstructure:"name"`
	Labels map[string]string `mapstructure:"labels"`
}

func (n NewNodePool) Validate() error {
	var violations []string

	if n.Name == "" {
		violations = append(violations, "name cannot be empty")
	}

	for key, value := range n.Labels {
		for _, v := range validation.IsQualifiedName(key) {
			violations = append(violations, fmt.Sprintf("invalid label key %q: %s", key, v))
		}

		for _, v := range validation.IsValidLabelValue(value) {
			violations = append(violations, fmt.Sprintf("invalid label value %q: %s", value, v))
		}
	}

	if len(violations) > 0 {
		return errors.WithStack(ValidationError{
			message:    "invalid node pool creation request",
			violations: violations,
		})
	}

	return nil
}

// NewRawNodePool is an unstructured, distribution specific descriptor for a new node pool.
type NewRawNodePool map[string]interface{}

type nodePoolService struct {
	clusters  Store
	nodePools NodePoolStore
	validator NodePoolValidator
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

// NodePoolValidator validates a new node pool descriptor.
type NodePoolValidator interface {
	// Validate validates a new node pool descriptor.
	Validate(ctx context.Context, cluster Cluster, rawNodePool NewRawNodePool) error
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
	manager NodePoolManager,
) NodePoolService {
	return nodePoolService{
		clusters:  clusters,
		nodePools: nodePools,
		validator: validator,
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

	var nodePool NewNodePool

	if err := mapstructure.Decode(rawNodePool, &nodePool); err != nil {
		return errors.Wrap(err, "failed to decode node pool")
	}

	if err := nodePool.Validate(); err != nil {
		return err
	}

	if err := s.validator.Validate(ctx, cluster, rawNodePool); err != nil {
		return err
	}

	exists, err := s.nodePools.NodePoolExists(ctx, clusterID, nodePool.Name)
	if err != nil {
		return err
	}

	if exists {
		return errors.WithStack(NodePoolAlreadyExistsError{
			ClusterID: clusterID,
			NodePool:  nodePool.Name,
		})
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
