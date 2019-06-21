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
	"github.com/goph/logur"
	"github.com/pkg/errors"
)

type dummyFeatureRepository struct {
	logger logur.Logger
}

func (dfr *dummyFeatureRepository) UpdateFeatureStatus(ctx context.Context, clusterId string, feature Feature, status string) (*Feature, error) {
	dfr.logger.Info("feature repo called", map[string]interface{}{"operation": "UpdateFeatureStatus", "clusterId": clusterId})
	return nil, nil
}

func (dfr *dummyFeatureRepository) GetFeature(ctx context.Context, clusterId string, feature Feature) (*Feature, error) {
	switch feature.Name {
	case "existingfeature":
		return &Feature{Name: "existingfeature"}, nil
	}
	return nil, errors.New("feature not found")
}

func (dfr *dummyFeatureRepository) SaveFeature(ctx context.Context, clusterId string, feature Feature) (string, error) {
	switch feature.Name {
	case "failtopersist":
		return "", errors.New("persistence error")
	}

	return "featureId", nil
}

type dummyClusterRepository struct {
}

func (dcr *dummyClusterRepository) IsClusterReady(ctx context.Context, clusterId string) (bool, error) {
	switch clusterId {
	case "notready":
		return false, nil
	}

	return true, nil

}

func (dcr *dummyClusterRepository) GetCluster(ctx context.Context, clusterId string) (cluster.CommonCluster, error) {
	panic("implement me")
}

type dummyFeatureManager struct {
}

func (dfm *dummyFeatureManager) Activate(ctx context.Context, clusterId string, feature Feature) (string, error) {
	switch feature.Name {
	case "success":
		return "ok", nil
	}

	return "", errors.New("test - failed to activate feature")
}

func (dfm *dummyFeatureManager) Update(ctx context.Context, clusterId string, feature Feature) (string, error) {
	panic("implement me")
}
