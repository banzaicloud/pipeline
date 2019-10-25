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

package clusteradapter

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/cluster"
)

// CommonClusterGetter defines cluster getter methods that return a CommonCluster
type CommonClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

// ClusterService is an adapter providing access to the core cluster layer.
type ClusterService interface {
	GetClusterUUID(ctx context.Context, orgID uint, clusterID uint) (string, error)
}

type AnchoreClusterService struct {
	clusterGetter CommonClusterGetter
}

// NewClusterService returns a new ClusterService instance.
func NewClusterService(getter CommonClusterGetter) ClusterService {
	return AnchoreClusterService{
		clusterGetter: getter,
	}
}

func (cs AnchoreClusterService) GetClusterUUID(ctx context.Context, orgID uint, clusterID uint) (string, error) {
	c, err := cs.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return "", errors.WrapIf(err, "failed to retrieve cluster UUID")
	}

	return c.GetUID(), nil
}
