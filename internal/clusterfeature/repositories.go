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
	"strconv"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/goph/emperror"
	"golang.org/x/net/context"
)

// ClusterRepository collects persistence related operations
type ClusterRepository interface {
	// IsClusterReady checks whether the cluster is ready for features (eg.: exists and it's running)
	IsClusterReady(ctx context.Context, clusterId string) (bool, error)

	// GetCluster retrieves the cluster representation based on the cluster identifier
	GetCluster(ctx context.Context, clusterId string) (cluster.CommonCluster, error)
}

// clusterGetter restricts the external dependencies for the repository
type clusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

//
type featureClusterRepository struct {
	clusterGetter clusterGetter
}

func (fcs *featureClusterRepository) GetCluster(ctx context.Context, clusterId string) (cluster.CommonCluster, error) {
	// todo use uint everywhere
	cid, err := strconv.ParseUint(clusterId, 0, 64)
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to parse clusterid", "clusterid", clusterId)
	}

	cluster, err := fcs.clusterGetter.GetClusterByIDOnly(ctx, uint(cid))
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to retrieve cluster", "clusterid", clusterId)
	}

	return cluster, nil
}

func (fcs *featureClusterRepository) IsClusterReady(ctx context.Context, clusterId string) (bool, error) {
	cluster, err := fcs.GetCluster(ctx, clusterId)
	if err != nil {
		return false, err
	}

	isReady, err := cluster.IsReady()
	if err != nil {
		return false, emperror.WrapWith(err, "failed to check cluster", "clusterid", clusterId)
	}

	return isReady, err
}

func (fcs *featureClusterRepository) GetKubeConfig(ctx context.Context, clusterId string) ([]byte, error) {

	cluster, err := fcs.GetCluster(ctx, clusterId)
	if err != nil {
		return nil, err
	}

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to retrieve kubeConfig", "clusterid", clusterId)
	}

	return kubeConfig, nil

}

func NewClusterRepository(getter clusterGetter) ClusterRepository {
	return &featureClusterRepository{
		clusterGetter: getter,
	}
}

// FeatureRepository collects persistence related operations
type FeatureRepository interface {
	SaveFeature(ctx context.Context, clusterId string, feature Feature) (string, error)
	GetFeature(ctx context.Context, clusterId string, feature Feature) (*Feature, error)
	UpdateFeatureStatus(ctx context.Context, clusterId string, feature Feature, status string) (*Feature, error)
}
