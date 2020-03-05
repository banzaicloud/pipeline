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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/secret"
)

func TestTLSType(t *testing.T) {
	assert.Implements(t, (*secret.Type)(nil), new(TLSType))
	assert.Implements(t, (*secret.GeneratorType)(nil), new(TLSType))
}

func TestTLSType_Validate(t *testing.T) {
	tests := []struct {
		name string
		data map[string]string

		message    string
		violations []string
	}{
		{
			name:    "Empty",
			message: "missing key: " + FieldTLSCACert,
			violations: []string{
				"missing key: " + FieldTLSCACert,
				"missing key: " + FieldTLSServerKey,
				"missing key: " + FieldTLSServerCert,
			},
		},
		{
			name: "Valid",
			data: map[string]string{
				FieldTLSCACert:     "",
				FieldTLSServerKey:  "",
				FieldTLSServerCert: "",
			},
		},
		{
			name: "MutualInvalid",
			data: map[string]string{
				FieldTLSCACert:     "",
				FieldTLSServerKey:  "",
				FieldTLSServerCert: "",
				"key":              "",
			},
			message: "missing key: " + FieldTLSClientKey,
			violations: []string{
				"missing key: " + FieldTLSClientKey,
				"missing key: " + FieldTLSClientCert,
			},
		},
		{
			name: "MutualValid",
			data: map[string]string{
				FieldTLSCACert:     "",
				FieldTLSServerKey:  "",
				FieldTLSServerCert: "",
				FieldTLSClientKey:  "",
				FieldTLSClientCert: "",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			typ := TLSType{}

			err := typ.Validate(test.data)

			if test.message != "" {
				assert.EqualError(t, err, test.message)
			}

			if len(test.violations) > 0 {
				var verr secret.ValidationError
				if !errors.As(err, &verr) {
					t.Fatal("error is expected to be a ValidationError")
				}

				assert.Equal(t, test.violations, verr.Violations())
			}
		})
	}
}

func TestTLSType_ValidateNew(t *testing.T) {
	tests := []struct {
		name string
		data map[string]string

		complete   bool
		message    string
		violations []string
	}{
		{
			name:    "Empty",
			message: "missing key: " + FieldTLSHosts,
			violations: []string{
				"missing key: " + FieldTLSHosts,
			},
		},
		{
			name: "Incomplete",
			data: map[string]string{
				FieldTLSHosts: "",
			},
		},
		{
			name: "Complete",
			data: map[string]string{
				FieldTLSCACert:     "ca",
				FieldTLSServerKey:  "",
				FieldTLSServerCert: "",
			},
			complete: true,
		},
		{
			name: "MutualInvalid",
			data: map[string]string{
				FieldTLSCACert:     "ca",
				FieldTLSServerKey:  "",
				FieldTLSServerCert: "",
				"key":              "",
			},
			complete: true,
			message:  "missing key: " + FieldTLSClientKey,
			violations: []string{
				"missing key: " + FieldTLSClientKey,
				"missing key: " + FieldTLSClientCert,
			},
		},
		{
			name: "MutualValid",
			data: map[string]string{
				FieldTLSCACert:     "ca",
				FieldTLSServerKey:  "",
				FieldTLSServerCert: "",
				FieldTLSClientKey:  "",
				FieldTLSClientCert: "",
			},
			complete: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			typ := TLSType{}

			complete, err := typ.ValidateNew(test.data)

			assert.Equal(t, test.complete, complete)

			if test.message != "" {
				assert.EqualError(t, err, test.message)
			}

			if len(test.violations) > 0 {
				var verr secret.ValidationError
				if !errors.As(err, &verr) {
					t.Fatal("error is expected to be a ValidationError")
				}

				assert.Equal(t, test.violations, verr.Violations())
			}
		})
	}
}

func TestTLSType_Generate(t *testing.T) {
	// TODO
}
