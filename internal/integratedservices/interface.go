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

package integratedservices

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
)

// IntegratedService represents the state of an integrated service.
type IntegratedService struct {
	Name   string                  `json:"name"`
	Spec   IntegratedServiceSpec   `json:"spec"`
	Output IntegratedServiceOutput `json:"output"`
	Status string                  `json:"status"`
}

// IntegratedServiceSpec represents an integrated service's specification (i.e. its input parameters).
type IntegratedServiceSpec = map[string]interface{}

// IntegratedServiceOutput represents an integrated service's output.
type IntegratedServiceOutput = map[string]interface{}

// IntegratedServiceStatus represents an integrated service's status.
type IntegratedServiceStatus = string

// IntegratedService status constants
const (
	IntegratedServiceStatusInactive IntegratedServiceStatus = "INACTIVE"
	IntegratedServiceStatusPending  IntegratedServiceStatus = "PENDING"
	IntegratedServiceStatusActive   IntegratedServiceStatus = "ACTIVE"
	IntegratedServiceStatusError    IntegratedServiceStatus = "ERROR"
)

// +testify:mock:testOnly=true

// IntegratedServiceManagerRegistry contains integrated service managers.
type IntegratedServiceManagerRegistry interface {
	// GetIntegratedServiceManager retrieves an integrated service manager by name.
	GetIntegratedServiceManager(integratedServiceName string) (IntegratedServiceManager, error)
	// GetIntegratedServiceNames retrieves all known integrated services
	GetIntegratedServiceNames() []string
}

// IntegratedServiceOperatorRegistry contains integrated service operators.
type IntegratedServiceOperatorRegistry interface {
	// GetIntegratedServiceOperator retrieves an integrated service operator by name.
	GetIntegratedServiceOperator(integratedServiceName string) (IntegratedServiceOperator, error)
}

// +testify:mock:testOnly=true

// IntegratedServiceRepository manages integrated service state.
type IntegratedServiceRepository interface {
	// GetIntegratedServices retrieves integrated services for a given cluster.
	GetIntegratedServices(ctx context.Context, clusterID uint) ([]IntegratedService, error)

	// GetIntegratedService retrieves an integrated service.
	GetIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string) (IntegratedService, error)

	// SaveIntegratedService persists an integrated service.
	SaveIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec, status string) error

	// UpdateIntegratedServiceStatus updates the status of an integrated service.
	UpdateIntegratedServiceStatus(ctx context.Context, clusterID uint, integratedServiceName string, status string) error

	// UpdateIntegratedServiceSpec updates the spec of an integrated service.
	UpdateIntegratedServiceSpec(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec) error

	// DeleteIntegratedService deletes an integrated service.
	DeleteIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string) error
}

type IntegratedServiceCleaner interface {
	DisableServiceInstance(ctx context.Context, clusterID uint) error
}

// IsIntegratedServiceNotFoundError returns true when the specified error is a "integrated service not found" error
func IsIntegratedServiceNotFoundError(err error) bool {
	var notFoundErr interface {
		IntegratedServiceNotFound() bool
	}
	return errors.As(err, &notFoundErr) && notFoundErr.IntegratedServiceNotFound()
}

// IsUnknownIntegratedServiceError returns true when the specified error is a "integrated service is unknown" error
func IsUnknownIntegratedServiceError(err error) bool {
	var unknownSvcErr interface {
		Unknown() bool
	}
	return errors.As(err, &unknownSvcErr) && unknownSvcErr.Unknown()
}

// IntegratedServiceManager is a collection of integrated service specific methods that are used synchronously when responding to integrated service related requests.
type IntegratedServiceManager interface {
	IntegratedServiceOutputProducer
	IntegratedServiceSpecValidator
	IntegratedServiceSpecPreparer

	// Name returns the integrated service's name.
	Name() string
}

// IntegratedServiceOutputProducer defines how to produce an integrated service's output.
type IntegratedServiceOutputProducer interface {
	// GetOutput returns an integrated service's output.
	GetOutput(ctx context.Context, clusterID uint, spec IntegratedServiceSpec) (IntegratedServiceOutput, error)
}

// IntegratedServiceSpecValidator defines how to validate an integrated service specification
type IntegratedServiceSpecValidator interface {
	// ValidateSpec validates an integrated service specification.
	ValidateSpec(ctx context.Context, spec IntegratedServiceSpec) error
}

// IsInputValidationError returns true if the error is an input validation error
func IsInputValidationError(err error) bool {
	var inputValidationError interface {
		InputValidationError() bool
	}
	return errors.As(err, &inputValidationError) && inputValidationError.InputValidationError()
}

// InvalidIntegratedServiceSpecError is returned when an integrated service specification fails the validation.
type InvalidIntegratedServiceSpecError struct {
	IntegratedServiceName string
	Problem               string
}

func (e InvalidIntegratedServiceSpecError) Error() string {
	return "invalid integrated service spec: " + e.Problem
}

// Details returns the error's details
func (e InvalidIntegratedServiceSpecError) Details() []interface{} {
	return []interface{}{"integrated service", e.IntegratedServiceName}
}

// InputValidationError returns true since InputValidationError is an input validation error
func (InvalidIntegratedServiceSpecError) InputValidationError() bool {
	return true
}

// Validation tells a client that this error is related to a semantic validation of the request.
// Can be used to translate the error to status codes for example.
func (InvalidIntegratedServiceSpecError) Validation() bool {
	return true
}

// ServiceError tells the consumer whether this error is caused by invalid input supplied by the client.
// Client errors are usually returned to the consumer without retrying the operation.
func (InvalidIntegratedServiceSpecError) ServiceError() bool {
	return true
}

// IntegratedServiceSpecPreparer defines how an integrated service specification is prepared before it's sent to be applied
type IntegratedServiceSpecPreparer interface {
	// PrepareSpec makes certain preparations to the spec before it's sent to be applied.
	// For example it rewrites the secret ID to it's internal representation, fills in defaults, etc.
	PrepareSpec(ctx context.Context, clusterID uint, spec IntegratedServiceSpec) (IntegratedServiceSpec, error)
}

// PassthroughIntegratedServiceSpecPreparer implements IntegratedServiceSpecPreparer by making no modifications to the integrated service spec
type PassthroughIntegratedServiceSpecPreparer struct{}

// PrepareSpec returns the provided spec without any modifications
func (PassthroughIntegratedServiceSpecPreparer) PrepareSpec(_ context.Context, _ uint, spec IntegratedServiceSpec) (IntegratedServiceSpec, error) {
	return spec, nil
}

// +testify:mock:testOnly=true

// IntegratedServiceOperationDispatcher dispatches cluster integrated service operations asynchronously.
type IntegratedServiceOperationDispatcher interface {
	// DispatchApply starts applying a desired state for an integrated service asynchronously.
	DispatchApply(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec) error

	// DispatchDeactivate starts deactivating an integrated service asynchronously.
	DispatchDeactivate(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec) error

	// IsBeingDispatched checks whether there are any actively running workflows for the service
	IsBeingDispatched(ctx context.Context, clusterID uint, integratedServiceName string) (bool, error)
}

// IntegratedServiceOperator defines the operations that can be applied to an integrated service.
type IntegratedServiceOperator interface {
	// Apply applies a desired state for an integrated service on the given cluster.
	Apply(ctx context.Context, clusterID uint, spec IntegratedServiceSpec) error

	// Deactivate deactivates an integrated service on the given cluster.
	Deactivate(ctx context.Context, clusterID uint, spec IntegratedServiceSpec) error

	// Name returns the integrated service's name.
	Name() string
}

// +testify:mock

// ClusterService provides a thin access layer to clusters.
type ClusterService interface {
	// CheckClusterReady checks whether the cluster is ready for integrated services (eg.: exists and it's running). If the cluster is not ready, a ClusterIsNotReadyError should be returned.
	CheckClusterReady(ctx context.Context, clusterID uint) error
}

// ClusterIsNotReadyError is returned when a cluster is not in a ready state.
type ClusterIsNotReadyError struct {
	ClusterID uint
}

func (e ClusterIsNotReadyError) Error() string {
	return "cluster is not ready"
}

// Details returns the error's details
func (e ClusterIsNotReadyError) Details() []interface{} {
	return []interface{}{"clusterId", e.ClusterID}
}

// ShouldRetry returns true if the operation resulting in this error should be retried later.
func (e ClusterIsNotReadyError) ShouldRetry() bool {
	return true
}

type SpecConversion interface {
	// ConvertSpec converts in integrated service spec while keeping it's original structure
	ConvertSpec(ctx context.Context, instance v1alpha1.ServiceInstance) (IntegratedServiceSpec, error)
}

type NotManagedIntegratedServiceError struct {
	IntegratedServiceName string
}

func (NotManagedIntegratedServiceError) ServiceError() bool {
	return true
}

func (n NotManagedIntegratedServiceError) Error() string {
	return fmt.Sprintf("the %s integrated service is not managed by pipeline", n.IntegratedServiceName)
}
