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

// MakeFeatureService returns a new FeatureService instance.
func MakeFeatureService(
	featureOperationDispatcher FeatureOperationDispatcher,
	featureManagerRegistry FeatureManagerRegistry,
	featureRepository FeatureRepository,
	logger common.Logger,
) FeatureService {
	return FeatureService{
		featureOperationDispatcher: featureOperationDispatcher,
		featureManagerRegistry:     featureManagerRegistry,
		featureRepository:          featureRepository,
		logger:                     logger.WithFields(map[string]interface{}{"component": "cluster-feature"}),
	}
}

// FeatureService implements a cluster feature service
type FeatureService struct {
	featureOperationDispatcher FeatureOperationDispatcher
	featureManagerRegistry     FeatureManagerRegistry
	featureRepository          FeatureRepository
	logger                     common.Logger
}

// List lists the activated features and their details.
func (s FeatureService) List(ctx context.Context, clusterID uint) ([]Feature, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID})
	logger.Info("listing features")

	features, err := s.featureRepository.GetFeatures(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to retrieve features", "clusterId", clusterID)
	}

	for i, f := range features {

		featureManager, err := s.featureManagerRegistry.GetFeatureManager(f.Name)
		if err != nil {

			return nil, err
		}

		output, err := featureManager.GetOutput(ctx, clusterID, f.Spec)
		if err != nil {

			return nil, err
		}

		features[i].Output = merge(f.Output, output)
	}

	logger.Info("features successfully listed")

	return features, nil
}

// Details returns the details of an activated feature.
func (s FeatureService) Details(ctx context.Context, clusterID uint, featureName string) (Feature, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	logger.Info("processing feature details request")

	// TODO: check cluster ID?

	logger.Debug("retrieving feature from repository")
	feature, err := s.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		const msg = "failed to retrieve feature from repository"
		logger.Debug(msg)
		return feature, errors.WrapIf(err, msg)
	}

	logger.Debug("retrieving feature manager")
	featureManager, err := s.featureManagerRegistry.GetFeatureManager(featureName)
	if err != nil {
		const msg = "failed to retrieve feature manager"
		logger.Debug(msg)
		return feature, errors.WrapIf(err, msg)
	}

	logger.Debug("retieving feature output")
	output, err := featureManager.GetOutput(ctx, clusterID, feature.Spec)
	if err != nil {
		const msg = "failed to retieve feature output"
		logger.Debug(msg)
		return feature, errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "feature", featureName)
	}

	feature.Output = merge(feature.Output, output)

	logger.Info("feature details request processed successfully")

	return feature, nil
}

// Activate activates a feature.
func (s FeatureService) Activate(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	logger.Info("processing feature activation request")

	// TODO: check cluster ID?

	logger.Debug("retieving feature manager")
	featureManager, err := s.featureManagerRegistry.GetFeatureManager(featureName)
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

	logger.Debug("preparing feature specification")
	preparedSpec, err := featureManager.PrepareSpec(ctx, spec)
	if err != nil {
		const msg = "failed to prepare feature specification"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("starting feature activation")
	if err := s.featureOperationDispatcher.DispatchApply(ctx, clusterID, featureName, preparedSpec); err != nil {
		const msg = "failed to start feature activation"
		logger.Debug(msg)
		return errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "feature", featureName)
	}

	logger.Debug("persisting feature")
	if err := s.featureRepository.SaveFeature(ctx, clusterID, featureName, spec, FeatureStatusPending); err != nil {
		const msg = "failed to persist feature"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Info("feature activation request processed successfully")

	return nil
}

// Deactivate deactivates a feature.
func (s FeatureService) Deactivate(ctx context.Context, clusterID uint, featureName string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	logger.Info("processing feature deactivation request")

	// TODO: check cluster ID?

	logger.Debug("checking feature name")
	if _, err := s.featureManagerRegistry.GetFeatureManager(featureName); err != nil {
		const msg = "failed to retrieve feature manager"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("get feature details")
	f, err := s.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		const msg = "failed to retrieve feature details"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("starting feature deactivation")
	if err := s.featureOperationDispatcher.DispatchDeactivate(ctx, clusterID, featureName, f.Spec); err != nil {
		const msg = "failed to start feature deactivation"
		logger.Debug(msg)
		return errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "feature", featureName)
	}

	logger.Debug("updating feature status")
	if err := s.featureRepository.UpdateFeatureStatus(ctx, clusterID, featureName, FeatureStatusPending); err != nil {
		if !IsFeatureNotFoundError(err) {
			const msg = "failed to update feature status"
			logger.Debug(msg)
			return errors.WrapIf(err, msg)
		}

		logger.Info("feature is already inactive")
	}

	logger.Info("feature deactivation request processed successfully")

	return nil
}

// Update updates a feature.
func (s FeatureService) Update(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterID": clusterID, "feature": featureName})
	logger.Info("processing feature update request")

	// TODO: check cluster ID?

	logger.Debug("retieving feature manager")
	featureManager, err := s.featureManagerRegistry.GetFeatureManager(featureName)
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

	logger.Debug("preparing feature specification")
	preparedSpec, err := featureManager.PrepareSpec(ctx, spec)
	if err != nil {
		const msg = "failed to prepare feature specification"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("starting feature update")
	if err := s.featureOperationDispatcher.DispatchApply(ctx, clusterID, featureName, preparedSpec); err != nil {
		const msg = "failed to start feature update"
		logger.Debug(msg)
		return errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "feature", featureName)
	}

	logger.Debug("persisting feature")
	if err := s.featureRepository.SaveFeature(ctx, clusterID, featureName, spec, FeatureStatusPending); err != nil {
		const msg = "failed to persist feature"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Info("feature updated successfully")

	return nil
}

func merge(this map[string]interface{}, that map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(this)+len(that))
	for k, v := range this {
		result[k] = v
	}
	for k, v := range that {
		result[k] = v
	}
	return result
}
