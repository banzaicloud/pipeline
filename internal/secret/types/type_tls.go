// Copyright © 2020 Banzai Cloud
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

package types

import (
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/tls"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/secret"
)

const TLS = "tls"

const (
	FieldTLSHosts      = "hosts"
	FieldTLSValidity   = "validity"
	FieldTLSCACert     = "caCert"
	FieldTLSCAKey      = "caKey"
	FieldTLSServerKey  = "serverKey"
	FieldTLSServerCert = "serverCert"
	FieldTLSClientKey  = "clientKey"
	FieldTLSClientCert = "clientCert"
	FieldTLSPeerKey    = "peerKey"
	FieldTLSPeerCert   = "peerCert"
)

type TLSType struct {
	DefaultValidity time.Duration
}

func (TLSType) Name() string {
	return TLS
}

func (TLSType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldTLSHosts, Required: true, IsSafeToDisplay: true},
			{Name: FieldTLSValidity, Required: false, IsSafeToDisplay: true},
			{Name: FieldTLSCACert, Required: false},
			{Name: FieldTLSCAKey, Required: false},
			{Name: FieldTLSServerKey, Required: false},
			{Name: FieldTLSServerCert, Required: false},
			{Name: FieldTLSClientKey, Required: false},
			{Name: FieldTLSClientCert, Required: false},
			{Name: FieldTLSPeerKey, Required: false},
			{Name: FieldTLSPeerCert, Required: false},
		},
	}
}

// Note: this will only require the TLS host field.
func (t TLSType) Validate(data map[string]string) error {
	var violations []string

	// Server TLS is the default
	// TODO: CA-only, client only?
	for _, field := range []string{FieldTLSCACert, FieldTLSServerKey, FieldTLSServerCert} {
		if _, ok := data[field]; !ok {
			violations = append(violations, fmt.Sprintf("missing key: %s", field))
		}
	}

	// We expect keys for mutual TLS
	if len(data) > 3 {
		for _, field := range []string{FieldTLSClientKey, FieldTLSClientCert} {
			if _, ok := data[field]; !ok {
				violations = append(violations, fmt.Sprintf("missing key: %s", field))
			}
		}
	}

	if len(violations) > 0 {
		// For backward compatibility reasons, return the first violation as message
		return secret.NewValidationError(violations[0], violations)
	}

	return nil
}

// TODO: this should determine incompleteness more reliably.
func (t TLSType) ValidateNew(data map[string]string) (bool, error) {
	complete := false

	for k, v := range data {
		if k != FieldTLSHosts && k != FieldTLSValidity && v != "" {
			complete = true

			break
		}
	}

	if !complete {
		if _, ok := data[FieldTLSHosts]; !ok {
			msg := fmt.Sprintf("missing key: %s", FieldTLSHosts)

			return false, secret.NewValidationError(msg, []string{msg})
		}

		return false, nil
	}

	return true, t.Validate(data)
}

func (t TLSType) Generate(_ uint, _ string, data map[string]string, _ []string) (map[string]string, error) {
	validity := data[FieldTLSValidity]
	if validity == "" {
		validity = t.DefaultValidity.String()
	}

	cc, err := tls.GenerateTLS(data[FieldTLSHosts], validity)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to generate TLS secret")
	}

	err = mapstructure.Decode(cc, &data)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode TLS secret")
	}

	return data, nil
}
