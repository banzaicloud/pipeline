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

	"github.com/banzaicloud/pipeline/pkg/brn"
)

// Cluster status constants
const (
	Creating = "CREATING"
	Running  = "RUNNING"
	Updating = "UPDATING"
	Deleting = "DELETING"
	Warning  = "WARNING"
	Error    = "ERROR"

	CreatingMessage = "Cluster creation is in progress"
	RunningMessage  = "Cluster is running"
	UpdatingMessage = "Update is in progress"
	DeletingMessage = "Termination is in progress"
)

// Cluster represents a generic, provider agnostic Kubernetes cluster structure.
type Cluster struct {
	ID   uint
	UID  string
	Name string

	OrganizationID uint

	Status        string
	StatusMessage string

	Cloud        string
	Distribution string
	Location     string

	SecretID       brn.ResourceName
	ConfigSecretID brn.ResourceName
	Tags           map[string]string
}

type Identifier struct {
	OrganizationID uint
	ClusterID      uint
	ClusterName    string
}

// +testify:mock:testOnly=true

// Store provides an interface to the generic Cluster model persistence.
type Store interface {
	// GetCluster returns a generic Cluster.
	// Returns an error with the NotFound behavior when the cluster cannot be found.
	GetCluster(ctx context.Context, id uint) (Cluster, error)

	// GetClusterByName returns a generic Cluster.
	// Returns an error with the NotFound behavior when the cluster cannot be found.
	GetClusterByName(ctx context.Context, orgID uint, clusterName string) (Cluster, error)

	// SetStatus sets the cluster status.
	SetStatus(ctx context.Context, id uint, status string, statusMessage string) error
}

// +testify:mock:testOnly=true
type ClusterGroupManager interface {
	ValidateClusterRemoval(ctx context.Context, clusterID uint) error
}

// ClusterDeleteNotPermittedError is returned if a cluster cannot be deleted.
type ClusterDeleteNotPermittedError struct {
	OrganizationID uint
	ClusterID      uint
	ClusterName    string
	Msg            string
}

// Error implements the error interface.
func (e ClusterDeleteNotPermittedError) Error() string {
	return e.Msg
}

func (e ClusterDeleteNotPermittedError) Validation() bool {
	return true
}

// ServiceError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (ClusterDeleteNotPermittedError) ServiceError() bool {
	return true
}

// Details returns error details.
func (e ClusterDeleteNotPermittedError) Details() []interface{} {
	return []interface{}{"clusterId", e.ClusterID, "clusterName", e.ClusterName, "orgId", e.OrganizationID}
}

// NotFoundError is returned if a cluster cannot be found.
type NotFoundError struct {
	OrganizationID uint
	ClusterID      uint
	ClusterName    string
}

// Error implements the error interface.
func (NotFoundError) Error() string {
	return "cluster not found"
}

// Details returns error details.
func (e NotFoundError) Details() []interface{} {
	return []interface{}{"clusterId", e.ClusterID, "clusterName", e.ClusterName, "orgId", e.OrganizationID}
}

// NotFound tells a client that this error is related to a resource being not found.
// Can be used to translate the error to status codes for example.
func (NotFoundError) NotFound() bool {
	return true
}

// IsNotFoundError returns true if the error implements the NotFound behavior and it returns true.
func IsNotFoundError(err error) bool {
	var nfe interface {
		NotFound() bool
	}

	return errors.As(err, &nfe) && nfe.NotFound()
}

// ServiceError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (NotFoundError) ServiceError() bool {
	return true
}

// NotReadyError is returned when a cluster is not ready for certain actions.
type NotReadyError struct {
	OrganizationID uint
	ID             uint
	Name           string
}

// Error implements the error interface.
func (NotReadyError) Error() string {
	return "cluster is not ready"
}

// Details returns error details.
func (e NotReadyError) Details() []interface{} {
	return []interface{}{"clusterId", e.ID, "clusterName", e.Name, "orgId", e.OrganizationID}
}

// Conflict tells a client that this error is related to a conflicting request.
// Can be used to translate the error to status codes for example.
func (NotReadyError) Conflict() bool {
	return true
}

// IsBusinessError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
// Deprecated: use ServiceError instead.
func (NotReadyError) IsBusinessError() bool {
	return true
}

// ServiceError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (NotReadyError) ServiceError() bool {
	return true
}

// +kit:endpoint:errorStrategy=service
// +testify:mock

// Service provides an interface to clusters.
type Service interface {
	// UpdateCluster updates the specified cluster.
	UpdateCluster(ctx context.Context, clusterIdentifier Identifier, clusterUpdate ClusterUpdate) (err error)

	// DeleteCluster deletes the specified cluster. It returns true if the cluster is already deleted.
	DeleteCluster(ctx context.Context, clusterIdentifier Identifier, options DeleteClusterOptions) (deleted bool, err error)

	// CreateNodePool creates a new node pool in a cluster.
	CreateNodePool(ctx context.Context, clusterID uint, rawNodePool NewRawNodePool) error

	// UpdateNodePool updates an existing node pool in a cluster.
	UpdateNodePool(ctx context.Context, clusterID uint, nodePoolName string, rawNodePoolUpdate RawNodePoolUpdate) (processID string, err error)

	// DeleteNodePool deletes a node pool from a cluster.
	DeleteNodePool(ctx context.Context, clusterID uint, name string) (deleted bool, err error)

	// ListNodePools lists node pools from a cluster.
	ListNodePools(ctx context.Context, clusterID uint) (nodePoolList RawNodePoolList, err error)
}

// DeleteClusterOptions represents cluster deletion options.
type DeleteClusterOptions struct {
	Force bool
}

// ClusterUpdate represents cluster update parameters.
type ClusterUpdate struct {
	Version string
}

type service struct {
	clusters            Store
	clusterManager      Manager
	clusterGroupManager ClusterGroupManager

	distributions map[string]Service

	nodePools         NodePoolStore
	nodePoolValidator NodePoolValidator
	nodePoolProcessor NodePoolProcessor
	nodePoolManager   NodePoolManager
}

// +testify:mock:testOnly=true

// Manager provides lower level cluster operations for Service.
type Manager interface {
	Deleter
}

// Deleter can be used to delete a cluster.
type Deleter interface {
	// DeleteCluster deletes the specified cluster.
	DeleteCluster(ctx context.Context, clusterID uint, options DeleteClusterOptions) error
}

// NewService returns a new Service instance
func NewService(
	clusters Store,
	clusterManager Manager,
	clusterGroupManager ClusterGroupManager,
	distributions map[string]Service,
	nodePools NodePoolStore,
	nodePoolValidator NodePoolValidator,
	nodePoolProcessor NodePoolProcessor,
	nodePoolManager NodePoolManager,
) Service {
	return service{
		clusters:            clusters,
		clusterManager:      clusterManager,
		clusterGroupManager: clusterGroupManager,

		distributions: distributions,

		nodePools:         nodePools,
		nodePoolValidator: nodePoolValidator,
		nodePoolProcessor: nodePoolProcessor,
		nodePoolManager:   nodePoolManager,
	}
}

// DeleteCluster deletes the specified cluster. It returns true if the cluster is already deleted.
func (s service) DeleteCluster(ctx context.Context, clusterIdentifier Identifier, options DeleteClusterOptions) (bool, error) {
	var (
		c   Cluster
		err error
	)

	if clusterIdentifier.ClusterName != "" {
		c, err = s.clusters.GetClusterByName(ctx, clusterIdentifier.OrganizationID, clusterIdentifier.ClusterName)
	} else {
		c, err = s.clusters.GetCluster(ctx, clusterIdentifier.ClusterID)
	}

	if IsNotFoundError(err) {
		// Already deleted
		return true, nil
	} else if err != nil {
		return false, err
	}

	err = s.clusterGroupManager.ValidateClusterRemoval(ctx, clusterIdentifier.ClusterID)
	if err != nil {
		return false, ClusterDeleteNotPermittedError{
			OrganizationID: clusterIdentifier.OrganizationID,
			ClusterName:    clusterIdentifier.ClusterName,
			ClusterID:      clusterIdentifier.ClusterID,
			Msg:            err.Error(),
		}
	}

	if err := s.clusters.SetStatus(ctx, c.ID, Deleting, DeletingMessage); err != nil {
		return false, err
	}

	if err := s.clusterManager.DeleteCluster(ctx, c.ID, options); err != nil {
		return false, err
	}

	return false, nil
}

// NotSupportedDistributionError is returned if an API does not support a certain distribution.
type NotSupportedDistributionError struct {
	ID           uint
	Cloud        string
	Distribution string

	Message string
}

// Error implements the error interface.
func (e NotSupportedDistributionError) Error() string {
	return e.Message
}

// Details returns error details.
func (e NotSupportedDistributionError) Details() []interface{} {
	return []interface{}{
		"clusterId", e.ID,
		"cloud", e.Cloud,
		"distribution", e.Distribution,
	}
}

// BadRequest tells a client that this error is related to an invalid request.
// Can be used to translate the error to status codes for example.
func (NotSupportedDistributionError) BadRequest() bool {
	return true
}

// IsBusinessError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
// Deprecated: use ServiceError instead.
func (NotSupportedDistributionError) IsBusinessError() bool {
	return true
}

// ServiceError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (NotSupportedDistributionError) ServiceError() bool {
	return true
}

func (s service) getDistributionService(cluster Cluster) (Service, error) {
	key := cluster.Distribution
	if key == "pke" {
		key += cluster.Cloud
	}
	service, ok := s.distributions[key]
	if !ok {
		return nil, errors.WithStack(NotSupportedDistributionError{
			ID:           cluster.ID,
			Cloud:        cluster.Cloud,
			Distribution: cluster.Distribution,

			Message: "not supported distribution",
		})
	}

	return service, nil
}
