// Copyright © 2020 Banzai Cloud
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

package helm3

import (
	"context"
	"fmt"
	"net/url"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

// Repository represents a Helm chart repository.
type Repository struct {
	// Name is a unique identifier for the repository.
	Name string `json:"name"`

	// URL is the repository URL.
	URL string `json:"url"`

	// PasswordSecretID is the identifier of a password type secret that contains the credentials for a repository.
	PasswordSecretID string `json:"passwordSecretId"`

	// TlsSecretID is the identifier of a TLS secret.
	//
	// If there is a Certificate Authority in the secret,
	// it will be used to verify the certificate presented by the repository server.
	//
	// If there is a client key pair in the secret,
	// it will be presented to the repository server.
	TlsSecretID string `json:"tlsSecretId"`
}

func (r Repository) Validate() error {
	var violations []string

	if r.Name == "" {
		violations = append(violations, "name cannot be empty")
	}

	// name matches a regex

	_, err := url.Parse(r.URL)
	if err != nil {
		violations = append(violations, fmt.Sprintf("invalid repository URL: %s", err.Error()))
	}

	if len(violations) > 0 {
		return errors.WithStack(cluster.NewValidationError("invalid chart repository", violations))
	}

	return nil
}

func validate() error {
	// name is not empty
	// name matches a regex
	// url is a valid URL

	// external validations:
	// index is available
	// password secret exists (if provided)
	// tls secret exists (if provided)

	return nil
}

// Service manages Helm chart repositories.
type Service interface {
	// AddRepository adds a new Helm chart repository.
	AddRepository(ctx context.Context, organizationID uint, repository Repository) error

	// ListRepositories lists Helm repositories.
	ListRepositories(ctx context.Context, organizationID uint) (repos []Repository, err error)

	// DeleteRepository(ctx context.Context, organizationID uint, repoName string) error

	// GetRepositoryIndex(ctx context.Context, organizationID uint, repoName string) (index []byte, err error)
	// PurgeIndexCache(ctx context.Context, organizationID uint, repoName string) error
}

// NewService returns a new Service.
func NewService() Service {
	return service{}
}

type service struct {
	store       Store
	secretStore SecretStore
}

// Store interface abstracting persistence operations
type Store interface {
	// AddRepository persists the repository item for the given organisation
	AddRepository(ctx context.Context, organizationID uint, repository Repository) error

	// DeleteRepository persists the repository item for the given organisation
	DeleteRepository(ctx context.Context, organizationID uint, repository Repository) error

	//ListRepositories retrieves persisted repositories for the given organisation
	ListRepositories(ctx context.Context, organizationID uint) ([]Repository, error)

	GetRepository(ctx context.Context, organizationID uint, repository Repository) (Repository, error)
}

type SecretStore interface {
	CheckPasswordSecret(ctx context.Context, secretID string) error
	CheckTLSSecret(ctx context.Context, secretID string) error
}

func (s service) AddRepository(ctx context.Context, organizationID uint, repository Repository) error {
	// todo error response strategy: collect all or return fast
	// validate repository
	if err := repository.Validate(); err != nil {
		return errors.WrapIf(err, "failed to add new helm repository")
	}

	if repository.PasswordSecretID != "" {
		if err := s.secretStore.CheckPasswordSecret(ctx, repository.PasswordSecretID); err != nil {
			return errors.WrapIf(err, "failed to add new helm repository")
		}
	}

	if repository.TlsSecretID != "" {
		if err := s.secretStore.CheckTLSSecret(ctx, repository.PasswordSecretID); err != nil {
			return errors.WrapIf(err, "failed to add new helm repository")
		}
	}

	// check record existence
	if _, err := s.store.GetRepository(ctx, organizationID, repository); err == nil {
		return errors.WrapIf(err, "helm repository already exists")
	}

	// validate repository index? todo

	// save in store
	if err := s.store.AddRepository(ctx, organizationID, repository); err != nil {
		return errors.WrapIf(err, "failed to persist new helm repository")
	}

	return nil
}

func (s service) ListRepositories(ctx context.Context, organizationID uint) (repos []Repository, err error) {
	return s.store.ListRepositories(ctx, organizationID)
}
