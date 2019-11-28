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

package helmadapter

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/src/cluster"
)

// clusterGetter restricts the external dependencies for the repository
type clusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

// ClusterService is an adapter providing access to the core cluster layer.
type ClusterService struct {
	clusterGetter clusterGetter
}

// NewClusterService returns a new ClusterService instance.
func NewClusterService(getter clusterGetter) *ClusterService {
	return &ClusterService{
		clusterGetter: getter,
	}
}

// GetCluster retrieves the cluster representation based on the cluster identifier.
func (s *ClusterService) GetCluster(ctx context.Context, clusterID uint) (*helm.Cluster, error) {
	c, err := s.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {

		return nil, err
	}

	org, err := auth.GetOrganizationById(c.GetOrganizationId())
	if err != nil {

		return nil, errors.WrapIfWithDetails(err, "failed to get organization", "organizationId", c.GetOrganizationId())
	}

	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to get kube config", "clusterId", c.GetID())
	}

	return &helm.Cluster{OrganizationName: org.Name, KubeConfig: kubeConfig}, nil
}
