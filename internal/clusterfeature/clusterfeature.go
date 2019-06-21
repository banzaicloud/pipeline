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

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
)

// ClusterFeatureService collects operations supporting cluster features
type ClusterFeatureService interface {
	// Activate deploys / enables a cluster feature to the cluster represented by it's identifier
	Activate(ctx context.Context, clusterId string, feature Feature) error

	// Update updates an existing feature on the cluster represented by it's identifier
	Update(ctx context.Context, clusterId string, feature Feature) error

	// Deactivate removes a feature from the cluster represented by it's identifier
	Deactivate(ctx context.Context, clusterId string, feature Feature) error
}

// FeatureRepository collects persistence related operations
type FeatureRepository interface {
	SaveFeature(ctx context.Context, clusterId string, feature Feature) (string, error)
	GetFeature(ctx context.Context, clusterId string, feature Feature) (*Feature, error)
	UpdateFeatureStatus(ctx context.Context, clusterId string, feature Feature, status string) (*Feature, error)
}

// ClusterRepository collects persistence related operations
type ClusterRepository interface {
	// IsClusterReady checks whether the cluster is ready for features (eg.: exists and it's running)
	IsClusterReady(ctx context.Context, clusterId string) (bool, error)

	// GetCluster retrieves the cluster representation based on the cluster identifier
	GetCluster(ctx context.Context, clusterId string) (cluster.CommonCluster, error)
}

// Feature represents a cluster feature instance
type Feature struct {
	Name   string
	Status string
	Spec   map[string]interface{}
	Output map[string]interface{}
}

// clusterFeature component struct, implements the ClusterFeatureService functionality
type clusterFeatureService struct {
	logger            logur.Logger
	clusterRepository ClusterRepository
	featureRepository FeatureRepository
	featureManager    FeatureManager
}

func (cfs *clusterFeatureService) Activate(ctx context.Context, clusterId string, feature Feature) error {
	cfs.logger.Info("activate feature", map[string]interface{}{"feature": feature.Name})

	ready, err := cfs.clusterRepository.IsClusterReady(ctx, clusterId)
	if err != nil {
		cfs.logger.Debug("failed to check the cluster", map[string]interface{}{"clusterId": clusterId})
		return emperror.Wrap(err, "failed to check the cluster")
	}

	if !ready {
		cfs.logger.Debug("cluster not ready", map[string]interface{}{"clusterId": clusterId})
		return errors.New("cluster is not ready")
	}

	if _, err := cfs.featureRepository.GetFeature(ctx, clusterId, feature); err == nil {
		cfs.logger.Debug("feature exists", map[string]interface{}{"clusterId": clusterId, "feature": feature.Name})
		return errors.New("feature already exists")
	}

	if _, err := cfs.featureRepository.SaveFeature(ctx, clusterId, feature); err != nil {
		cfs.logger.Debug("failed to save feature", map[string]interface{}{"clusterId": clusterId, "feature": feature.Name})
		return emperror.WrapWith(err, "failed to persist feature", "clusterId", clusterId, "feature", feature.Name)
	}

	// delegate the task of "deploying" the feature to the manager
	if _, err := cfs.featureManager.Activate(ctx, clusterId, feature); err != nil {
		cfs.logger.Debug("failed to activate feature", map[string]interface{}{"clusterId": clusterId, "feature": feature.Name})
		return emperror.WrapWith(err, "failed to persist feature", "clusterId", clusterId, "feature", feature.Name)
	}

	// todo update the status in case of errors!
	if _, err := cfs.featureRepository.UpdateFeatureStatus(ctx, clusterId, feature, "ACTIVE"); err != nil {
		cfs.logger.Debug("failed to update feature status ", map[string]interface{}{"clusterId": clusterId, "feature": feature.Name})
		return emperror.WrapWith(err, "failed to update feature status", "clusterId", clusterId, "feature", feature.Name)
	}

	cfs.logger.Info("feature successfully activated ", map[string]interface{}{"clusterId": clusterId, "feature": feature.Name})
	// activation succeeded
	return nil
}

func (cfs *clusterFeatureService) Update(ctx context.Context, clusterId string, feature Feature) error {
	panic("implement me")
}

func (cfs *clusterFeatureService) Deactivate(ctx context.Context, clusterId string, feature Feature) error {
	panic("implement me")
}

func NewClusterFeatureService(logger logur.Logger, clusterRepository ClusterRepository, featureRepository FeatureRepository, featureManager FeatureManager) ClusterFeatureService {
	return &clusterFeatureService{
		logger:            logger,
		clusterRepository: clusterRepository,
		featureRepository: featureRepository,
		featureManager:    featureManager,
	}
}
