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
	"github.com/banzaicloud/pipeline/pkg/auth"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

// InternalSecretStore is an interface for the internal secret store.
type InternalSecretStore interface {
	// GetOrCreate create new secret or get if it's exist.
	GetOrCreate(organizationID auth.OrganizationID, value *secret.CreateSecretRequest) (pkgSecret.SecretID, error)
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
func (s *SecretStore) EnsureSecretExists(organizationID uint, sec clustersecret.NewSecret) (string, error) {
	createSecret := &secret.CreateSecretRequest{
		Name:   sec.Name,
		Type:   sec.Type,
		Values: sec.Values,
		Tags:   sec.Tags,
	}

	id, err := s.secrets.GetOrCreate(auth.OrganizationID(organizationID), createSecret)

	return string(id), err
}
