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

package securityscanadapter

import (
	"context"

	"github.com/banzaicloud/pipeline/src/cluster"
)

// CommonClusterGetter defines cluster getter methods that return a CommonCluster
type CommonClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

type clusterService struct {
	clusterGetter CommonClusterGetter
}

// NewClusterService returns a new ClusterService instance.
func NewClusterService(getter CommonClusterGetter) ClusterService {
	return clusterService{
		clusterGetter: getter,
	}
}

func (s clusterService) GetClusterUID(ctx context.Context, clusterID uint) (string, error) {
	c, err := s.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return "", err
	}

	return c.GetUID(), nil
}
