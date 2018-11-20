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
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
		"io"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

const (
	privateKeyBits = 2048
	validity       = 365 * 24 * time.Hour
)

type Clock interface {
	Now() time.Time
}

// TODO: install tardis package
// SystemClock uses the real time.
var SystemClock = NewSystemClock()

type systemClock struct{}

// NewSystemClock returns a clock that uses real time.
func NewSystemClock() Clock {
	return &systemClock{}
}

// Now tells the current time.
func (*systemClock) Now() time.Time {
	return time.Now()
}

// Generator generates a self-signed cert-key pair.
type Generator struct {
	signerCert *x509.Certificate
	signerKey  crypto.Signer

	clock      Clock
	randReader io.Reader
}

// NewGenerator returns a new Generator instance.
func NewGenerator(
	signerCert *x509.Certificate,
	signerKey crypto.Signer,
	clock Clock,
	randReader io.Reader,
) *Generator {
	return &Generator{
		signerCert: signerCert,
		signerKey:  signerKey,
		clock:      clock,
		randReader: randReader,
	}
}

// NewGeneratorFromFile creates a generator and loads CA cert and key from file.
func NewGeneratorFromFile(
	signerCertPath string,
	signerKeyPath string,
	clock Clock,
	randReader io.Reader,
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

	parsedSignerKey, err := x509.ParsePKCS8PrivateKey(signerKeyPem.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse CA key")
	}

	signerKey, ok := parsedSignerKey.(crypto.Signer)
	if !ok {
		return nil, errors.Wrap(err, "invalid CA key")
	}

	return NewGenerator(signerCert, signerKey, clock, randReader), nil
}

// CertificateRequest contains a minimal set of information to generate a self-signed certificate.
type CertificateRequest struct {
	CommonName       string
	AlternativeNames []string
}

// Generate generates a self-signed cert-key pair.
func (g *Generator) Generate(request CertificateRequest) ([]byte, []byte, error) {
	notBefore := g.clock.Now()
	notAfter := notBefore.Add(validity)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(g.randReader, serialNumberLimit)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate a serial number for the certificate")
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, privateKeyBits)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate private key")
	}

	privateKeyBytes, err := keyToBytes(privateKey)
	if err != nil {
		return nil, nil, err
	}

	certTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: request.CommonName,
		},
		DNSNames:              request.AlternativeNames,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	cert, err := x509.CreateCertificate(g.randReader, certTemplate, g.signerCert, privateKey.PublicKey, g.signerKey)

	return cert, privateKeyBytes, emperror.Wrap(err, "failed to sign certificate")
}

func keyToBytes(key *rsa.PrivateKey) ([]byte, error) {
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	buffer := bytes.NewBuffer(nil)

	if err := pem.Encode(buffer, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, errors.Wrap(err, "failed to pem-encode key")
	}

	return buffer.Bytes(), nil
}
