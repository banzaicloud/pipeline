// Copyright © 2018 Banzai Cloud
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

package api

import (
	"context"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

type AzureCluster interface {
	Cluster
	GetResourceGroupName() string
}

// Cluster interface for cluster implementations
type Cluster interface {
	GetID() uint
	GetName() string
	GetOrganizationId() uint
	GetCloud() string
	GetDistribution() string
	GetK8sConfig() ([]byte, error)
	GetSecretWithValidation() (*secret.SecretItemResponse, error)
	GetLocation() string
	RbacEnabled() bool
	GetStatus() (*pkgCluster.GetClusterStatusResponse, error)
}

// ClusterManager interface for getting clusters
type ClusterManager interface {
	GetClusters(context.Context, uint) ([]Cluster, error)
}

// Service manages integrated services on Kubernetes clusters.
type Service interface {
	// Activate activates a integrated service.
	Activate(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error

	// Deactivate deactivates a integrated service.
	Deactivate(ctx context.Context, clusterID uint, serviceName string) error
}
