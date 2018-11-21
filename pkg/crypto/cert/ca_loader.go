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
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/pkg/errors"
)

// CALoader fetches a parent certificate and signing key.
type CALoader interface {
	// Load fetches a parent certificate and signing key.
	Load() (*x509.Certificate, crypto.Signer, error)
}

func parseCABundle(certBytes []byte, keyBytes []byte) (*x509.Certificate, crypto.Signer, error) {
	certPem, _ := pem.Decode(certBytes)
	if certPem == nil {
		return nil, nil, errors.New("failed to pem-decode certificate")
	}

	cert, err := x509.ParseCertificate(certPem.Bytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse certificate")
	}

	keyPem, _ := pem.Decode(keyBytes)
	if keyPem == nil {
		return nil, nil, errors.New("failed to pem-decode key")
	}

	key, err := parsePrivateKey(keyPem.Bytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse key")
	}

	return cert, key, nil
}

func parsePrivateKey(der []byte) (crypto.Signer, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey:
			return key, nil

		case *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}

	return nil, errors.New("failed to parse private key")
}
