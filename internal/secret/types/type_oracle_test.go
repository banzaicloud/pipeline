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

func TestOracleType(t *testing.T) {
	assert.Implements(t, (*secret.Type)(nil), new(OracleType))
	assert.Implements(t, (*secret.VerifierType)(nil), new(OracleType))
}

func TestOracleType_Validate(t *testing.T) {
	tests := []struct {
		name string
		data map[string]string

		message    string
		violations []string
	}{
		{
			name:    "Empty",
			message: "missing key: " + FieldOracleUserOCID,
			violations: []string{
				"missing key: " + FieldOracleUserOCID,
				"missing key: " + FieldOracleTenancyOCID,
				"missing key: " + FieldOracleAPIKey,
				"missing key: " + FieldOracleAPIKeyFingerprint,
				"missing key: " + FieldOracleRegion,
				"missing key: " + FieldOracleCompartmentOCID,
			},
		},
		{
			name: "Valid",
			data: map[string]string{
				FieldOracleUserOCID:          "",
				FieldOracleTenancyOCID:       "",
				FieldOracleAPIKey:            "",
				FieldOracleAPIKeyFingerprint: "",
				FieldOracleRegion:            "",
				FieldOracleCompartmentOCID:   "",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			typ := OracleType{}

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
