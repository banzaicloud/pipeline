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
	"encoding/pem"
	"time"

	"github.com/banzaicloud/bank-vaults/pkg/sdk/tls"
	"github.com/pkg/errors"
)

const (
	validity = 365 * 24 * time.Hour
)

// Generator creates a cert-key pair issued by the loaded CA.
type Generator struct {
	caLoader CALoader
}

// NewGenerator returns a new Generator instance.
func NewGenerator(caLoader CALoader) *Generator {
	return &Generator{
		caLoader: caLoader,
	}
}

// GenerateServerCertificate generates a cert-key pair for server usage issued by the loaded CA.
func (g *Generator) GenerateServerCertificate(req tls.ServerCertificateRequest) ([]byte, []byte, []byte, error) {
	rootCA, signingKey, err := g.caLoader.Load()
	if err != nil {
		return nil, nil, nil, err
	}

	req.Validity = validity

	cert, err := tls.GenerateServerCertificate(req, rootCA, signingKey)
	if err != nil {
		return nil, nil, nil, errors.WithMessage(err, "failed to generate server certificate")
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCA.Raw}), cert.Certificate, cert.Key, nil
}
