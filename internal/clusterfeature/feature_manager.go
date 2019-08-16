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

	"github.com/banzaicloud/pipeline/internal/common"
)

// FeatureManager operations in charge for applying features to the cluster.
type FeatureManager interface {
	// Details returns feature details.
	Details(ctx context.Context, clusterID uint) (*Feature, error)

	// Name returns the name of the managed feature.
	Name() string

	// Deploys and activates a feature on the given cluster
	Activate(ctx context.Context, clusterID uint, spec FeatureSpec) error

	// ValidateSpec
	ValidateSpec(ctx context.Context, spec FeatureSpec) error

	// Removes feature from the given cluster
	Deactivate(ctx context.Context, clusterID uint) error

	// Updates a feature on the given cluster
	Update(ctx context.Context, clusterID uint, spec FeatureSpec) error
}

// InvalidFeatureSpecError is returned when a feature specification fails the validation.
type InvalidFeatureSpecError struct {
	FeatureName string
	Problem     string
}

func (e InvalidFeatureSpecError) Error() string {
	return "invalid feature spec: " + e.Problem
}

func (e InvalidFeatureSpecError) Details() []interface{} {
	return []interface{}{"feature", e.FeatureName}
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

func (e ClusterIsNotReadyError) Details() []interface{} {
	return []interface{}{"clusterId", e.ClusterID}
}

func (e ClusterIsNotReadyError) ShouldRetry() bool {
	return true
}

type syncFeatureManager struct {
	FeatureManager
	featureRepository FeatureRepository
	logger            common.Logger
}

// NewSyncFeatureManager wraps a feature manager and adds synchronous behaviour.
func NewSyncFeatureManager(
	featureManager FeatureManager,
	featureRepository FeatureRepository,
	logger common.Logger,
) FeatureManager {
	return &syncFeatureManager{
		FeatureManager:    featureManager,
		featureRepository: featureRepository,
		logger:            logger,
	}
}

// FeatureAlreadyActivatedError is returned when a feature is already activated.
type FeatureAlreadyActivatedError struct {
	FeatureName string
}

func (FeatureAlreadyActivatedError) Error() string {
	return "feature already activated"
}

func (e FeatureAlreadyActivatedError) Details() []interface{} {
	return []interface{}{"feature", e.FeatureName}
}

func (m *syncFeatureManager) Activate(ctx context.Context, clusterID uint, spec FeatureSpec) error {
	logger := m.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": m.Name()})

	if err := m.FeatureManager.Activate(ctx, clusterID, spec); err != nil {
		const msg = "cluster feature activation failed"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	if _, err := m.featureRepository.UpdateFeatureStatus(ctx, clusterID, m.Name(), FeatureStatusActive); err != nil {
		const msg = "failed to update feature status"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	return nil
}

// FeatureNotActiveError is returned when a feature is already activated.
type FeatureNotActiveError struct {
	FeatureName string
}

func (FeatureNotActiveError) Error() string {
	return "feature is not active"
}

func (e FeatureNotActiveError) Details() []interface{} {
	return []interface{}{"feature", e.FeatureName}
}

func (m *syncFeatureManager) Deactivate(ctx context.Context, clusterID uint) error {
	logger := m.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": m.Name()})

	if err := m.FeatureManager.Deactivate(ctx, clusterID); err != nil {
		// The feature's status is uncertain, so we make it inactive.
		const msg = "cluster feature deactivation failed"
		logger.Debug(msg, map[string]interface{}{"error": err.Error()})
		return errors.WrapIf(err, msg)
	}

	if err := m.featureRepository.DeleteFeature(ctx, clusterID, m.Name()); err != nil {
		const msg = "failed to delete feature from repository"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	return nil
}

func (m *syncFeatureManager) Update(ctx context.Context, clusterID uint, spec FeatureSpec) error {
	logger := m.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": m.Name()})

	if err := m.FeatureManager.Update(ctx, clusterID, spec); err != nil {
		const msg = "cluster feature update failed"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	if _, err := m.featureRepository.UpdateFeatureStatus(ctx, clusterID, m.Name(), FeatureStatusActive); err != nil {
		const msg = "failed to update feature status"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	return nil
}
