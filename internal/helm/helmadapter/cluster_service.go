// Copyright © 2020 Banzai Cloud
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

package helmadapter

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/src/cluster"
)

// clusterGetter restricts the external dependencies for the repository
type clusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

type clusterService struct {
	clusterGetter clusterGetter
}

// NewClusterService returns a new ClusterService instance.
func NewClusterService(getter clusterGetter) *clusterService {
	return &clusterService{
		clusterGetter: getter,
	}
}

func (c clusterService) GetKubeConfig(ctx context.Context, clusterID uint) ([]byte, error) {
	commonCluster, err := c.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	return commonCluster.GetK8sConfig()
}
