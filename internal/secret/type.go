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

// Type describes a secret type.
type Type interface {
	// Name is the type name.
	Name() string

	// Definition returns a descriptor for the secret type.
	//
	// Definition is currently used by clients and internally for validating certain types.
	Definition() TypeDefinition

	// Validate validates a secret.
	Validate(data map[string]string) error
}

// TypeDefinition describes the structure of a secret type.
type TypeDefinition struct {
	Fields []FieldDefinition `json:"fields"`
}

// FieldDefinition describes a secret field.
type FieldDefinition struct {
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	Required        bool   `json:"required"`
	IsSafeToDisplay bool   `json:"IsSafeToDisplay,omitempty"`
	Opaque          bool   `json:"opaque,omitempty"`
}

// GeneratorType can be implemented by a secret type that adds secret generation abilities to the type.
//
// When a type can generate secrets, a secret should be validated differently.
type GeneratorType interface {
	// ValidateNew validates a new, potentially incomplete secret.
	//
	// If the first returned result is false, the secret is incomplete and needs generation.
	ValidateNew(data map[string]string) (bool, error)

	// Generate generates values for the secret.
	//
	// Note: organizationID, secretName and tags are added for the PKE type.
	Generate(organizationID uint, secretName string, data map[string]string, tags []string) (map[string]string, error)
}

// ProcessorType can be implemented by a secret type that adds secret processing abilities to the type.
//
// Secret processing is done when a secret is created or updated (eg. making sure a secret is in a specific format).
type ProcessorType interface {
	// Process processes values for the secret.
	Process(data map[string]string) (map[string]string, error)
}

// VerifierType can be implemented by a secret type that adds secret verification abilities to the type.
//
// Verification can check if credentials are actually valid (ie. can access a remote service).
type VerifierType interface {
	// Verify verifies a secret.
	Verify(data map[string]string) error
}

// CleanupType can be implemented by a secret type that adds secret cleanup abilities to the type.
//
// This is added temporarily for PKE secret type.
type CleanupType interface {
	// Cleanup is called before a secret is deleted to allow the type to clean up any resources used for the secret.
	Cleanup(organizationID uint, data map[string]string, tags []string) error
}

// TypeList is an accessor to a list of secret types.
type TypeList struct {
	types   []Type
	typeMap map[string]Type
}

// NewTypeList returns a new TypeList.
func NewTypeList(types []Type) TypeList {
	typeMap := make(map[string]Type, len(types))

	for _, typ := range types {
		typeMap[typ.Name()] = typ
	}

	return TypeList{
		types:   types,
		typeMap: typeMap,
	}
}

// Types returns the list of secret types.
func (t TypeList) Types() []Type {
	return t.types
}

// Type returns a type from the list (if it exists).
func (t TypeList) Type(typ string) Type {
	return t.typeMap[typ]
}
