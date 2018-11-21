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
	"crypto/x509/pkix"
	"time"

	"github.com/banzaicloud/bank-vaults/pkg/tls"
	"github.com/pkg/errors"
)

const (
	validity = 365 * 24 * time.Hour
)

// Generator generates a self-signed cert-key pair.
type Generator struct {
	caLoader CALoader
}

// NewGenerator returns a new Generator instance.
func NewGenerator(caLoader CALoader) *Generator {
	return &Generator{
		caLoader: caLoader,
	}
}

// CertificateRequest contains a minimal set of information to generate a self-signed certificate.
type CertificateRequest struct {
	CommonName       string
	AlternativeNames []string
}

// Generate generates a self-signed cert-key pair.
func (g *Generator) Generate(request CertificateRequest) ([]byte, []byte, error) {
	rootCA, signingKey, err := g.caLoader.Load()
	if err != nil {
		return nil, nil, err
	}

	certRequest := tls.ServerCertificateRequest{
		Subject: pkix.Name{
			CommonName: request.CommonName,
		},
		DNSNames: request.AlternativeNames,
		Validity: validity,
	}
	cert, err := tls.GenerateServerCertificate(certRequest, rootCA, signingKey)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "failed to generate server certificate")
	}

	return cert.Certificate, cert.Key, nil
}
