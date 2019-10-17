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
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"

	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
)

// SSHKeyPair struct to store SSH key data
type SSHKeyPair struct {
	User                 string `json:"user,omitempty"`
	Identifier           string `json:"identifier,omitempty"`
	PublicKeyData        string `json:"publicKeyData,omitempty"`
	PublicKeyFingerprint string `json:"publicKeyFingerprint,omitempty"`
	PrivateKeyData       string `json:"PrivateKeyData,omitempty"`
}

// NewSSHKeyPair constructs a SSH Key from the values stored
// in the given secret
func NewSSHKeyPair(s *SecretItemResponse) *SSHKeyPair {
	return &SSHKeyPair{
		User:                 s.Values[secretTypes.User],
		Identifier:           s.Values[secretTypes.Identifier],
		PublicKeyData:        s.Values[secretTypes.PublicKeyData],
		PublicKeyFingerprint: s.Values[secretTypes.PublicKeyFingerprint],
		PrivateKeyData:       s.Values[secretTypes.PrivateKeyData],
	}
}

// StoreSSHKeyPair to store SSH Key to Bank Vaults
func StoreSSHKeyPair(key *SSHKeyPair, organizationID uint, clusterID uint, clusterName string, clusterUID string) (secretID string, err error) {
	log.Info("Store SSH Key to Bank Vaults")
	var createSecretRequest CreateSecretRequest
	createSecretRequest.Type = secretTypes.SSHSecretType
	createSecretRequest.Name = fmt.Sprint("ssh-cluster-", clusterID)

	clusterUidTag := fmt.Sprintf("clusterUID:%s", clusterUID)
	createSecretRequest.Tags = []string{
		"cluster:" + clusterName,
		clusterUidTag,
		TagBanzaiReadonly,
	}

	createSecretRequest.Values = map[string]string{
		secretTypes.User:                 key.User,
		secretTypes.Identifier:           key.Identifier,
		secretTypes.PublicKeyData:        key.PublicKeyData,
		secretTypes.PublicKeyFingerprint: key.PublicKeyFingerprint,
		secretTypes.PrivateKeyData:       key.PrivateKeyData,
	}

	secretID, err = Store.Store(organizationID, &createSecretRequest)

	if err != nil {
		log.Errorf("Error during store: %s", err.Error())
		return "", err
	}

	log.Info("SSH Key stored.")
	return
}

// GenerateSSHKeyPair for Generate new SSH Key pair
func GenerateSSHKeyPair() (*SSHKeyPair, error) {
	log.Info("Generate new SSH key")

	key := new(SSHKeyPair)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Errorf("PrivateKey generator failed reason: %s", err.Error())
		return key, err
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	keyBuff := new(bytes.Buffer)
	if err := pem.Encode(keyBuff, privateKeyPEM); err != nil {
		log.Errorf("PrivateKey generator failed reason: %s", err.Error())
		return key, err
	}
	key.PrivateKeyData = keyBuff.String()
	log.Debug("Private key generated.")

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Errorf("PublicKey generator failed reason: %s", err.Error())
		return key, err
	}
	log.Debug("Public key generated.")

	key.PublicKeyData = fmt.Sprintf("%s %s \n", strings.TrimSuffix(string(ssh.MarshalAuthorizedKey(pub)), "\n"), "no-reply@banzaicloud.com")

	key.PublicKeyFingerprint = ssh.FingerprintSHA256(pub)
	log.Info("SSH key generated.")

	return key, nil
}
