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

//go:generate sh -c "test -x ${MOCKERY} && ${MOCKERY} -name ClusterGetter -inpkg"
// ClusterGetter restricts the external dependencies for the repository
type ClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (Cluster, error)
}

// Cluster defines operations that can be performed on a k8s cluster
type Cluster interface {
	GetK8sConfig() ([]byte, error)
	GetName() string
	GetOrganizationId() uint
	GetUID() string
	GetID() uint
	IsReady() (bool, error)
	NodePoolExists(nodePoolName string) bool
	RbacEnabled() bool
}

// MakeClusterGetter creates a ClusterGetter using a common cluster getter
func MakeClusterGetter(clusterGetter CommonClusterGetter) ClusterGetter {
	return clusterGetterAdapter{
		ccGetter: clusterGetter,
	}
}

// CommonClusterGetter defines cluster getter methods that return a CommonCluster
type CommonClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

type clusterGetterAdapter struct {
	ccGetter CommonClusterGetter
}

func (a clusterGetterAdapter) GetClusterByIDOnly(ctx context.Context, clusterID uint) (Cluster, error) {
	return a.ccGetter.GetClusterByIDOnly(ctx, clusterID)
}

// ClusterService is an adapter providing access to the core cluster layer.
type ClusterService struct {
	clusterGetter ClusterGetter
}

// NewClusterService returns a new ClusterService instance.
func NewClusterService(getter ClusterGetter) ClusterService {
	return ClusterService{
		clusterGetter: getter,
	}
}

// CheckClusterReady returns true is the cluster is ready to be accessed
func (s ClusterService) CheckClusterReady(ctx context.Context, clusterID uint) error {
	c, err := s.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {

		return errors.WrapIfWithDetails(err, "failed to retrieve cluster", "clusterId", clusterID)
	}

	isReady, err := c.IsReady()
	if err != nil {

		return errors.WrapIfWithDetails(err, "failed to check cluster", "clusterId", clusterID)
	}

	if !isReady {
		return clusterfeature.ClusterIsNotReadyError{
			ClusterID: clusterID,
		}
	}

	return nil
}
