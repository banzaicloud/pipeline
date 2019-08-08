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

	// CreateFeature creates a feature.
	CreateFeature(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec, status string) error

	// CreateOrUpdateFeature creates or updates a feature.
	CreateOrUpdateFeature(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec, status string) error

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

	retFeatures := make([]Feature, len(features))
	for i, f := range features {

		featureManager, err := s.featureRegistry.GetFeatureManager(f.Name)
		if err != nil {

			return nil, err
		}

		feature, err := featureManager.Details(ctx, clusterID)
		if err != nil {

			return nil, err
		}

		retFeatures[i] = *feature
	}

	logger.Info("features successfully listed")

	return retFeatures, nil
}

// FeatureNotFoundError is returned when a feature is not found.
type FeatureNotFoundError struct {
	FeatureName string
}

func (FeatureNotFoundError) Error() string {
	return "feature is not found"
}

func (e FeatureNotFoundError) Details() []interface{} {
	return []interface{}{"feature", e.FeatureName}
}

// Details returns the details of an activated feature.
func (s *FeatureService) Details(ctx context.Context, clusterID uint, featureName string) (*Feature, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	logger.Info("processing feature details request")

	// TODO: check cluster ID?

	logger.Debug("retieving feature manager")
	featureManager, err := s.featureRegistry.GetFeatureManager(featureName)
	if err != nil {
		const msg = "failed to retieve feature manager"
		logger.Debug(msg)
		return nil, errors.WrapIf(err, msg)
	}

	logger.Debug("retieving feature details")
	feature, err := featureManager.Details(ctx, clusterID)
	if err != nil {
		const msg = "failed to retieve feature details"
		logger.Debug(msg)
		return nil, errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "feature", featureName)
	}

	logger.Info("feature details request processed successfully")

	return feature, nil
}

// Activate activates a feature.
func (s *FeatureService) Activate(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	logger.Info("processing feature activation request")

	// TODO: check cluster ID?
	_, err := s.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		return nil
	}

	//if feature != nil {
	//	logger.Debug("feature cannot be activated: it's not inactive", map[string]interface{}{
	//		"status": feature.Status,
	//	})
	//
	//	return FeatureAlreadyActivatedError{FeatureName: featureName}
	//}

	logger.Debug("retieving feature manager")
	featureManager, err := s.featureRegistry.GetFeatureManager(featureName)
	if err != nil {
		const msg = "failed to retieve feature manager"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("validating feature specification")
	if err := featureManager.ValidateSpec(ctx, spec); err != nil {
		logger.Debug("feature specification validation failed")
		return InvalidFeatureSpecError{FeatureName: featureName, Problem: err.Error()}
	}

	logger.Debug("activating feature")
	if err := featureManager.Activate(ctx, clusterID, spec); err != nil {
		const msg = "failed to activate feature"
		logger.Debug(msg)
		return errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "feature", featureName)
	}

	logger.Info("feature activation request processed successfully")

	return nil
}

// Deactivate deactivates a feature.
func (s *FeatureService) Deactivate(ctx context.Context, clusterID uint, featureName string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	logger.Info("processing feature deactivation request")

	// TODO: check cluster ID?

	logger.Debug("retieving feature manager")
	featureManager, err := s.featureRegistry.GetFeatureManager(featureName)
	if err != nil {
		const msg = "failed to retieve feature manager"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("deactivating feature")
	if err := featureManager.Deactivate(ctx, clusterID); err != nil {
		logger.Debug("failed to deactivate feature")

		return errors.WrapIfWithDetails(err, "failed to deactivate feature", "clusterID", clusterID, "feature", featureName)
	}

	logger.Info("feature deactivation request processed successfully")

	return nil
}

// Update updates a feature.
func (s *FeatureService) Update(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterID": clusterID, "feature": featureName})
	logger.Info("processing feature update request")

	// TODO: check cluster ID?

	logger.Debug("retieving feature manager")
	featureManager, err := s.featureRegistry.GetFeatureManager(featureName)
	if err != nil {
		const msg = "failed to retieve feature manager"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("validating feature specification")
	if err := featureManager.ValidateSpec(ctx, spec); err != nil {
		logger.Debug("feature specification validation failed")
		return InvalidFeatureSpecError{FeatureName: featureName, Problem: err.Error()}
	}

	logger.Debug("updating feature")
	if err := featureManager.Update(ctx, clusterID, spec); err != nil {
		const msg = "failed to update feature"
		logger.Debug(msg)
		return errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "feature", featureName)
	}

	logger.Info("feature updated successfully")

	return nil
}
