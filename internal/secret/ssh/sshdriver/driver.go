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

package sshdriver

import (
	"fmt"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/internal/secret/ssh"
	"github.com/banzaicloud/pipeline/src/secret"
)

// StoreSSHKeyPair to store SSH Key to Bank Vaults
func StoreSSHKeyPair(key ssh.KeyPair, organizationID uint, clusterID uint, clusterName string, clusterUID string) (string, error) {
	createSecretRequest := secret.CreateSecretRequest{
		Name: fmt.Sprint("ssh-cluster-", clusterID),
		Type: secrettype.SSHSecretType,
		Tags: []string{
			"cluster:" + clusterName,
			fmt.Sprintf("clusterUID:%s", clusterUID),
			secret.TagBanzaiReadonly,
		},
		Values: map[string]string{
			secrettype.User:                 key.User,
			secrettype.Identifier:           key.Identifier,
			secrettype.PublicKeyData:        key.PublicKeyData,
			secrettype.PublicKeyFingerprint: key.PublicKeyFingerprint,
			secrettype.PrivateKeyData:       key.PrivateKeyData,
		},
	}

	secretID, err := secret.Store.Store(organizationID, &createSecretRequest)
	return secretID, errors.WrapIf(err, "failed to store SSH secret in secret store")
}
