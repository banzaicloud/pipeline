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

package clustersecret

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

// SecretCreateRequest represents a new secret.
type SecretCreateRequest struct {
	Name   string
	Type   string
	Values map[string]string
	Tags   []string
}

// Store manages cluster level secrets.
type Store struct {
	clusters Clusters
	secrets  SecretStore
}

// NewStore returns a new cluster secret store.
func NewStore(clusters Clusters, secrets SecretStore) *Store {
	return &Store{
		clusters: clusters,
		secrets:  secrets,
	}
}

// Clusters provides access to the list of clusters.
type Clusters interface {
	// GetCluster returns a new cluster based on it's ID.
	GetCluster(ctx context.Context, id uint) (Cluster, error)
}

// Cluster represents a Kubernetes cluster.
type Cluster interface {
	// GetOrganizationID returns the organization ID of the cluster.
	GetOrganizationID() uint

	// GetUID returns the unique ID of the cluster.
	GetUID() string
}

// SecretStore is a generic secret store.
type SecretStore interface {
	// EnsureSecretExists creates a secret for an organization if it cannot be found and returns it's ID.
	EnsureSecretExists(organizationID uint, secret SecretCreateRequest) (string, error)
}

// EnsureSecretExists creates a secret for a cluster if it cannot be found and returns it's ID.
func (s *Store) EnsureSecretExists(ctx context.Context, clusterID uint, secret SecretCreateRequest) (string, error) {
	cluster, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return "", errors.Wrap(err, "failed to create secret")
	}

	// Prepend the name with a cluster prefix
	secret.Name = fmt.Sprintf("cluster-%d-%s", clusterID, secret.Name)

	// Append cluster tags
	// TODO: check for uniqueness?
	secret.Tags = append(
		secret.Tags,
		fmt.Sprintf("clusterUID:%s", cluster.GetUID()),
		fmt.Sprintf("clusterID:%d", clusterID),
	)

	return s.secrets.EnsureSecretExists(cluster.GetOrganizationID(), secret)
}
