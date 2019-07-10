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
	"fmt"

	"emperror.dev/emperror"
	"github.com/goph/logur"
)

// Feature represents the internal state of a cluster feature.
type Feature struct {
	Name   string                 `json:"name"`
	Spec   map[string]interface{} `json:"spec"`
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
	logger                 logur.Logger
	featureManagerRegistry FeatureManagerRegistry
	featureLister          FeatureLister
}

// FeatureRepository collects persistence related operations.
type FeatureRepository interface {
	// SaveFeature persists the feature into the persistent storage
	SaveFeature(ctx context.Context, clusterID uint, featureName string, featureSpec map[string]interface{}) (uint, error)

	// GetFeature retrieves the feature from the persistent storage
	GetFeature(ctx context.Context, clusterID uint, featureName string) (*Feature, error)

	// Updates the status of the feature in the persistent storage
	UpdateFeatureStatus(ctx context.Context, clusterID uint, featureName string, status string) (*Feature, error)

	// Updates the spec of the feature in the persistent storage
	UpdateFeatureSpec(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) (*Feature, error)

	// DeleteFeature deletes the feature from the persistent storage
	DeleteFeature(ctx context.Context, clusterID uint, featureName string) error

	// Retrieves features for a given cluster
	ListFeatures(ctx context.Context, clusterID uint) ([]Feature, error)
}

// FeatureManager operations in charge for applying features to the cluster.
type FeatureManager interface {
	// Deploys and activates a feature on the given cluster
	Activate(ctx context.Context, clusterID uint, feature Feature) error

	// Removes feature from the given cluster
	Deactivate(ctx context.Context, clusterID uint, featureName string) error

	// Updates a feature on the given cluster
	Update(ctx context.Context, clusterID uint, feature Feature) error

	// Validate validates the feature, chsecks its prerequisites
	Validate(ctx context.Context, clusterID uint, featureName string, featureSpec map[string]interface{}) error

	// Details returns feature details
	Details(ctx context.Context, clusterID uint, featureName string) (*Feature, error)
}

// FeatureLister component interface for listing features
type FeatureLister interface {
	// List retrieves the list of features for the given clusterid
	List(ctx context.Context, clusterID uint) ([]Feature, error)
}

// NewClusterFeatureService returns a new FeatureService instance.
func NewClusterFeatureService(
	logger logur.Logger,
	featureLister FeatureLister,
	featureManagerRegistry FeatureManagerRegistry,
) *FeatureService {
	return &FeatureService{
		logger:                 logur.WithFields(logger, map[string]interface{}{"cluster-feature-service": "comp"}),
		featureManagerRegistry: featureManagerRegistry,
		featureLister:          featureLister,
	}
}

func (s *FeatureService) Activate(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error {
	log := logur.WithFields(s.logger, map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	log.Info("activate feature")

	var (
		featureManager FeatureManager
		err            error
	)

	featureManager, err = s.featureManagerRegistry.GetFeatureManager(ctx, featureName)
	if err != nil {

		return newUnsupportedFeatureError(featureName)
	}

	if err = featureManager.Validate(ctx, clusterID, featureName, spec); err != nil {
		log.Debug("feature validation failed")

		return emperror.Wrap(err, "failed to validate feature")
	}

	// delegate the task of "deploying" the feature to the manager
	if err = featureManager.Activate(ctx, clusterID, Feature{Name: featureName, Spec: spec}); err != nil {
		log.Error("failed to activate feature")

		return emperror.WrapWith(err, "failed to activate feature", "clusterId", clusterID, "feature", featureName)
	}

	log.Info("feature successfully activated")

	return nil
}

func (s *FeatureService) Deactivate(ctx context.Context, clusterID uint, featureName string) error {
	mLogger := logur.WithFields(s.logger, map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	mLogger.Info("deactivating feature")


	featureManager, err := s.featureManagerRegistry.GetFeatureManager(ctx, featureName)
	if err != nil {
		mLogger.Debug("failed to get feature manager")

		return newUnsupportedFeatureError(featureName)
	}

	if err = featureManager.Validate(ctx, clusterID, featureName, nil); err != nil {
		mLogger.Debug("feature validation failed")

		return emperror.Wrap(err, "failed to activate feature")
	}

	if err := featureManager.Deactivate(ctx, clusterID, featureName); err != nil {
		mLogger.Debug("failed to deactivate feature on cluster")

		return emperror.WrapWith(err, "failed to deactivate feature", "clusterID", clusterID, "feature", featureName)
	}

	mLogger.Info("successfully deactivated feature")
	return nil
}

func (s *FeatureService) Details(ctx context.Context, clusterID uint, featureName string) (*Feature, error) {

	mLogger := logur.WithFields(s.logger, map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	mLogger.Info("retrieving feature details ...")

	var (
		feature        *Feature
		featureManager FeatureManager
		err            error
	)

	if featureManager, err = s.featureManagerRegistry.GetFeatureManager(ctx, featureName); err != nil {
		mLogger.Info("failed to get feature manager")

		return nil, newUnsupportedFeatureError(featureName)
	}

	if feature, err = featureManager.Details(ctx, clusterID, featureName); err != nil {
		mLogger.Info("failed to retrieve feature details")
		// wrap the error here?
		return nil, err
	}

	mLogger.Info("successfully retrieved feature details")
	return feature, nil
}

func (s *FeatureService) List(ctx context.Context, clusterID uint) ([]Feature, error) {

	mLogger := logur.WithFields(s.logger, map[string]interface{}{"clusterId": clusterID})
	mLogger.Info("retrieving features ...")

	var (
		features []Feature
		err      error
	)

	if features, err = s.featureLister.List(ctx, clusterID); err != nil {
		mLogger.Info("failed to retrieve features")

		return nil, emperror.Wrap(err, "failed to retrieve features")
	}

	mLogger.Info("features successfully retrieved")
	return features, nil
}

func (s *FeatureService) Update(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error {

	mLogger := logur.WithFields(s.logger, map[string]interface{}{"clusterID": clusterID, "feature": featureName})
	mLogger.Info("updating feature spec...")

	featureManager, err := s.featureManagerRegistry.GetFeatureManager(ctx, featureName)
	if err != nil {
		mLogger.Debug("failed to get feature manager")

		return newUnsupportedFeatureError(featureName)
	}

	if err = featureManager.Validate(ctx, clusterID, featureName, spec); err != nil {
		mLogger.Debug("feature validation failed")

		return emperror.Wrap(err, "failed to validate feature")
	}

	if err := featureManager.Update(ctx, clusterID, Feature{Name: featureName, Spec: spec}); err != nil {
		mLogger.Debug("failed to update feature")

		return emperror.WrapWith(err, "failed to update feature", "clusterID", clusterID, "feature", featureName)
	}

	mLogger.Info("successfully updated feature spec")
	return nil
}

// featureError "Business" error type
type featureError struct {
	msg         string
	featureName string
}

func (e featureError) Error() string {

	return fmt.Sprintf("Feature: %s, Message: %s", e.featureName, e.msg)
}

func (e featureError) FeatureName() string {
	return e.featureName
}

func (e featureError) Context() []string {

	return []string{"featureName", e.featureName}
}

func (e featureError) IsBusinnessError() bool {

	return true
}

const (
	errorFeatureExists      = "feature already exists"
	errorFeatureNotFound    = "feature could not be found"
	errorUnsupportedFeature = "feature is not supported"
	errorClusterNotReady    = "cluster is not ready"
	errorDatabaseAccess     = "could not access the database"
)

type featureExistsError struct {
	featureError
}

func newFeatureExistsError(featureName string) error {
	return featureExistsError{featureError{
		featureName: featureName,
		msg:         errorFeatureExists,
	}}
}

type clusterNotReadyError struct {
	featureError
}

func newClusterNotReadyError(featureName string) error {

	return clusterNotReadyError{featureError{
		featureName: featureName,
		msg:         errorClusterNotReady,
	}}
}

type unsupportedFeatureError struct {
	featureError
}

func newUnsupportedFeatureError(featureName string) error {
	return unsupportedFeatureError{featureError{
		featureName: featureName,
		msg:         errorUnsupportedFeature,
	}}
}

type databaseAccessError struct {
	featureError
}

func newDatabaseAccessError(featureName string) error {
	return databaseAccessError{featureError{
		featureName: featureName,
		msg:         errorDatabaseAccess,
	}}
}

type FeatureNotFoundError struct {
	featureError
}

func newFeatureNotFoundError(featureName string) error {
	return databaseAccessError{featureError{
		featureName: featureName,
		msg:         errorFeatureNotFound,
	}}
}
