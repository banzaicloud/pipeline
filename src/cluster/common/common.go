// Copyright © 2021 Banzai Cloud
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

package common

import (
	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

// CommonCluster interface for clusters.
type CommonCluster interface {
	// Entity properties
	GetID() uint
	GetUID() string
	GetOrganizationId() uint
	GetName() string
	GetCloud() string
	GetDistribution() string
	GetLocation() string

	// Secrets
	GetSecretId() string
	GetSshSecretId() string
	SaveSshSecretId(string) error
	SaveConfigSecretId(string) error
	GetConfigSecretId() string
	GetSecretWithValidation() (*secret.SecretItemResponse, error)

	// Persistence
	Persist() error
	DeleteFromDatabase() error

	// Cluster management
	CreateCluster() error
	ValidateCreationFields(r *cluster.CreateClusterRequest) error
	UpdateCluster(*cluster.UpdateClusterRequest, uint) error
	UpdateNodePools(*cluster.UpdateNodePoolsRequest, uint) error
	CheckEqualityToUpdate(*cluster.UpdateClusterRequest) error
	AddDefaultsToUpdate(*cluster.UpdateClusterRequest)
	DeleteCluster() error
	GetScaleOptions() *cluster.ScaleOptions
	SetScaleOptions(*cluster.ScaleOptions)

	// Kubernetes
	GetAPIEndpoint() (string, error)
	GetK8sConfig() ([]byte, error)
	GetK8sUserConfig() ([]byte, error)
	RequiresSshPublicKey() bool
	RbacEnabled() bool

	// Cluster info
	GetStatus() (*cluster.GetClusterStatusResponse, error)
	IsReady() (bool, error)
	NodePoolExists(nodePoolName string) bool

	SetStatus(status, statusMessage string) error
}
