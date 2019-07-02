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

	"github.com/goph/logur"
	"github.com/pkg/errors"
)

const (
	clusterNotReady = 100
	clusterReady    = 200

	featureExists             = "existing-feature"
	featureCouldNotPersist    = "feature-fail-to-persist"
	featureSelectionErrorName = "feature-couldnotselect"
)

type dummyFeatureRepository struct {
	logger logur.Logger
}

func (dfr *dummyFeatureRepository) UpdateFeatureSpec(ctx context.Context, clusterId uint, featureName string, spec map[string]interface{}) (*Feature, error) {
	panic("implement me")
}

func (dfr *dummyFeatureRepository) DeleteFeature(ctx context.Context, clusterId uint, featureName string) error {
	panic("implement me")
}

func (dfr *dummyFeatureRepository) ListFeatures(ctx context.Context, clusterId uint) ([]*Feature, error) {
	panic("implement me")
}

func (dfr *dummyFeatureRepository) UpdateFeatureStatus(ctx context.Context, clusterId uint, featureName string, status string) (*Feature, error) {
	dfr.logger.Info("feature repo called", map[string]interface{}{"operation": "UpdateFeatureStatus", "clusterId": clusterId})
	return nil, nil
}

func (dfr *dummyFeatureRepository) GetFeature(ctx context.Context, clusterId uint, featureName string) (*Feature, error) {
	switch featureName {
	case featureExists:
		return &Feature{Name: featureExists}, nil
	}
	return nil, errors.New("feature not found")
}

func (dfr *dummyFeatureRepository) SaveFeature(ctx context.Context, clusterId uint, feature Feature) (uint, error) {
	switch feature.Name {
	case featureCouldNotPersist:
		return 0, errors.New("persistence error")
	}

	return 0, nil
}

type dummyClusterRepository struct {
}

func (dcr *dummyClusterRepository) GetCluster(ctx context.Context, clusterID uint) (Cluster, error) {
	panic("implement me")
}

func (dcr *dummyClusterRepository) IsClusterReady(ctx context.Context, clusterId uint) (bool, error) {
	switch clusterId {
	case clusterNotReady:
		return false, nil
	}

	return true, nil

}

type dummyFeatureManager struct {
}

func (dfm *dummyFeatureManager) Deactivate(ctx context.Context, clusterId uint, feature Feature) (error) {
	panic("implement me")
}

func (dfm *dummyFeatureManager) Activate(ctx context.Context, clusterId uint, feature Feature) (string, error) {
	switch feature.Name {
	case "success":
		return "ok", nil
	}

	return "", errors.New("test - failed to activate feature")
}

func (dfm *dummyFeatureManager) Update(ctx context.Context, clusterId uint, feature Feature) (string, error) {
	panic("implement me")
}

type dummyFeatureSelector struct {
}

func (fs *dummyFeatureSelector) SelectFeature(ctx context.Context, feature Feature) (*Feature, error) {
	switch feature.Name {
	case featureSelectionErrorName:
		return nil, newFeatureSelectionError(feature.Name)

	}
	return &feature, nil
}
