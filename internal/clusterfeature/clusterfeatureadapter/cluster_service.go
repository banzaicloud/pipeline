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

	"emperror.dev/emperror"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

// clusterGetter restricts the external dependencies for the repository
type clusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

// ClusterService is an adapter providing access to the core cluster layer.
type clusterService struct {
	clusterGetter clusterGetter
}

// NewClusterService returns a new ClusterService instance.
func NewClusterService(getter clusterGetter) clusterfeature.ClusterService {
	return &clusterService{
		clusterGetter: getter,
	}
}

func (s *clusterService) GetCluster(ctx context.Context, clusterID uint) (clusterfeature.Cluster, error) {
	c, err := s.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	org, err := auth.GetOrganizationById(c.GetOrganizationId())
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to get organization", "organizationId", c.GetOrganizationId())
	}

	return clusterAdapter{c, org.Name}, nil
}

func (s *clusterService) IsClusterReady(ctx context.Context, clusterID uint) (bool, error) {
	c, err := s.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return false, err
	}

	isReady, err := c.IsReady()
	if err != nil {
		return false, emperror.WrapWith(err, "failed to check cluster", "clusterId", clusterID)
	}

	return isReady, err
}

type clusterAdapter struct {
	cluster cluster.CommonCluster
	orgName string
}

func (c clusterAdapter) GetOrganizationID() uint {
	return c.cluster.GetOrganizationId()
}

func (c clusterAdapter) GetID() uint {
	return c.cluster.GetID()
}

func (c clusterAdapter) GetOrganizationName() string {
	return c.orgName
}

func (c clusterAdapter) GetKubeConfig() ([]byte, error) {
	return c.cluster.GetK8sConfig()
}

func (c clusterAdapter) IsReady() (bool, error) {
	isReady, err := c.cluster.IsReady()
	if err != nil {
		return false, emperror.WrapWith(err, "failed to check cluster", "clusterId", c.GetID())
	}

	return isReady, err
}
