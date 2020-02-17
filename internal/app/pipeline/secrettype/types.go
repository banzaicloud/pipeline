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

package secrettype

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

// TypeDefinition represents a secret type definition.
type TypeDefinition struct {
	Fields []TypeField `json:"fields"`
}

// TypeField represents the fields in a secret.
type TypeField struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Opaque      bool   `json:"opaque,omitempty"`
	Description string `json:"description,omitempty"`
}

//go:generate mga gen mockery --name Service --inpkg
// +kit:endpoint:errorStrategy=service

// Service provides information about secret types.
type Service interface {
	// ListSecretTypes lists secret type definitions.
	ListSecretTypes(ctx context.Context) (secretTypes map[string]TypeDefinition, err error)

	// GetSecretType returns a single secret type definition.
	GetSecretType(ctx context.Context, secretType string) (secretTypeDef TypeDefinition, err error)
}

// NewService returns a new Service.
func NewService() Service {
	return service{types: types}
}

type service struct {
	types map[string]TypeDefinition
}

// nolint: gochecknoglobals
var types map[string]TypeDefinition

// Load type definitions of the secrettype package.
func init() {
	types = make(map[string]TypeDefinition, len(secrettype.DefaultRules))

	for key, st := range secrettype.DefaultRules {
		var typeDef TypeDefinition
		for _, field := range st.Fields {
			typeDef.Fields = append(typeDef.Fields, TypeField(field))
		}

		types[key] = typeDef
	}
}

func (t service) ListSecretTypes(ctx context.Context) (map[string]TypeDefinition, error) {
	return t.types, nil
}

// ErrNotSupportedSecretType describe an error if the secret type is not supported.
var ErrNotSupportedSecretType = errors.Sentinel("not supported secret type")

func (t service) GetSecretType(ctx context.Context, secretType string) (TypeDefinition, error) {
	typeDef, ok := t.types[secretType]
	if !ok {
		return TypeDefinition{}, errors.WithStack(ErrNotSupportedSecretType)
	}

	return typeDef, nil
}
