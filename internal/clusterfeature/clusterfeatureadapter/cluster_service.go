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

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

// ClusterService is an adapter providing access to the core cluster layer.
type ClusterService struct {
	clusterStatuser ClusterStatuser
}

// ClusterStatuser supports getting a cluster's status
type ClusterStatuser interface {
	GetClusterStatus(ctx context.Context, clusterID uint) (string, error)
}

// NewClusterService returns a new ClusterService instance.
func NewClusterService(clusterStatuser ClusterStatuser) ClusterService {
	return ClusterService{
		clusterStatuser: clusterStatuser,
	}
}

// CheckClusterReady returns true if the cluster is ready to be accessed
func (s ClusterService) CheckClusterReady(ctx context.Context, clusterID uint) error {
	status, err := s.clusterStatuser.GetClusterStatus(ctx, clusterID)
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to get cluster status", "clusterId", clusterID)
	}

	if status != pkgCluster.Running {
		return clusterfeature.ClusterIsNotReadyError{
			ClusterID: clusterID,
		}
	}

	return nil
}
