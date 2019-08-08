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

package commonadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/secret"
)

// SecretStore implements the common.SecretStore interface and acts as a lightweight wrapper around
// the global secret store.
type SecretStore struct {
	store     OrganizationalSecretStore
	extractor OrgIDContextExtractor
}

// OrganizationalSecretStore is the global secret store that stores values under a compound key:
// the organization ID and a secret ID.
type OrganizationalSecretStore interface {
	// Get returns a secret in the internal format of the secret store.
	Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error)
}

// OrgIDContextExtractor extracts an organization ID from a context (if there is any).
type OrgIDContextExtractor interface {
	// GetOrganizationID extracts an organization ID from a context (if there is any).
	GetOrganizationID(ctx context.Context) (uint, bool)
}

// OrgIDContextExtractorFunc converts an ordinary function to an OrgIDContextExtractor
// (given it's method signature is compatible with the interface).
type OrgIDContextExtractorFunc func(ctx context.Context) (uint, bool)

// GetOrganizationID implements the OrgIDContextExtractor interface.
func (f OrgIDContextExtractorFunc) GetOrganizationID(ctx context.Context) (uint, bool) {
	return f(ctx)
}

// NewSecretStore returns a new SecretStore instance.
func NewSecretStore(store OrganizationalSecretStore, extractor OrgIDContextExtractor) *SecretStore {
	return &SecretStore{
		store:     store,
		extractor: extractor,
	}
}

// GetSecretValues implements the common.SecretStore interface.
func (s *SecretStore) GetSecretValues(ctx context.Context, secretID string) (map[string]string, error) {

	secretResponse, err := s.getSecret(ctx, secretID)
	if err != nil {
		return nil, err
	}

	return secretResponse.Values, nil
}

// GetSecret implements the common.SecretStore interface.
func (s *SecretStore) GetSecret(ctx context.Context, secretID string) (*secret.SecretItemResponse, error) {
	return s.getSecret(ctx, secretID)
}

func (s *SecretStore) getSecret(ctx context.Context, secretID string) (*secret.SecretItemResponse, error) {
	organizationID, ok := s.extractor.GetOrganizationID(ctx)
	if !ok {
		return nil, errors.NewWithDetails(
			"organization ID cannot be found in the context",
			"organizationId", organizationID,
			"secretId", secretID,
		)
	}

	secretResponse, err := s.store.Get(organizationID, secretID)
	if err == secret.ErrSecretNotExists {
		return nil, errors.WithDetails(
			errors.WithStack(common.SecretNotFoundError{SecretID: secretID}),
			"organizationId", organizationID,
		)
	}
	if err != nil {
		return nil, errors.WithDetails(
			errors.WithStackIf(err),
			"organizationId", organizationID,
			"secretId", secretID,
		)
	}

	return secretResponse, nil
}
