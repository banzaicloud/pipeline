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

package jsonstructure

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	testCases := map[string]struct {
		Value interface{}
		Error interface{}
	}{
		"nil": {
			Value: nil,
		},
		"false": {
			Value: false,
		},
		"true": {
			Value: true,
		},
		"zero": {
			Value: Number(0),
		},
		"number": {
			Value: Number(42),
		},
		"empty string": {
			Value: "",
		},
		"string": {
			Value: "lorem ipsum",
		},
		"empty array": {
			Value: Array{},
		},
		"array": {
			Value: Array{
				nil,
				false,
				true,
				Number(0),
				Number(42),
				"",
				"lorem ipsum",
				Array{},
				Object{},
			},
		},
		"empty object": {
			Value: Object{},
		},
		"object": {
			Value: Object{
				"nil":          nil,
				"false":        false,
				"true":         true,
				"0":            Number(0),
				"42":           Number(42),
				"empty string": "",
				"string":       "lorem ipsum",
				"array":        Array{},
				"object":       Object{},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			copy, err := Copy(testCase.Value)

			switch testCase.Error {
			case nil, false:
				require.NoError(t, err)
			case true:
				require.Error(t, err)
			default:
				require.Equal(t, testCase.Error, err)
			}

			require.Equal(t, testCase.Value, copy)
		})
	}
}
