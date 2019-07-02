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
	logger            logur.Logger
	clusterService    ClusterService
	featureRepository FeatureRepository
	featureManager    FeatureManager
	featureSelector   FeatureSelector
}

// ClusterService provides a thin access layer to clusters.
type ClusterService interface {
	// GetCluster retrieves the cluster representation based on the cluster identifier
	// TODO: this is an implementation detail for the helm installer. Remove it from here/relocate to another interface.
	GetCluster(ctx context.Context, clusterID uint) (Cluster, error)

	// IsClusterReady checks whether the cluster is ready for features (eg.: exists and it's running).
	IsClusterReady(ctx context.Context, clusterID uint) (bool, error)
}

// Cluster represents a Kubernetes cluster.
// TODO: this is an implementation detail for the helm installer. Remove it from here/relocate to another interface.
type Cluster interface {
	GetID() uint
	GetOrganizationName() string
	GetKubeConfig() ([]byte, error)
}

// FeatureRepository collects persistence related operations.
type FeatureRepository interface {
	// SaveFeature persists the feature into the persistent storage
	SaveFeature(ctx context.Context, clusterId uint, feature Feature) (uint, error)

	// GetFeature retrieves the feature from the persistent storage
	GetFeature(ctx context.Context, clusterId uint, featureName string) (*Feature, error)

	// Updates the status of the feature in the persistent storage
	UpdateFeatureStatus(ctx context.Context, clusterId uint, featureName string, status string) (*Feature, error)

	// Updates the status of the feature in the persistent storage
	UpdateFeatureSpec(ctx context.Context, clusterId uint, featureName string, spec map[string]interface{}) (*Feature, error)

	// DeleteFeature deletes the feature from the persistent storage
	DeleteFeature(ctx context.Context, clusterId uint, featureName string) error

	// Retrieves features for a given cluster
	ListFeatures(ctx context.Context, clusterId uint) ([]*Feature, error)
}

// FeatureManager operations in charge for applying features to the cluster.
type FeatureManager interface {
	// Deploys and activates a feature on the given cluster
	Activate(ctx context.Context, clusterId uint, feature Feature) (string, error)

	// Removes feature from the given cluster
	Deactivate(ctx context.Context, clusterId uint, feature Feature) error

	// Updates a feature on the given cluster
	Update(ctx context.Context, clusterId uint, feature Feature) (string, error)
}

// NewClusterFeatureService returns a new FeatureService instance.
func NewClusterFeatureService(
	logger logur.Logger,
	clusterService ClusterService,
	featureRepository FeatureRepository,
	featureManager FeatureManager,
) *FeatureService {
	return &FeatureService{
		logger:            logger,
		clusterService:    clusterService,
		featureRepository: featureRepository,
		featureManager:    featureManager,
		featureSelector:   NewFeatureSelector(logger),
	}
}

func (s *FeatureService) Activate(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error {
	s.logger.Info("activate feature", map[string]interface{}{"feature": featureName})

	selectedFeature, err := s.featureSelector.SelectFeature(ctx, Feature{Name: featureName, Spec: spec})
	if err != nil {
		return newFeatureSelectionError(featureName)
	}

	if _, err := s.featureRepository.GetFeature(ctx, clusterID, featureName); err == nil {
		s.logger.Debug("feature exists", map[string]interface{}{"clusterId": clusterID, "feature": featureName})

		return newFeatureExistsError(featureName)
	}

	ready, err := s.clusterService.IsClusterReady(ctx, clusterID)
	if err != nil {
		return emperror.Wrap(err, "could not access cluster")
	}

	if !ready {
		s.logger.Debug("cluster not ready", map[string]interface{}{"clusterId": clusterID})

		return newClusterNotReadyError(featureName)
	}

	// TODO: save feature name and spec (pending status?)
	if _, err := s.featureRepository.SaveFeature(ctx, clusterID, *selectedFeature); err != nil {
		return emperror.WrapWith(err, "failed to persist feature", "clusterId", clusterID, "feature", featureName)
	}

	// delegate the task of "deploying" the feature to the manager
	if _, err := s.featureManager.Activate(ctx, clusterID, *selectedFeature); err != nil {
		return emperror.WrapWith(err, "failed to activate feature", "clusterId", clusterID, "feature", featureName)
	}

	// TODO: this should be done asynchronously
	if _, err := s.featureRepository.UpdateFeatureStatus(ctx, clusterID, featureName, FeatureStatusActive); err != nil {
		return emperror.WrapWith(err, "failed to update feature status", "clusterId", clusterID, "feature", featureName)
	}

	s.logger.Info("feature successfully activated ", map[string]interface{}{"clusterId": clusterID, "feature": featureName})

	return nil
}

func (s *FeatureService) Deactivate(ctx context.Context, clusterID uint, featureName string) error {
	s.logger.Info("deactivating feature", map[string]interface{}{"clusterID": clusterID, "feature": featureName})

	var feature *Feature

	_, err := s.featureSelector.SelectFeature(ctx, Feature{Name: featureName})
	if err != nil {
		return newFeatureSelectionError(featureName)
	}

	if feature, err = s.featureRepository.GetFeature(ctx, clusterID, featureName); err != nil {
		s.logger.Debug("feature could not be found", map[string]interface{}{"clusterId": clusterID, "feature": featureName})

		return newFeatureNotFoundError(featureName)
	}

	ready, err := s.clusterService.IsClusterReady(ctx, clusterID)
	if err != nil {
		return emperror.Wrap(err, "could not access cluster")
	}

	if !ready {
		s.logger.Debug("cluster not ready", map[string]interface{}{"clusterId": clusterID})

		return newClusterNotReadyError(featureName)
	}

	if err := s.featureManager.Deactivate(ctx, clusterID, *feature); err != nil {
		s.logger.Debug("failed to deactivate feature on cluster", map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	}

	if err := s.featureRepository.DeleteFeature(ctx, clusterID, featureName); err != nil {
		return emperror.WrapWith(err, "failed to delete feature", "clusterID", clusterID, "feature", featureName)
	}

	s.logger.Info("successfully deactivated feature", map[string]interface{}{"clusterID": clusterID, "feature": featureName})
	return nil
}

func (s *FeatureService) Details(ctx context.Context, clusterID uint, featureName string) (*Feature, error) {
	s.logger.Info("retrieving feature details", map[string]interface{}{"clusterid": clusterID, "feature": featureName})

	fd, err := s.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		return nil, newFeatureNotFoundError(featureName)
	}

	s.logger.Info("successfully retrieved feature details", map[string]interface{}{"clusterid": clusterID, "feature": featureName})
	return fd, nil
}

func (s *FeatureService) List(ctx context.Context, clusterID uint) ([]Feature, error) {
	s.logger.Info("retrieve features", map[string]interface{}{"clusterid": clusterID})
	var (
		featurePtrs []*Feature
		features    []Feature
		err         error
	)
	if featurePtrs, err = s.featureRepository.ListFeatures(ctx, clusterID); err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve features")
	}

	for _, fp := range featurePtrs {
		features = append(features, *fp)
	}
	s.logger.Info("successfully retrieved features", map[string]interface{}{"clusterid": clusterID})
	return features, nil
}

func (s *FeatureService) Update(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error {
	s.logger.Info("updating feature spec", map[string]interface{}{"clusterID": clusterID, "feature": featureName})

	// todo
	//s.featureManager.Update(ctx, clusterID)

	if _, err := s.featureRepository.UpdateFeatureSpec(ctx, clusterID, featureName, spec); err != nil {
		return emperror.WrapWith(err, "failed to update feature spec", "clusterID", clusterID, "feature", featureName)
	}

	s.logger.Info("successfully updated feature spec", map[string]interface{}{"clusterID": clusterID, "feature": featureName})
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
	errorFeatureExists    = "feature already exists"
	errorFeatureSelection = "could not select feature"
	errorClusterNotReady  = "cluster is not ready"
	errorFeatureNotFound  = "feature could not be found"
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

type featureSelectionError struct {
	featureError
}

func newFeatureSelectionError(featureName string) error {
	return featureSelectionError{featureError{
		featureName: featureName,
		msg:         errorFeatureSelection,
	}}
}

type featureNotFoundError struct {
	featureError
}

func newFeatureNotFoundError(featureName string) error {
	return featureNotFoundError{featureError{
		featureName: featureName,
		msg:         errorFeatureNotFound,
	}}
}
