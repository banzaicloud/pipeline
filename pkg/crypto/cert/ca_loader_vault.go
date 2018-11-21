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

package cert

import (
	"crypto"
	"crypto/x509"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// VaultCALoader loads a CA bundle from Vault.
type VaultCALoader struct {
	client *api.Logical
	path   string
}

// NewVaultCALoader returns a new VaultCALoader.
func NewVaultCALoader(client *api.Logical, path string) *VaultCALoader {
	return &VaultCALoader{
		client: client,
		path:   path,
	}
}

func (s *VaultCALoader) Load() (*x509.Certificate, crypto.Signer, error) {
	secret, err := s.client.Read(s.path)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read CA bundle")
	}

	if secret == nil || secret.Data == nil {
		return nil, nil, errors.Wrap(err, "CA bundle not found")
	}

	caBundle := cast.ToStringMapString(secret.Data["data"])

	cert, ok := caBundle["caCert"]
	if !ok {
		return nil, nil, errors.Wrap(err, "certificate not found in the CA bundle")
	}

	key, ok := caBundle["caKey"]
	if !ok {
		return nil, nil, errors.Wrap(err, "key not found in the CA bundle")
	}

	return parseCABundle([]byte(cert), []byte(key))
}
