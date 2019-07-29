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

package clusterfeatureadapter

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

// ClusterGetter restricts the external dependencies for the repository
type ClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

// ClusterService is an adapter providing access to the core cluster layer.
type clusterService struct {
	clusterGetter ClusterGetter
}

// NewClusterService returns a new ClusterService instance.
func NewClusterService(getter ClusterGetter) clusterfeature.ClusterService {
	return &clusterService{
		clusterGetter: getter,
	}
}

func (s *clusterService) IsClusterReady(ctx context.Context, clusterID uint) (bool, error) {
	c, err := s.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {

		return false, errors.WrapIf(err, "failed to retrieve cluster")
	}

	isReady, err := c.IsReady()
	if err != nil {

		return false, errors.WrapIfWithDetails(err, "failed to check cluster", "clusterId", clusterID)
	}

	return isReady, err
}
