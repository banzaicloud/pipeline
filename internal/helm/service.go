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

package helm

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
)

type Logger = common.Logger

// Repository represents a Helm chart repository.
type Repository struct {
	// Name is a unique identifier for the repository.
	Name string `json:"name"`

	// URL is the repository URL.
	URL string `json:"url"`

	// PasswordSecretID is the identifier of a password type secret that contains the credentials for a repository.
	PasswordSecretID string `json:"passwordSecretId,omitempty"`

	// TlsSecretID is the identifier of a TLS secret.
	//
	// If there is a Certificate Authority in the secret,
	// it will be used to verify the certificate presented by the repository server.
	//
	// If there is a client key pair in the secret,
	// it will be presented to the repository server.
	TlsSecretID string `json:"tlsSecretId,omitempty"`
}

// +kit:endpoint:errorStrategy=service
// +testify:mock:testOnly=true

// Service manages Helm chart repositories.
type Service interface {
	// AddRepository adds a new Helm chart repository.
	AddRepository(ctx context.Context, organizationID uint, repository Repository) error
	// ListRepositories lists Helm repositories.
	ListRepositories(ctx context.Context, organizationID uint) (repos []Repository, err error)
	// ListRepositories deletes a Helm repository
	DeleteRepository(ctx context.Context, organizationID uint, repoName string) error
	// UpdateRepository updates an existing repository
	UpdateRepository(ctx context.Context, organizationID uint, repository Repository) error
}

// NewService returns a new Service.
func NewService(store Store, secretStore SecretStore, validator RepoValidator, envService Service, logger Logger) Service {
	return service{
		store:         store,
		secretStore:   secretStore,
		repoValidator: validator,
		envService:    envService,
		logger:        logger,
	}
}

// +testify:mock:testOnly=true

// Store interface abstracting persistence operations
type Store interface {
	// Create persists the repository item for the given organisation
	Create(ctx context.Context, organizationID uint, repository Repository) error

	// Delete persists the repository item for the given organisation
	Delete(ctx context.Context, organizationID uint, repository Repository) error

	//List retrieves persisted repositories for the given organisation
	List(ctx context.Context, organizationID uint) ([]Repository, error)

	//Getretrieves a repository entry
	Get(ctx context.Context, organizationID uint, repository Repository) (Repository, error)

	// Update updates the given repository
	Update(ctx context.Context, organizationID uint, repository Repository) error
}

type PasswordSecret struct {
	UserName string
	Password string
}

type TlsSecret struct {
	CAFile   string
	CertFile string
	KeyFile  string
}

// +testify:mock:testOnly=true

// SecretStore abstracts secret related operations
type SecretStore interface {
	// CheckPasswordSecret checks the existence and the type of the secret
	CheckPasswordSecret(ctx context.Context, secretID string) error
	// CheckTLSSecret checks the existence and the type of the secret
	CheckTLSSecret(ctx context.Context, secretID string) error
	// ResolvePasswordSecrets resolves the password type secret values
	ResolvePasswordSecrets(ctx context.Context, secretID string) (PasswordSecret, error)
	// ResolveTlsSecrets resolves the tls type secret values
	ResolveTlsSecrets(ctx context.Context, secretID string) (TlsSecret, error)
}

type service struct {
	store         Store
	secretStore   SecretStore
	repoValidator RepoValidator
	envService    Service
	logger        Logger
}

func (s service) AddRepository(ctx context.Context, organizationID uint, repository Repository) error {
	// validate repository
	if err := s.repoValidator.Validate(ctx, repository); err != nil {
		return errors.WrapIf(err, "failed to add new helm repository")
	}

	if repository.PasswordSecretID != "" {
		if err := s.secretStore.CheckPasswordSecret(ctx, repository.PasswordSecretID); err != nil {
			return ValidationError{message: err.Error(), violations: []string{"password secret must exist"}}
		}
	}

	if repository.TlsSecretID != "" {
		if err := s.secretStore.CheckTLSSecret(ctx, repository.TlsSecretID); err != nil {
			return ValidationError{message: err.Error(), violations: []string{"tls secret must exist"}}
		}
	}

	exists, err := s.repoExists(ctx, organizationID, repository)
	if err != nil {
		return errors.WrapIf(err, "failed to add helm repository")
	}

	if exists {
		return AlreadyExistsError{
			RepositoryName: repository.Name,
			OrganizationID: organizationID,
		}
	}

	// save in store
	if err := s.store.Create(ctx, organizationID, repository); err != nil {
		return errors.WrapIf(err, "failed to add helm repository")
	}

	if err := s.envService.AddRepository(ctx, organizationID, repository); err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	s.logger.Debug("created helm repository", map[string]interface{}{"orgID": organizationID, "helm repository": repository.Name})
	return nil
}

func (s service) ListRepositories(ctx context.Context, organizationID uint) (repos []Repository, err error) {
	return s.store.List(ctx, organizationID)
}

func (s service) DeleteRepository(ctx context.Context, organizationID uint, repoName string) error {
	repoExists, err := s.repoExists(ctx, organizationID, Repository{Name: repoName})
	if err != nil {
		return err
	}

	if !repoExists {
		return nil
	}

	// delete the environment
	if err := s.envService.DeleteRepository(ctx, organizationID, repoName); err != nil {
		return errors.WrapIf(err, "failed to delete helm repository environment")
	}

	// delete form the persistent store
	if err := s.store.Delete(ctx, organizationID, Repository{Name: repoName}); err != nil {
		return errors.WrapIf(err, "failed to delete helm repository")
	}

	s.logger.Debug("deleted helm repository", map[string]interface{}{"orgID": organizationID, "helm repository": repoName})
	return nil
}

func (s service) UpdateRepository(ctx context.Context, organizationID uint, repository Repository) error {

	// validate repository
	if err := s.repoValidator.Validate(ctx, repository); err != nil {
		return errors.WrapIf(err, "failed to add new helm repository")
	}

	if repository.PasswordSecretID != "" {
		if err := s.secretStore.CheckPasswordSecret(ctx, repository.PasswordSecretID); err != nil {
			return ValidationError{message: err.Error(), violations: []string{"password secret must exist"}}
		}
	}

	if repository.TlsSecretID != "" {
		if err := s.secretStore.CheckTLSSecret(ctx, repository.TlsSecretID); err != nil {
			return ValidationError{message: err.Error(), violations: []string{"tls secret must exist"}}
		}
	}

	exists, err := s.repoExists(ctx, organizationID, Repository{Name: repository.Name})
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to retrieve helm repository",
			"orgID", organizationID, "repoName", repository.Name)
	}

	if !exists {
		return NotFoundError{
			RepositoryName: repository.Name,
			OrganizationID: organizationID,
		}
	}

	// save in store
	if err := s.store.Update(ctx, organizationID, repository); err != nil {
		return errors.WrapIf(err, "failed to add helm repository")
	}

	if err := s.envService.AddRepository(ctx, organizationID, repository); err != nil {
		return errors.WrapIf(err, "failed to set up helm repository environment")
	}

	s.logger.Debug("created helm repository", map[string]interface{}{"orgID": organizationID, "helm repository": repository.Name})
	return nil
}

func (s service) repoExists(ctx context.Context, orgID uint, repository Repository) (bool, error) {
	_, err := s.store.Get(ctx, orgID, repository)

	if err != nil {
		// TODO refine this implementation, separate results by error type
		return false, nil
	}

	return true, nil
}
