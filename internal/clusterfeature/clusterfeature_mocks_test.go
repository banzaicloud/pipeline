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

package clusterfeature_test

import (
	"context"

	"github.com/goph/logur"
	"github.com/pkg/errors"

	. "github.com/banzaicloud/pipeline/internal/clusterfeature"
)

const (
	clusterNotReady = 100
)

type dummyFeatureRepository struct {
	logger logur.Logger
}

func (dfr *dummyFeatureRepository) UpdateFeatureStatus(ctx context.Context, clusterId uint, feature Feature, status FeatureStatus) (*Feature, error) {
	dfr.logger.Info("feature repo called", map[string]interface{}{"operation": "UpdateFeatureStatus", "clusterId": clusterId})
	return nil, nil
}

func (dfr *dummyFeatureRepository) GetFeature(ctx context.Context, clusterId uint, feature Feature) (*Feature, error) {
	switch feature.Name {
	case "existingfeature":
		return &Feature{Name: "existingfeature"}, nil
	}
	return nil, errors.New("feature not found")
}

func (dfr *dummyFeatureRepository) SaveFeature(ctx context.Context, clusterId uint, feature Feature) (uint, error) {
	switch feature.Name {
	case "failtopersist":
		return 0, errors.New("persistence error")
	}

	return 111, nil
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
