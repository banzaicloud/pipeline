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

// FeatureSpec contains the input parameters for a feature.
type FeatureSpec = map[string]interface{}

// Feature represents the internal state of a cluster feature.
type Feature struct {
	Name   string                 `json:"name"`
	Spec   FeatureSpec            `json:"spec"`
	Output map[string]interface{} `json:"output"`
	Status string                 `json:"status"`
}

// Feature status constants
const (
	FeatureStatusPending = "PENDING"
	FeatureStatusActive  = "ACTIVE"
)

// FeatureService manages features on Kubernetes clusters.
type FeatureService struct {
	featureRegistry   FeatureRegistry
	featureRepository FeatureRepository

	logger common.Logger
}

// FeatureRegistry contains feature managers.
// It returns an error if the feature cannot be found.
type FeatureRegistry interface {
	// GetFeatureManager retrieves a feature manager.
	GetFeatureManager(featureName string) (FeatureManager, error)
}

// FeatureRepository manages feature state.
type FeatureRepository interface {
	// Retrieves features for a given cluster.
	GetFeatures(ctx context.Context, clusterID uint) ([]Feature, error)

	// GetFeature retrieves a feature.
	GetFeature(ctx context.Context, clusterID uint, featureName string) (*Feature, error)

	// SaveFeature persists feature state.
	SaveFeature(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error

	// Updates the status of a feature.
	UpdateFeatureStatus(ctx context.Context, clusterID uint, featureName string, status string) (*Feature, error)

	// Updates the spec of a feature.
	UpdateFeatureSpec(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) (*Feature, error)

	// DeleteFeature deletes a feature.
	DeleteFeature(ctx context.Context, clusterID uint, featureName string) error
}

// NewFeatureService returns a new FeatureService instance.
func NewFeatureService(
	featureRegistry FeatureRegistry,
	featureRepository FeatureRepository,
	logger common.Logger,
) *FeatureService {
	return &FeatureService{
		featureRegistry:   featureRegistry,
		featureRepository: featureRepository,

		logger: logger.WithFields(map[string]interface{}{"component": "cluster-feature"}),
	}
}

// List lists the activated features and their details.
func (s *FeatureService) List(ctx context.Context, clusterID uint) ([]Feature, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID})
	logger.Info("listing features")

	features, err := s.featureRepository.GetFeatures(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to retrieve features", "clusterId", clusterID)
	}

	// TODO(laszlop): fetch details by feature managers? (eg. output is not static information)

	logger.Info("features successfully listed")

	return features, nil
}

// Details returns the details of an activated feature.
func (s *FeatureService) Details(ctx context.Context, clusterID uint, featureName string) (*Feature, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	logger.Info("retrieving feature details")

	featureManager, err := s.featureRegistry.GetFeatureManager(featureName)
	if err != nil {
		return nil, err
	}

	feature, err := featureManager.Details(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	logger.Info("successfully retrieved feature details")

	return feature, nil
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

// Activate activates a feature.
func (s *FeatureService) Activate(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	logger.Info("activating feature")

	feature, err := s.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		return nil
	}

	if feature != nil {
		logger.Debug("feature cannot be activated: it's not inactive", map[string]interface{}{
			"status": feature.Status,
		})

		return FeatureAlreadyActivatedError{FeatureName: featureName}
	}

	featureManager, err := s.featureRegistry.GetFeatureManager(featureName)
	if err != nil {
		return err
	}

	logger.Debug("validating feature specification")
	if err := featureManager.ValidateSpec(ctx, spec); err != nil {
		return InvalidFeatureSpecError{FeatureName: featureName, Problem: err.Error()}
	}

	err = s.featureRepository.SaveFeature(ctx, clusterID, featureName, spec)
	if err != nil {
		return err
	}

	err = featureManager.Activate(ctx, clusterID, spec)
	if err != nil {
		// Deletion is best effort here, activation failed anyway
		_ = s.featureRepository.DeleteFeature(ctx, clusterID, featureName)

		return err
	}

	if _, err := s.featureRepository.UpdateFeatureStatus(ctx, clusterID, featureName, FeatureStatusActive); err != nil {

		return err
	}

	logger.Info("feature activation request processed successfully")

	return nil
}

// Deactivate deactivates a feature.
func (s *FeatureService) Deactivate(ctx context.Context, clusterID uint, featureName string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	logger.Info("deactivating feature")

	feature, err := s.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		return nil
	}

	if feature == nil {
		logger.Info("feature is not activated")

		return FeatureNotActiveError{FeatureName: featureName}
	}

	featureManager, err := s.featureRegistry.GetFeatureManager(featureName)
	if err != nil {
		return err
	}

	if err := featureManager.Deactivate(ctx, clusterID); err != nil {
		logger.Debug("failed to deactivate feature")

		return errors.WrapIfWithDetails(err, "failed to deactivate feature", "clusterID", clusterID, "feature", featureName)
	}

	logger.Info("feature deactivation request processed successfully")

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

// Update updates a feature.
func (s *FeatureService) Update(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterID": clusterID, "feature": featureName})
	logger.Info("updating feature")

	feature, err := s.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		return nil
	}

	if feature == nil {
		logger.Debug("feature cannot be updated: it is not active yet")

		return errors.WithStack(FeatureNotActiveError{FeatureName: featureName})
	}

	featureManager, err := s.featureRegistry.GetFeatureManager(featureName)
	if err != nil {
		logger.Debug("failed to get feature manager")

		return err
	}

	logger.Debug("validating feature specification")
	if err := featureManager.ValidateSpec(ctx, spec); err != nil {

		return InvalidFeatureSpecError{FeatureName: featureName, Problem: err.Error()}
	}

	if err := featureManager.Update(ctx, clusterID, spec); err != nil {
		logger.Debug("failed to update feature")

		return errors.WrapIfWithDetails(err, "failed to update feature", "clusterID", clusterID, "feature", featureName)
	}

	if _, err := s.featureRepository.UpdateFeatureStatus(ctx, clusterID, featureName, FeatureStatusActive); err != nil {

		return err
	}

	logger.Info("feature updated successfully")

	return nil
}
