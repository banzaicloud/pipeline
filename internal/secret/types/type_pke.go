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

package types

import (
	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/secret"
)

const PKE = "pkecert"

const (
	FieldPKECACert           = "caCert"
	FieldPKECAKey            = "caKey"
	FieldPKEKubernetesCACert = "kubernetesCaCert"
	FieldPKEKubernetesCAKey  = "kubernetesCaKey"
	FieldPKEEtcdCACert       = "etcdCaCert"
	FieldPKEEtcdCAKey        = "etcdCaKey"
	FieldPKEFrontProxyCACert = "frontProxyCaCert"
	FieldPKEFrontProxyCAKey  = "frontProxyCaKey"
	FieldPKESAPub            = "saPub"
	FieldPKESAKey            = "saKey"
)

// PkeSecreter is a temporary interface for splitting the PKE secret generation/deletion code from the legacy secret store.
type PkeSecreter interface {
	GeneratePkeSecret(organizationID uint, tags []string) (map[string]string, error)
	DeletePkeSecret(organizationID uint, tags []string) error
}

type PKEType struct {
	PkeSecreter PkeSecreter
}

func (PKEType) Name() string {
	return PKE
}

func (PKEType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldPKECACert, Required: false},
			{Name: FieldPKECAKey, Required: false},
			{Name: FieldPKEKubernetesCACert, Required: false},
			{Name: FieldPKEKubernetesCAKey, Required: false},
			{Name: FieldPKEEtcdCACert, Required: false},
			{Name: FieldPKEEtcdCAKey, Required: false},
			{Name: FieldPKEFrontProxyCACert, Required: false},
			{Name: FieldPKEFrontProxyCAKey, Required: false},
			{Name: FieldPKESAPub, Required: false},
			{Name: FieldPKESAKey, Required: false},
		},
	}
}

// PKE secret is always generated. It's always valid.
func (t PKEType) Validate(_ map[string]string) error {
	return nil
}

// PKE secret is always generated. It's always valid and incomplete.
func (PKEType) ValidateNew(_ map[string]string) (bool, error) {
	return false, nil
}

func (t PKEType) Generate(organizationID uint, _ string, _ map[string]string, tags []string) (map[string]string, error) {
	generatedData, err := t.PkeSecreter.GeneratePkeSecret(organizationID, tags)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to generate PKE certs")
	}

	return generatedData, nil
}

func (t PKEType) Cleanup(organizationID uint, _ map[string]string, tags []string) error {
	err := t.PkeSecreter.DeletePkeSecret(organizationID, tags)
	if err != nil {
		return err
	}

	return nil
}
