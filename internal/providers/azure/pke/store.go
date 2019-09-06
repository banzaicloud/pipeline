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
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/pkg/errors"
)

type CreateParams struct {
	Name               string
	OrganizationID     uint
	CreatedBy          uint
	Location           string
	SecretID           string
	SSHSecretID        string
	RBAC               bool
	OIDC               bool
	KubernetesVersion  string
	ScaleOptions       pkgCluster.ScaleOptions
	ResourceGroupName  string
	VirtualNetworkName string
	NodePools          []NodePool
	Features           []intCluster.Feature
}

// AzurePKEClusterStore defines behaviors of PKEOnAzureCluster persistent storage
type AzurePKEClusterStore interface {
	Create(params CreateParams) (PKEOnAzureCluster, error)
	CreateNodePool(clusterID uint, nodePool NodePool) error
	Delete(clusterID uint) error
	DeleteNodePool(clusterID uint, nodePoolName string) error
	GetByID(clusterID uint) (PKEOnAzureCluster, error)
	SetStatus(clusterID uint, status, message string) error
	SetActiveWorkflowID(clusterID uint, workflowID string) error
	SetConfigSecretID(clusterID uint, secretID string) error
	SetSSHSecretID(clusterID uint, sshSecretID string) error
	SetFeature(clusterID uint, feature string, state bool) error
	SetNodePoolSizes(clusterID uint, nodePoolName string, min, max, desiredCount uint, autoscaling bool) error
}

// IsNotFound returns true if the error is about a resource not being found
func IsNotFound(err error) bool {
	// Check the root cause error.
	err = errors.Cause(err)

	if e, ok := err.(interface {
		NotFound() bool
	}); ok {
		return e.NotFound()
	}

	return false
}
