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
	clusterService         ClusterService
	featureRepository      FeatureRepository
	featureManagerRegistry FeatureManagerRegistry
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
	SaveFeature(ctx context.Context, clusterId uint, featureName string, featureSpec map[string]interface{}) (uint, error)

	// GetFeature retrieves the feature from the persistent storage
	GetFeature(ctx context.Context, clusterId uint, featureName string) (*Feature, error)

	// Updates the status of the feature in the persistent storage
	UpdateFeatureStatus(ctx context.Context, clusterId uint, featureName string, status string) (*Feature, error)

	// Updates the status of the feature in the persistent storage
	UpdateFeatureSpec(ctx context.Context, clusterId uint, featureName string, spec map[string]interface{}) (*Feature, error)

	// DeleteFeature deletes the feature from the persistent storage
	DeleteFeature(ctx context.Context, clusterId uint, featureName string) error

	// Retrieves features for a given cluster
	ListFeatures(ctx context.Context, clusterId uint) ([]Feature, error)
}

// FeatureManager operations in charge for applying features to the cluster.
type FeatureManager interface {
	// Deploys and activates a feature on the given cluster
	Activate(ctx context.Context, clusterId uint, feature Feature) error

	// Removes feature from the given cluster
	Deactivate(ctx context.Context, clusterId uint, feature Feature) error

	// Updates a feature on the given cluster
	Update(ctx context.Context, clusterId uint, feature Feature) error
}

// NewClusterFeatureService returns a new FeatureService instance.
func NewClusterFeatureService(
	logger logur.Logger,
	clusterService ClusterService,
	featureRepository FeatureRepository,
	featureManagerRegistry FeatureManagerRegistry,
) *FeatureService {
	return &FeatureService{
		logger:                 logger,
		clusterService:         clusterService,
		featureRepository:      featureRepository,
		featureManagerRegistry: featureManagerRegistry,
	}
}

func (s *FeatureService) Activate(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error {
	log := logur.WithFields(s.logger, map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	log.Info("activate feature")

	featureManager, err := s.featureManagerRegistry.GetFeatureManager(ctx, featureName)
	if err != nil {
		return newUnsupportedFeatureError(featureName)
	}

	if _, err := s.featureRepository.GetFeature(ctx, clusterID, featureName); err == nil {
		log.Debug("feature exists")

		return newFeatureExistsError(featureName)
	}

	ready, err := s.clusterService.IsClusterReady(ctx, clusterID)
	if err != nil {

		return emperror.Wrap(err, "could not access cluster")
	}

	if !ready {
		s.logger.Debug("cluster not ready")

		return newClusterNotReadyError(featureName)
	}

	if _, err := s.featureRepository.SaveFeature(ctx, clusterID, featureName, spec); err != nil {

		return emperror.WrapWith(err, "failed to persist feature", "clusterId", clusterID, "feature", featureName)
	}

	// delegate the task of "deploying" the feature to the manager
	if err := featureManager.Activate(ctx, clusterID, Feature{Name: featureName, Spec: spec}); err != nil {

		return emperror.WrapWith(err, "failed to activate feature", "clusterId", clusterID, "feature", featureName)
	}

	// TODO: this should be done asynchronously
	if _, err := s.featureRepository.UpdateFeatureStatus(ctx, clusterID, featureName, FeatureStatusActive); err != nil {

		return emperror.WrapWith(err, "failed to update feature status", "clusterId", clusterID, "feature", featureName)
	}

	log.Info("feature successfully activated ")

	return nil
}

func (s *FeatureService) Deactivate(ctx context.Context, clusterID uint, featureName string) error {
	log := logur.WithFields(s.logger, map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	log.Info("deactivating feature")

	var feature *Feature

	featureManager, err := s.featureManagerRegistry.GetFeatureManager(ctx, featureName)
	if err != nil {
		return newUnsupportedFeatureError(featureName)
	}

	if feature, err = s.featureRepository.GetFeature(ctx, clusterID, featureName); err != nil {
		log.Debug("feature could not be found")

		return newDatabaseAccessError(featureName)
	}

	ready, err := s.clusterService.IsClusterReady(ctx, clusterID)
	if err != nil {

		return emperror.Wrap(err, "could not access cluster")
	}

	if !ready {
		log.Debug("cluster not ready")

		return newClusterNotReadyError(featureName)
	}

	if err := featureManager.Deactivate(ctx, clusterID, *feature); err != nil {
		log.Debug("failed to deactivate feature on cluster")

		return emperror.WrapWith(err, "failed to deactivate feature", "clusterID", clusterID, "feature", featureName)
	}

	if err := s.featureRepository.DeleteFeature(ctx, clusterID, featureName); err != nil {
		return emperror.WrapWith(err, "failed to delete feature", "clusterID", clusterID, "feature", featureName)
	}

	log.Info("successfully deactivated feature")
	return nil
}

func (s *FeatureService) Details(ctx context.Context, clusterID uint, featureName string) (*Feature, error) {

	log := logur.WithFields(s.logger, map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	log.Info("retrieving feature details")

	fd, err := s.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {

		return nil, newDatabaseAccessError(featureName)
	}

	log.Info("successfully retrieved feature details")
	return fd, nil
}

func (s *FeatureService) List(ctx context.Context, clusterID uint) ([]Feature, error) {

	log := logur.WithFields(s.logger, map[string]interface{}{"clusterId": clusterID})
	log.Info("retrieve features")

	var (
		err error
	)

	if features, err := s.featureRepository.ListFeatures(ctx, clusterID); err == nil {
		log.Info("successfully retrieved features")

		return features, nil
	}

	return nil, emperror.Wrap(err, "failed to retrieve features")
}

func (s *FeatureService) Update(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error {

	log := logur.WithFields(s.logger, map[string]interface{}{"clusterID": clusterID, "feature": featureName})
	log.Info("updating feature spec")

	featureManager, err := s.featureManagerRegistry.GetFeatureManager(ctx, featureName)
	if err != nil {

		return newUnsupportedFeatureError(featureName)
	}

	if _, err := s.featureRepository.GetFeature(ctx, clusterID, featureName); err != nil {
		log.Debug("feature could not be found")

		return newDatabaseAccessError(featureName)
	}

	ready, err := s.clusterService.IsClusterReady(ctx, clusterID)
	if err != nil {

		return emperror.Wrap(err, "could not access cluster")
	}

	if !ready {
		log.Debug("cluster not ready")

		return newClusterNotReadyError(featureName)
	}

	if err := featureManager.Update(ctx, clusterID, Feature{Name: featureName, Spec: spec}); err != nil {
		log.Debug("failed to update feature")

		return emperror.WrapWith(err, "failed to update feature", "clusterID", clusterID, "feature", featureName)
	}

	if _, err := s.featureRepository.UpdateFeatureSpec(ctx, clusterID, featureName, spec); err != nil {

		return emperror.WrapWith(err, "failed to update feature spec", "clusterID", clusterID, "feature", featureName)
	}

	log.Info("successfully updated feature spec")
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

type databaseAccessEerror struct {
	featureError
}

func newDatabaseAccessError(featureName string) error {
	return databaseAccessEerror{featureError{
		featureName: featureName,
		msg:         errorDatabaseAccess,
	}}
}
