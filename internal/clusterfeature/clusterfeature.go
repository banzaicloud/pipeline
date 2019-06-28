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

	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
)

// Feature represents the internal state of a cluster feature.
type Feature struct {
	Name   string                 `json:"name"`
	Spec   map[string]interface{} `json:"spec"`
	Output map[string]interface{} `json:"output"`
	Status FeatureStatus          `json:"status"`
}

type FeatureStatus string

const (
	FeatureStatusActive  FeatureStatus = "ACTIVE"
	FeatureStatusPending FeatureStatus = "PENDING"
)

// FeatureService manages features on Kubernetes clusters.
type FeatureService struct {
	logger            logur.Logger
	clusterService    ClusterService
	featureRepository FeatureRepository
	featureManager    FeatureManager
}

// ClusterService provides a thin access layer to clusters.
type ClusterService interface {
	// GetCluster retrieves the cluster representation based on the cluster identifier
	GetCluster(ctx context.Context, clusterID uint) (Cluster, error)

	// IsClusterReady checks whether the cluster is ready for features (eg.: exists and it's running).
	IsClusterReady(ctx context.Context, clusterID uint) (bool, error)
}

// Cluster represents a Kubernetes cluster.
type Cluster interface {
	GetID() uint
	GetOrganizationName() string
	GetKubeConfig() ([]byte, error)
}

// FeatureRepository collects persistence related operations.
type FeatureRepository interface {
	SaveFeature(ctx context.Context, clusterId uint, feature Feature) (uint, error)
	GetFeature(ctx context.Context, clusterId uint, feature Feature) (*Feature, error)
	UpdateFeatureStatus(ctx context.Context, clusterId uint, feature Feature, status FeatureStatus) (*Feature, error)
}

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
	}
}

func (s *FeatureService) Activate(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error {
	s.logger.Info("activate feature", map[string]interface{}{"feature": featureName})

	ready, err := s.clusterService.IsClusterReady(ctx, clusterID)
	if err != nil {
		return err
	}

	if !ready {
		s.logger.Debug("cluster not ready", map[string]interface{}{"clusterId": clusterID})
		return errors.New("cluster is not ready")
	}

	feature := Feature{
		Name: featureName,
		Spec: spec,
	}

	if _, err := s.featureRepository.GetFeature(ctx, clusterID, feature); err == nil {
		s.logger.Debug("feature exists", map[string]interface{}{"clusterId": clusterID, "feature": featureName})
		return errors.New("feature already exists")
	}

	if _, err := s.featureRepository.SaveFeature(ctx, clusterID, feature); err != nil {
		s.logger.Debug("failed to save feature", map[string]interface{}{"clusterId": clusterID, "feature": featureName})
		return emperror.WrapWith(err, "failed to persist feature", "clusterId", clusterID, "feature", featureName)
	}

	// delegate the task of "deploying" the feature to the manager
	if _, err := s.featureManager.Activate(ctx, clusterID, feature); err != nil {
		s.logger.Debug("failed to activate feature", map[string]interface{}{"clusterId": clusterID, "feature": featureName})
		return emperror.WrapWith(err, "failed to activate feature", "clusterId", clusterID, "feature", featureName)
	}

	if _, err := s.featureRepository.UpdateFeatureStatus(ctx, clusterID, feature, FeatureStatusActive); err != nil {
		s.logger.Debug("failed to update feature status ", map[string]interface{}{"clusterId": clusterID, "feature": featureName})
		return emperror.WrapWith(err, "failed to update feature status", "clusterId", clusterID, "feature", featureName)
	}

	s.logger.Info("feature successfully activated ", map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	// activation succeeded
	return nil
}

func (s *FeatureService) List(ctx context.Context, clusterID uint) ([]Feature, error) {
	panic("implement me")
}

func (s *FeatureService) Details(ctx context.Context, clusterID uint, featureName string) (*Feature, error) {
	panic("implement me")
}

func (s *FeatureService) Deactivate(ctx context.Context, clusterID uint, featureName string) error {
	panic("implement me")
}

func (s *FeatureService) Update(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error {
	panic("implement me")
}
