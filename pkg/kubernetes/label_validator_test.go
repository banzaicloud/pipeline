// Copyright © 2019 Banzai Cloud
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

package kubernetes

import (
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelValidator_ValidateKey(t *testing.T) {
	tests := []struct {
		key    string
		errors []string
	}{
		{
			key: "key*",
			errors: []string{
				"invalid label key \"key*\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
			},
		},
		{
			key: "example.com/key",
			errors: []string{
				"forbidden label key domain in \"example.com/key\": \"example.com\" domain is not allowed",
			},
		},
		{
			key: "node.example.com/key",
			errors: []string{
				"forbidden label key domain in \"node.example.com/key\": \"example.com\" domain is not allowed",
			},
		},
		{
			key: "node-role.kubernetes.io/master",
			errors: []string{
				"label key \"node-role.kubernetes.io/master\" is not allowed",
			},
		},
	}

	validator := LabelValidator{
		ForbiddenDomains: []string{"example.com"},
	}

	for _, test := range tests {
		test := test

		t.Run("", func(t *testing.T) {
			err := validator.ValidateKey(test.key)
			require.Error(t, err)

			var verr LabelValidationError

			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, test.errors, verr.Violations())
		})
	}
}

func TestLabelValidator_ValidateValue(t *testing.T) {
	validator := LabelValidator{}

	err := validator.ValidateValue("value.*-/")
	require.Error(t, err)

	var verr LabelValidationError

	assert.True(t, errors.As(err, &verr))
	assert.Equal(t,
		[]string{
			"invalid label value \"value.*-/\": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')",
		},
		verr.Violations(),
	)
}
