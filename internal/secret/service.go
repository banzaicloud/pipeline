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

package secret

import (
	"context"
	"time"
)

// NotFoundError is returned when a secret cannot be found.
type NotFoundError struct {
	OrganizationID uint
	SecretID       string
}

// Error implements the error interface.
func (NotFoundError) Error() string {
	return "secret not found"
}

// Details returns error details.
func (e NotFoundError) Details() []interface{} {
	return []interface{}{"organizationId", e.OrganizationID, "secretId", e.SecretID}
}

// NotFound tells a consumer that this error is related to a resource being not found.
// Can be used to translate the error to the consumer's response format (eg. status codes).
func (NotFoundError) NotFound() bool {
	return true
}

// ServiceError tells the consumer that this is a business error and it should be returned to the client.
// Non-service errors are usually translated into "internal" errors.
func (NotFoundError) ServiceError() bool {
	return true
}

// AlreadyExistsError is returned when a secret already exists in the store.
type AlreadyExistsError struct {
	OrganizationID uint
	SecretID       string
}

// Error implements the error interface.
func (AlreadyExistsError) Error() string {
	return "secret already exists"
}

// Details returns error details.
func (e AlreadyExistsError) Details() []interface{} {
	return []interface{}{"organizationId", e.OrganizationID, "secretId", e.SecretID}
}

// Conflict tells the consumer that this error is related to a conflicting request.
// Can be used to translate the error to the consumer's response format (eg. status codes).
func (AlreadyExistsError) Conflict() bool {
	return true
}

// ServiceError tells the consumer that this is a business error and it should be returned to the client.
// Non-service errors are usually translated into "internal" errors.
func (AlreadyExistsError) ServiceError() bool {
	return true
}

// Model is an internal, low-level representation of a secret.
type Model struct {
	ID        string            `mapstructure:"-"`
	Name      string            `mapstructure:"name"`
	Type      string            `mapstructure:"type"`
	Values    map[string]string `mapstructure:"values"`
	Tags      []string          `mapstructure:"tags"`
	UpdatedAt time.Time         `mapstructure:"-"`
	UpdatedBy string            `mapstructure:"updatedBy"`
}

// Store is a low-level interface for a key-value like secret store.
type Store interface {
	// Create writes a new secret in the store.
	//
	// Compared to Put, Create returns a AlreadyExistsError if the secret already exists.
	Create(ctx context.Context, organizationID uint, model Model) error

	// Put updates an existing secret or writes a new one in the store.
	Put(ctx context.Context, organizationID uint, model Model) error

	// Get retrieves a secret from the store.
	Get(ctx context.Context, organizationID uint, id string) (Model, error)

	// List lists secrets in the store.
	List(ctx context.Context, organizationID uint) ([]Model, error)

	// Delete deletes a secret from the store.
	Delete(ctx context.Context, organizationID uint, id string) error
}
