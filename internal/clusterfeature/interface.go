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

package clusterfeature

import (
	"context"

	"emperror.dev/errors"
)

// Feature represents the state of a cluster feature.
type Feature struct {
	Name   string        `json:"name"`
	Spec   FeatureSpec   `json:"spec"`
	Output FeatureOutput `json:"output"`
	Status string        `json:"status"`
}

// FeatureSpec represents a feature's specification (i.e. its input parameters).
type FeatureSpec = map[string]interface{}

// FeatureOutput represents a feature's output.
type FeatureOutput = map[string]interface{}

// FeatureStatus represents a feature's status.
type FeatureStatus = string

// Feature status constants
const (
	FeatureStatusPending FeatureStatus = "PENDING"
	FeatureStatusActive  FeatureStatus = "ACTIVE"
	FeatureStatusError   FeatureStatus = "ERROR"
)

// FeatureManagerRegistry contains feature managers.
type FeatureManagerRegistry interface {
	// GetFeatureManager retrieves a feature manager by name.
	GetFeatureManager(featureName string) (FeatureManager, error)
}

// FeatureOperatorRegistry contains feature operators.
type FeatureOperatorRegistry interface {
	// GetFeatureOperator retrieves a feature operator by name.
	GetFeatureOperator(featureName string) (FeatureOperator, error)
}

// FeatureRepository manages feature state.
type FeatureRepository interface {
	// GetFeatures retrieves features for a given cluster.
	GetFeatures(ctx context.Context, clusterID uint) ([]Feature, error)

	// GetFeature retrieves a feature.
	GetFeature(ctx context.Context, clusterID uint, featureName string) (Feature, error)

	// SaveFeature persists a feature.
	SaveFeature(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec, status string) error

	// UpdateFeatureStatus updates the status of a feature.
	UpdateFeatureStatus(ctx context.Context, clusterID uint, featureName string, status string) error

	// UpdateFeatureSpec updates the spec of a feature.
	UpdateFeatureSpec(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error

	// DeleteFeature deletes a feature.
	DeleteFeature(ctx context.Context, clusterID uint, featureName string) error
}

// IsFeatureNotFoundError returns true when the specified error is a "feature not found" error
func IsFeatureNotFoundError(err error) bool {
	var notFoundErr interface {
		FeatureNotFound() bool
	}
	return errors.As(err, &notFoundErr) && notFoundErr.FeatureNotFound()
}

// FeatureManager is a collection of feature specific methods that are used synchronously when responding to feature related requests.
type FeatureManager interface {
	FeatureOutputProducer
	FeatureSpecValidator
	FeatureSpecPreparer

	// Name returns the feature's name.
	Name() string
}

// FeatureOutputProducer defines how to produce a cluster feature's output.
type FeatureOutputProducer interface {
	// GetOutput returns a cluster feature's output.
	GetOutput(ctx context.Context, clusterID uint) (FeatureOutput, error)
}

// FeatureSpecValidator defines how to validate a feature specification
type FeatureSpecValidator interface {
	// ValidateSpec validates a feature specification.
	ValidateSpec(ctx context.Context, spec FeatureSpec) error
}

// IsInputValidationError returns true if the error is an input validation error
func IsInputValidationError(err error) bool {
	var inputValidationError interface {
		InputValidationError() bool
	}
	return errors.As(err, &inputValidationError) && inputValidationError.InputValidationError()
}

// InvalidFeatureSpecError is returned when a feature specification fails the validation.
type InvalidFeatureSpecError struct {
	FeatureName string
	Problem     string
}

func (e InvalidFeatureSpecError) Error() string {
	return "invalid feature spec: " + e.Problem
}

// Details returns the error's details
func (e InvalidFeatureSpecError) Details() []interface{} {
	return []interface{}{"feature", e.FeatureName}
}

// InputValidationError returns true since InputValidationError is an input validation error
func (InvalidFeatureSpecError) InputValidationError() bool {
	return true
}

// FeatureSpecPreparer defines how a feature specification is prepared before it's sent to be applied
type FeatureSpecPreparer interface {
	// PrepareSpec makes certain preparations to the spec before it's sent to be applied.
	// For example it rewrites the secret ID to it's internal representation, fills in defaults, etc.
	PrepareSpec(ctx context.Context, spec FeatureSpec) (FeatureSpec, error)
}

// FeatureOperationDispatcher dispatches cluster feature operations asynchronously.
type FeatureOperationDispatcher interface {
	// DispatchApply starts applying a desired state for a cluster feature asynchronously.
	DispatchApply(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error

	// DispatchDeactivate starts deactivating a cluster feature asynchronously.
	DispatchDeactivate(ctx context.Context, clusterID uint, featureName string) error
}

// FeatureOperator defines the operations that can be applied to a cluster feature.
type FeatureOperator interface {
	// Apply applies a desired state for a feature on the given cluster.
	Apply(ctx context.Context, clusterID uint, spec FeatureSpec) error

	// Deactivate deactivates a feature on the given cluster.
	Deactivate(ctx context.Context, clusterID uint) error

	// Name returns the feature's name.
	Name() string
}

// ClusterService provides a thin access layer to clusters.
type ClusterService interface {
	// CheckClusterReady checks whether the cluster is ready for features (eg.: exists and it's running). If the cluster is not ready, a ClusterIsNotReadyError should be returned.
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
