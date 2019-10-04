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
	"github.com/banzaicloud/pipeline/pkg/brn"
	"github.com/banzaicloud/pipeline/secret"
)

// SecretStore implements the common.SecretStore interface and acts as a lightweight wrapper around
// the global secret store.
type SecretStore struct {
	store     ReadWriteOrganizationalSecretStore
	extractor OrgIDContextExtractor
}

// ReadWriteOrganizationalSecretStore is the global secret store that stores values under a compound key:
// the organization ID and a secret ID.
type ReadWriteOrganizationalSecretStore interface {
	ReadOnlyOrganizationalSecretStore

	Store(organizationID uint, request *secret.CreateSecretRequest) (string, error)

	// GetByName returns a secret in the internal format of the secret store based on secret name.
	GetByName(organizationID uint, name string) (*secret.SecretItemResponse, error)

	Delete(organizationID uint, secretID string) error
}

type ReadOnlyOrganizationalSecretStore interface {
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
func NewSecretStore(store ReadWriteOrganizationalSecretStore, extractor OrgIDContextExtractor) *SecretStore {
	return &SecretStore{
		store:     store,
		extractor: extractor,
	}
}

// GetSecretValues implements the common.SecretStore interface.
func (s *SecretStore) GetSecretValues(ctx context.Context, secretID string) (map[string]string, error) {
	organizationID, secretID, err := s.extractOrganizationID(ctx, secretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to extract orgID")
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

	return secretResponse.Values, nil
}

func (s *SecretStore) Store(ctx context.Context, request *secret.CreateSecretRequest) (string, error) {
	organizationID, ok := s.extractor.GetOrganizationID(ctx)
	if !ok {
		return "", errors.NewWithDetails(
			"organization ID cannot be found in the context",
			"organizationId", organizationID,
		)
	}

	secretID, err := s.store.Store(organizationID, request)
	if err != nil {
		return "", errors.WrapIf(err, "failed to store secret in Vault")
	}

	return secretID, nil
}

func (s *SecretStore) Delete(ctx context.Context, secretID string) error {
	organizationID, ok := s.extractor.GetOrganizationID(ctx)
	if !ok {
		return errors.NewWithDetails(
			"organization ID cannot be found in the context",
			"organizationId", organizationID,
			"secretID", secretID,
		)
	}

	return s.store.Delete(organizationID, secretID)
}

func (s *SecretStore) GetNameByID(ctx context.Context, secretID string) (string, error) {
	organizationID, secretID, err := s.extractOrganizationID(ctx, secretID)
	if err != nil {
		return "", errors.WrapIf(err, "failed to extract orgID")
	}

	secretResponse, err := s.store.Get(organizationID, secretID)
	if err == secret.ErrSecretNotExists {
		return "", errors.WrapIfWithDetails(err, "failed to get secret by ID", "secretID", secretID)
	}
	if err != nil {
		return "", errors.WithDetails(errors.WithStackIf(err), "organizationID", organizationID, "secretID", secretID)
	}

	return secretResponse.Name, nil
}

func (s *SecretStore) GetIDByName(ctx context.Context, secretName string) (string, error) {
	organizationID, ok := s.extractor.GetOrganizationID(ctx)
	if !ok {
		return "", errors.NewWithDetails(
			"organization ID cannot be found in the context",
			"organizationId", organizationID,
			"secretName", secretName,
		)
	}

	secretResponse, err := s.store.GetByName(organizationID, secretName)
	if err == secret.ErrSecretNotExists {
		return "", errors.WrapIfWithDetails(err, "failed to get secret by name", "secretName", secretName)
	}
	if err != nil {
		return "", errors.WithDetails(errors.WithStackIf(err), "organizationID", organizationID, "secretName", secretName)
	}

	return secretResponse.ID, nil
}

func (s *SecretStore) extractOrganizationID(ctx context.Context, secretID string) (uint, string, error) {
	var organizationID uint

	if brn.IsBRN(secretID) {
		rn, err := brn.ParseAs(secretID, brn.SecretResourceType)
		if err != nil {
			return 0, "", err
		}

		organizationID = rn.OrganizationID
		secretID = rn.ResourceID
	} else { // fall back to organization extracted from context
		var ok bool
		organizationID, ok = s.extractor.GetOrganizationID(ctx)
		if !ok {
			return 0, "", errors.NewWithDetails(
				"organization ID cannot be found in the context",
				"organizationId", organizationID,
				"secretId", secretID,
			)
		}
	}

	return organizationID, secretID, nil
}
