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
}

// Store provides an interface to the generic Cluster model persistence.
type Store interface {
	// GetCluster returns a generic Cluster.
	// Returns a NotFoundError when the cluster cannot be found.
	GetCluster(ctx context.Context, id uint) (Cluster, error)

	// SetStatus sets the cluster status.
	SetStatus(ctx context.Context, id uint, status string, statusMessage string) error
}

// NotFoundError is returned if a cluster cannot be found.
type NotFoundError struct {
	ID uint
}

// Error implements the error interface.
func (NotFoundError) Error() string {
	return "cluster not found"
}

// Details returns error details.
func (e NotFoundError) Details() []interface{} {
	return []interface{}{"clusterId", e.ID}
}

// NotFound tells a client that this error is related to a resource being not found.
// Can be used to translate the error to status codes for example.
func (NotFoundError) NotFound() bool {
	return true
}

// IsBusinessError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
// Deprecated: use ClientError instead.
func (NotFoundError) IsBusinessError() bool {
	return true
}

// ClientError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (NotFoundError) ClientError() bool {
	return true
}

// NotReadyError is returned when a cluster is not ready for certain actions.
type NotReadyError struct {
	ID uint
}

// Error implements the error interface.
func (NotReadyError) Error() string {
	return "cluster is not ready"
}

// Details returns error details.
func (e NotReadyError) Details() []interface{} {
	return []interface{}{"clusterId", e.ID}
}

// NotFound tells a client that this error is related to a conflicting request.
// Can be used to translate the error to status codes for example.
func (NotReadyError) Conflict() bool {
	return true
}

// IsBusinessError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
// Deprecated: use ClientError instead.
func (NotReadyError) IsBusinessError() bool {
	return true
}

// ClientError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (NotReadyError) ClientError() bool {
	return true
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

// NotFound tells a client that this error is related to an invalid request.
// Can be used to translate the error to status codes for example.
func (NotSupportedDistributionError) BadRequest() bool {
	return true
}

// IsBusinessError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
// Deprecated: use ClientError instead.
func (NotSupportedDistributionError) IsBusinessError() bool {
	return true
}

// ClientError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (NotSupportedDistributionError) ClientError() bool {
	return true
}
