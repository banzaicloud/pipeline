// Copyright © 2018 Banzai Cloud
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
	"fmt"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/internal/secret/ssh"
)

// NewSSHKeyPair constructs a SSH Key from the values stored
// in the given secret
func NewSSHKeyPair(s *SecretItemResponse) ssh.KeyPair {
	return ssh.KeyPair{
		User:                 s.Values[secrettype.User],
		Identifier:           s.Values[secrettype.Identifier],
		PublicKeyData:        s.Values[secrettype.PublicKeyData],
		PublicKeyFingerprint: s.Values[secrettype.PublicKeyFingerprint],
		PrivateKeyData:       s.Values[secrettype.PrivateKeyData],
	}
}

// StoreSSHKeyPair to store SSH Key to Bank Vaults
func StoreSSHKeyPair(key ssh.KeyPair, organizationID uint, clusterID uint, clusterName string, clusterUID string) (secretID string, err error) {
	log.Info("Store SSH Key to Bank Vaults")
	var createSecretRequest CreateSecretRequest
	createSecretRequest.Type = secrettype.SSHSecretType
	createSecretRequest.Name = fmt.Sprint("ssh-cluster-", clusterID)

	clusterUidTag := fmt.Sprintf("clusterUID:%s", clusterUID)
	createSecretRequest.Tags = []string{
		"cluster:" + clusterName,
		clusterUidTag,
		TagBanzaiReadonly,
	}

	createSecretRequest.Values = map[string]string{
		secrettype.User:                 key.User,
		secrettype.Identifier:           key.Identifier,
		secrettype.PublicKeyData:        key.PublicKeyData,
		secrettype.PublicKeyFingerprint: key.PublicKeyFingerprint,
		secrettype.PrivateKeyData:       key.PrivateKeyData,
	}

	secretID, err = Store.Store(organizationID, &createSecretRequest)

	if err != nil {
		log.Errorf("Error during store: %s", err.Error())
		return "", err
	}

	log.Info("SSH Key stored.")
	return
}
