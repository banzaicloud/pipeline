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

package pke

import (
	"emperror.dev/errors"

	intPKE "github.com/banzaicloud/pipeline/internal/pke"
)

type CreateParams struct {
	Name              string
	OrganizationID    uint
	CreatedBy         uint
	SecretID          string
	StorageSecretID   string
	SSHSecretID       string
	RBAC              bool
	OIDC              bool
	KubernetesVersion string
	NodePools         []NodePool
	HTTPProxy         intPKE.HTTPProxy

	ResourcePoolName    string
	FolderName          string
	DatastoreName       string
	Kubernetes          intPKE.Kubernetes
	LoadBalancerIPRange string
}

// ClusterStore defines behaviors of PKEOnVsphereCluster persistent storage
type ClusterStore interface {
	Create(params CreateParams) (PKEOnVsphereCluster, error)
	CreateNodePool(clusterID uint, nodePool NodePool) error
	Delete(clusterID uint) error
	DeleteNodePool(clusterID uint, nodePoolName string) error
	UpdateNodePoolSize(clusterID uint, nodePoolName string, size int) error
	GetByID(clusterID uint) (PKEOnVsphereCluster, error)
	SetStatus(clusterID uint, status, message string) error
	SetActiveWorkflowID(clusterID uint, workflowID string) error
	SetConfigSecretID(clusterID uint, secretID string) error
	SetSSHSecretID(clusterID uint, sshSecretID string) error
}

// IsNotFound returns true if the error is about a resource not being found
func IsNotFound(err error) bool {
	var notFoundErr interface {
		NotFound() bool
	}

	return errors.As(err, &notFoundErr) && notFoundErr.NotFound()
}
