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
	"github.com/banzaicloud/pipeline/internal/secret"
)

const SSH = "ssh"

const (
	FieldSSHUser                 = "user"
	FieldSSHIdentifier           = "identifier"
	FieldSSHPublicKeyData        = "public_key_data"
	FieldSSHPublicKeyFingerprint = "public_key_fingerprint"
	FieldSSHPrivateKeyData       = "private_key_data"
)

type SSHType struct{}

func (SSHType) Name() string {
	return SSH
}

func (SSHType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldSSHUser, Required: true, IsSafeToDisplay: true},
			{Name: FieldSSHIdentifier, Required: true, IsSafeToDisplay: true},
			{Name: FieldSSHPublicKeyData, Required: true, IsSafeToDisplay: true},
			{Name: FieldSSHPublicKeyFingerprint, Required: true, IsSafeToDisplay: true},
			{Name: FieldSSHPrivateKeyData, Required: true},
		},
	}
}

func (t SSHType) Validate(data map[string]string) error {
	return validateDefinition(data, t.Definition())
}
