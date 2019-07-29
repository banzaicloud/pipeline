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

type ClusterSecretStore interface {
	// GetSecret gets a secret for a cluster if exists
	GetSecret(ctx context.Context, clusterID uint, secretID string) (map[string]string, error)

	GetSecretByName(ctx context.Context, clusterID uint, secretName string) (map[string]string, error)
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
	// IsClusterReady checks whether the cluster is ready for features (eg.: exists and it's running).
	IsClusterReady(ctx context.Context, clusterID uint) (bool, error)
}

type syncFeatureManager struct {
	featureManager    FeatureManager
	clusterService    ClusterService
	featureRepository FeatureRepository
}

// NewSyncFeatureManager wraps a feature manager and adds synchronous behaviour.
func NewSyncFeatureManager(
	featureManager FeatureManager,
	clusterService ClusterService,
	featureRepository FeatureRepository,
) FeatureManager {
	return &syncFeatureManager{
		featureManager:    featureManager,
		clusterService:    clusterService,
		featureRepository: featureRepository,
	}
}

func (m *syncFeatureManager) Details(ctx context.Context, clusterID uint) (*Feature, error) {
	return m.featureManager.Details(ctx, clusterID)
}

func (m *syncFeatureManager) Name() string {
	return m.featureManager.Name()
}

func (m *syncFeatureManager) Activate(ctx context.Context, clusterID uint, spec FeatureSpec) error {
	if err := m.isClusterReady(ctx, clusterID); err != nil {
		return err
	}

	return m.featureManager.Activate(ctx, clusterID, spec)
}

func (m *syncFeatureManager) ValidateSpec(ctx context.Context, spec FeatureSpec) error {
	return m.featureManager.ValidateSpec(ctx, spec)
}

func (m *syncFeatureManager) Deactivate(ctx context.Context, clusterID uint) error {
	if err := m.isClusterReady(ctx, clusterID); err != nil {
		return err
	}

	if err := m.isFeaturePending(ctx, clusterID, m.featureManager.Name()); err != nil {
		return err
	}

	err := m.featureManager.Deactivate(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := m.featureRepository.DeleteFeature(ctx, clusterID, m.Name()); err != nil {
		return err
	}

	return nil
}

func (m *syncFeatureManager) Update(ctx context.Context, clusterID uint, spec FeatureSpec) error {
	if err := m.isClusterReady(ctx, clusterID); err != nil {
		return err
	}

	if err := m.isFeaturePending(ctx, clusterID, m.featureManager.Name()); err != nil {
		return err
	}

	if _, err := m.featureRepository.UpdateFeatureStatus(ctx, clusterID, m.featureManager.Name(), FeatureStatusPending); err != nil {

		return err
	}

	return m.featureManager.Update(ctx, clusterID, spec)
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

func (m *syncFeatureManager) isClusterReady(ctx context.Context, clusterID uint) error {
	ready, err := m.clusterService.IsClusterReady(ctx, clusterID)
	if err != nil {
		return err
	}

	if !ready {
		return errors.WithStack(ClusterIsNotReadyError{ClusterID: clusterID})
	}

	return nil
}

func (m *syncFeatureManager) isFeaturePending(ctx context.Context, clusterID uint, featureName string) error {
	feature, err := m.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		return nil
	}

	if feature == nil || feature.Status == FeatureStatusPending {
		return errors.WithStack(FeatureNotActiveError{FeatureName: featureName})
	}

	return nil
}
