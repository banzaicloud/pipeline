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

package clustersecretadapter

import (
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret"
	"github.com/banzaicloud/pipeline/secret"
)

// InternalSecretStore is an interface for the internal secret store.
type InternalSecretStore interface {
	// GetOrCreate create new secret or get if it's exist.
	GetOrCreate(organizationID uint, value *secret.CreateSecretRequest) (string, error)

	// GetByName gets a secret by name if it's exist.
	GetByName(organizationID uint, name string) (*secret.SecretItemResponse, error)
}

// SecretStore is a wrapper for the internal secret store.
type SecretStore struct {
	secrets InternalSecretStore
}

// NewSecretStore returns a wrapper for the internal secret store.
func NewSecretStore(secrets InternalSecretStore) *SecretStore {
	return &SecretStore{
		secrets: secrets,
	}
}

// EnsureSecretExists creates a secret for an organization if it cannot be found.
func (s *SecretStore) EnsureSecretExists(organizationID uint, sec clustersecret.SecretCreateRequest) (string, error) {
	createSecret := &secret.CreateSecretRequest{
		Name:   sec.Name,
		Type:   sec.Type,
		Values: sec.Values,
		Tags:   sec.Tags,
	}

	id, err := s.secrets.GetOrCreate(organizationID, createSecret)

	return string(id), err
}

// GetSecret gets a secret by name if it exists
func (s *SecretStore) GetSecret(organizationID uint, name string) (clustersecret.SecretResponse, error) {
	sec, err := s.secrets.GetByName(organizationID, name)

	if err != nil {
		return clustersecret.SecretResponse{}, err
	}

	if sec == nil {
		return clustersecret.SecretResponse{}, clustersecret.ErrSecretNotFound
	}

	return clustersecret.SecretResponse{
		Name:   sec.Name,
		Type:   sec.Type,
		Values: sec.Values,
		Tags:   sec.Tags,
	}, nil
}
