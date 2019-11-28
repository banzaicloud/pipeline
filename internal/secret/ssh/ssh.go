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

package ssh

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"strings"

	"emperror.dev/errors"
	"golang.org/x/crypto/ssh"
)

type KeyPair struct {
	User                 string
	Identifier           string
	PublicKeyData        string
	PublicKeyFingerprint string
	PrivateKeyData       string
}

type KeyPairGenerator struct {
	Bits    int
	Comment string
	Random  io.Reader
}

func NewKeyPairGenerator() KeyPairGenerator {
	return KeyPairGenerator{
		Random:  rand.Reader,
		Bits:    2048,
		Comment: "no-reply@banzaicloud.com",
	}
}

func (g KeyPairGenerator) Generate() (KeyPair, error) {
	var keyPair KeyPair

	privateKey, err := rsa.GenerateKey(g.Random, g.Bits)
	if err != nil {
		return keyPair, errors.WrapIf(err, "failed to generate RSA private key")
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	privateKeyPEMBuff := new(bytes.Buffer)
	if err := pem.Encode(privateKeyPEMBuff, privateKeyPEM); err != nil {
		return keyPair, errors.WrapIf(err, "failed to encode private key as PEM")
	}

	keyPair.PrivateKeyData = privateKeyPEMBuff.String()

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return keyPair, errors.WrapIf(err, "failed to convert RSA public key to SSH public key")
	}

	keyPair.PublicKeyData = fmt.Sprintf("%s %s \n", strings.TrimSuffix(string(ssh.MarshalAuthorizedKey(publicKey)), "\n"), g.Comment)

	keyPair.PublicKeyFingerprint = ssh.FingerprintSHA256(publicKey)

	return keyPair, nil
}
