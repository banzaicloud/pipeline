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

	"github.com/banzaicloud/pipeline/cluster"
)

//go:generate mockery -name ClusterGetter -inpkg
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

// CommonClusterGetter defines cluster getter methods that return a CommonCluster
type CommonClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

// MakeClusterGetter adapts a "CommonCluster" cluster getter to a clusterfeature cluster getter
func MakeClusterGetter(clusterGetter CommonClusterGetter) ClusterGetterAdapter {
	return ClusterGetterAdapter{
		clusterGetter: clusterGetter,
	}
}

// ClusterGetterAdapter adapts a "CommonCluster" cluster getter to a clusterfeature cluster getter
type ClusterGetterAdapter struct {
	clusterGetter CommonClusterGetter
}

// GetClusterByIDOnly returns the cluster with the specified ID
func (a ClusterGetterAdapter) GetClusterByIDOnly(ctx context.Context, clusterID uint) (Cluster, error) {
	return a.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
}
