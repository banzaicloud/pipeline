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

package brn

import (
	"fmt"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nolint: gochecknoglobals
var tests = []struct {
	brn      string
	valid    bool
	expected ResourceName
}{
	{
		"",
		false,
		ResourceName{},
	},
	{
		"::",
		false,
		ResourceName{},
	},
	{
		":::",
		true,
		ResourceName{},
	},
	{
		"brn:::",
		true,
		ResourceName{
			Scheme: "brn",
		},
	},
	{
		"brn:1:secret:dc460da4ad72c482231e28e688e01f2778a88ce31a08826899d54ef7183998b5",
		true,
		ResourceName{
			Scheme:         "brn",
			OrganizationID: uint(1),
			ResourceType:   "secret",
			ResourceID:     "dc460da4ad72c482231e28e688e01f2778a88ce31a08826899d54ef7183998b5",
		},
	},
	{
		"brn::cluster:f00bb631-0e1e-4433-a12a-2f4070b48f50",
		true,
		ResourceName{
			Scheme:       "brn",
			ResourceType: "cluster",
			ResourceID:   "f00bb631-0e1e-4433-a12a-2f4070b48f50",
		},
	},
}

func TestParse(t *testing.T) {
	for _, test := range tests {
		test := test
		t.Run(test.brn, func(t *testing.T) {
			rn, err := Parse(test.brn)

			if test.valid {
				require.NoError(t, err)
				assert.Equal(t, test.expected, rn)
			} else {
				require.Error(t, err)
				assert.Equal(t, errors.Cause(err), ErrInvalid)
			}
		})
	}
}

func TestParseAs(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		expected := ResourceName{
			Scheme:         "brn",
			OrganizationID: uint(1),
			ResourceType:   "secret",
			ResourceID:     "dc460da4ad72c482231e28e688e01f2778a88ce31a08826899d54ef7183998b5",
		}

		rn, err := ParseAs("brn:1:secret:dc460da4ad72c482231e28e688e01f2778a88ce31a08826899d54ef7183998b5", SecretResourceType)

		require.NoError(t, err)
		assert.Equal(t, expected, rn)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := ParseAs("brn:1:notSecret:dc460da4ad72c482231e28e688e01f2778a88ce31a08826899d54ef7183998b5", SecretResourceType)

		require.Error(t, err)
		assert.Equal(t, ErrUnexpectedResourceType, errors.Cause(err))
	})
}

func TestResourceName_String(t *testing.T) {
	for _, test := range tests {
		if test.valid == false {
			continue
		}

		t.Run(test.brn, func(t *testing.T) {
			assert.Equal(t, test.brn, test.expected.String())
		})
	}
}

func ExampleParse() {
	resourceName, err := Parse("brn:1:secret:dc460da4ad72c482231e28e688e01f2778a88ce31a08826899d54ef7183998b5")
	if err != nil {
		panic(err)
	}

	fmt.Println(resourceName.String())

	// Output:
	// brn:1:secret:dc460da4ad72c482231e28e688e01f2778a88ce31a08826899d54ef7183998b5
}
