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

package common

import (
	"context"

	"github.com/banzaicloud/pipeline/secret"
)

// SecretStore is a common interface for various parts of the application
// to read secrets from the platform's secret store.
//
// It is not supposed to expose any implementation specific details.
// If lower level access is required, use a different interface.
type SecretStore interface {
	// GetSecretValues returns the values stored within a secret.
	// If the underlying store uses additional keys for determining the exact secret path
	// (eg. organization ID), it should be retrieved from the context.
	GetSecretValues(ctx context.Context, secretID string) (map[string]string, error)

	Store(ctx context.Context, request *secret.CreateSecretRequest) (string, error)

	GetNameByID(ctx context.Context, secretID string) (string, error)

	GetIDByName(ctx context.Context, secretName string) (string, error)

	Delete(ctx context.Context, secretID string) error
}

// SecretNotFoundError is returned from a SecretStore if a secret cannot be found.
type SecretNotFoundError struct {
	SecretID string
}

// Error implements the builtin error interface.
func (SecretNotFoundError) Error() string {
	return "secret not found"
}

// Details returns details about the error in a generic, loggable format.
func (e SecretNotFoundError) Details() []interface{} {
	return []interface{}{"secretId", e.SecretID}
}
