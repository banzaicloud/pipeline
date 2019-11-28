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

package pkeworkflowadapter

import (
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/src/secret"
)

// InternalSecretStore is an interface for the internal secret store.
type InternalSecretStore interface {
	// Get retrieves a secret from the store.
	Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
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

func (s *SecretStore) GetSecret(organizationID uint, secretID string) (pkeworkflow.Secret, error) {
	sec, err := s.secrets.Get(organizationID, secretID)
	if err != nil {
		return nil, err
	}

	return &secretWrapper{sec}, nil
}

type secretWrapper struct {
	secretItem *secret.SecretItemResponse
}

func (s *secretWrapper) GetValues() map[string]string {
	return s.secretItem.Values
}

func (s *secretWrapper) ValidateSecretType(t string) error {
	return s.secretItem.ValidateSecretType(t)
}
