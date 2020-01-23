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
			key:    "key",
			errors: nil,
		},
		{
			key: "key*",
			errors: []string{
				"invalid label key \"key*\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
			},
		},
		{
			key: "",
			errors: []string{
				"invalid label key \"\": name part must be non-empty", "invalid label key \"\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
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
			if test.errors == nil {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)

			var verr LabelValidationError

			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, test.errors, verr.Violations())
		})
	}
}

func TestLabelValidator_ValidateValue(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		validator := LabelValidator{}

		err := validator.ValidateValue("value")
		require.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
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
	})
}

func TestLabelValidator_ValidateLabel(t *testing.T) {
	tests := []struct {
		key    string
		value  string
		errors []string
	}{
		{
			key:    "key",
			value:  "",
			errors: nil,
		},
		{
			key:   "key*",
			value: "value.*-/",
			errors: []string{
				"invalid label key \"key*\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
				"invalid label value \"value.*-/\": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')",
			},
		},
		{
			key:   "",
			value: "",
			errors: []string{
				"invalid label key \"\": name part must be non-empty", "invalid label key \"\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
			},
		},
		{
			key:   "example.com/key",
			value: "",
			errors: []string{
				"forbidden label key domain in \"example.com/key\": \"example.com\" domain is not allowed",
			},
		},
		{
			key:   "node.example.com/key",
			value: "",
			errors: []string{
				"forbidden label key domain in \"node.example.com/key\": \"example.com\" domain is not allowed",
			},
		},
		{
			key:   "node-role.kubernetes.io/master",
			value: "",
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
			err := validator.ValidateLabel(test.key, test.value)
			if test.errors == nil {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)

			var verr LabelValidationError

			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, test.errors, verr.Violations())
		})
	}
}

func TestLabelValidator_ValidateLabels(t *testing.T) {
	labels := map[string]string{
		"key":                            "",
		"key*":                           "value.*-/",
		"":                               "",
		"example.com/key":                "",
		"node.example.com/key":           "",
		"node-role.kubernetes.io/master": "",
	}

	expectedViolations := []string{
		"invalid label key \"key*\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
		"invalid label value \"value.*-/\": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')",
		"invalid label key \"\": name part must be non-empty", "invalid label key \"\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
		"forbidden label key domain in \"example.com/key\": \"example.com\" domain is not allowed",
		"forbidden label key domain in \"node.example.com/key\": \"example.com\" domain is not allowed",
		"label key \"node-role.kubernetes.io/master\" is not allowed",
	}

	validator := LabelValidator{
		ForbiddenDomains: []string{"example.com"},
	}

	err := validator.ValidateLabels(labels)
	require.Error(t, err)

	var verr LabelValidationError

	assert.True(t, errors.As(err, &verr))

	violations := verr.Violations()

	for _, violation := range expectedViolations {
		assert.Contains(t, violations, violation)
	}
}
