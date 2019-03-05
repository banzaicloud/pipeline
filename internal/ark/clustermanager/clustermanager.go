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
package clustermanager

import (
	"context"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/ark/api"
)

type ClusterManager struct {
	clusterManager *cluster.Manager
}

// New creates a new ClusterManager
func New(manager *cluster.Manager) *ClusterManager {
	return &ClusterManager{
		clusterManager: manager,
	}
}

// GetClusters returns clusters registered to an organization
func (cm *ClusterManager) GetClusters(ctx context.Context, organizationID uint) ([]api.Cluster, error) {
	clusters, err := cm.clusterManager.GetClusters(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	apiClusters := make([]api.Cluster, len(clusters))
	for i, cluster := range clusters {
		apiClusters[i] = cluster
	}

	return apiClusters, nil
}
