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
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"time"

	"github.com/banzaicloud/bank-vaults/pkg/tls"
	"github.com/pkg/errors"
)

const (
	validity = 365 * 24 * time.Hour
)

// Generator generates a self-signed cert-key pair.
type Generator struct {
	signerCert *x509.Certificate
	signerKey  crypto.Signer
}

// NewGenerator returns a new Generator instance.
func NewGenerator(
	signerCert *x509.Certificate,
	signerKey crypto.Signer,
) *Generator {
	return &Generator{
		signerCert: signerCert,
		signerKey:  signerKey,
	}
}

// NewGeneratorFromFile creates a generator and loads CA cert and key from file.
func NewGeneratorFromFile(
	signerCertPath string,
	signerKeyPath string,
) (*Generator, error) {
	signerCertFile, err := ioutil.ReadFile(signerCertPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read CA cert file")
	}

	signerCertPem, _ := pem.Decode(signerCertFile)
	if signerCertPem == nil {
		return nil, errors.New("failed to pem-decode CA cert")
	}

	signerCert, err := x509.ParseCertificate(signerCertPem.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse CA cert")
	}

	signerKeyFile, err := ioutil.ReadFile(signerKeyPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read CA key file")
	}

	signerKeyPem, _ := pem.Decode(signerKeyFile)
	if signerKeyPem == nil {
		return nil, errors.New("failed to pem-decode CA key")
	}

	signerKey, err := parsePrivateKey(signerKeyPem.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse CA key")
	}

	return NewGenerator(signerCert, signerKey), nil
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

// CertificateRequest contains a minimal set of information to generate a self-signed certificate.
type CertificateRequest struct {
	CommonName       string
	AlternativeNames []string
}

// Generate generates a self-signed cert-key pair.
func (g *Generator) Generate(request CertificateRequest) ([]byte, []byte, error) {
	certRequest := tls.ServerCertificateRequest{
		Subject: pkix.Name{
			CommonName: request.CommonName,
		},
		DNSNames: request.AlternativeNames,
		Validity: validity,
	}
	cert, err := tls.GenerateServerCertificate(certRequest, g.signerCert, g.signerKey)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "failed to generate server certificate")
	}

	return cert.Certificate, cert.Key, nil
}
