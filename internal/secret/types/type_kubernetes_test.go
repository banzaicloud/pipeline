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
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/secret"
)

func TestKubernetesType(t *testing.T) {
	assert.Implements(t, (*secret.Type)(nil), new(KubernetesType))
	assert.Implements(t, (*secret.ProcessorType)(nil), new(KubernetesType))
	assert.Implements(t, (*secret.VerifierType)(nil), new(KubernetesType))
}

func TestKubernetesType_Validate(t *testing.T) {
	tests := []struct {
		name string
		data map[string]string

		message    string
		violations []string
	}{
		{
			name:    "Empty",
			message: "missing key: " + FieldKubernetesConfig,
			violations: []string{
				"missing key: " + FieldKubernetesConfig,
			},
		},
		{
			name: "Valid",
			data: map[string]string{
				FieldKubernetesConfig: "",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			typ := KubernetesType{}

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

func TestKubernetesType_Process(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]string
		output map[string]string
	}{
		{
			name: "NonBase64",
			input: map[string]string{
				FieldKubernetesConfig: "config",
			},
			output: map[string]string{
				FieldKubernetesConfig: "Y29uZmln",
			},
		},
		{
			name: "Base64",
			input: map[string]string{
				FieldKubernetesConfig: "Y29uZmln",
			},
			output: map[string]string{
				FieldKubernetesConfig: "Y29uZmln",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			typ := KubernetesType{}

			output, err := typ.Process(test.input)
			require.NoError(t, err)

			assert.Equal(t, test.output, output)
		})
	}
}
