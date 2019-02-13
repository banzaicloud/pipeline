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

package pkeworkflowadapter

import (
	"context"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
)

// ClusterManagerAdapter provides an adapter for pkeworkflow.Clusters.
type ClusterManagerAdapter struct {
	clusterManager *cluster.Manager
}

// NewClusterManagerAdapter creates a new ClusterManagerAdapter.
func NewClusterManagerAdapter(clusterManager *cluster.Manager) *ClusterManagerAdapter {
	return &ClusterManagerAdapter{
		clusterManager: clusterManager,
	}
}

// GetCluster returns a Cluster.
func (a *ClusterManagerAdapter) GetCluster(ctx context.Context, id uint) (pkeworkflow.Cluster, error) {
	return a.clusterManager.GetClusterByIDOnly(ctx, id)
}
