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

package spotguide

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	type (
		arr    = []interface{}
		obj    = map[string]interface{}
		number = float64
	)

	var null interface{} = nil

	testCases := map[string]struct {
		Dst interface{}
		Src interface{}
		Res interface{}
		Err interface{}
	}{
		"null dst, null src": {
			Dst: null,
			Src: null,
			Res: null,
		},
		"null dst, array src": {
			Dst: null,
			Src: arr{},
			Res: arr{},
		},
		"null dst, boolean src": {
			Dst: null,
			Src: true,
			Res: true,
		},
		"null dst, number src": {
			Dst: null,
			Src: number(42),
			Res: number(42),
		},
		"null dst, object src": {
			Dst: null,
			Src: obj{},
			Res: obj{},
		},
		"shorter array dst, longer array src": {
			Dst: arr{
				false,
				false,
				false,
			},
			Src: arr{
				true,
				true,
				true,
				true,
			},
			Res: arr{
				true,
				true,
				true,
				true,
			},
		},
		"longer array dst, shorter array src": {
			Dst: arr{
				false,
				false,
				false,
				false,
			},
			Src: arr{
				true,
				true,
				true,
			},
			Res: arr{
				true,
				true,
				true,
				false,
			},
		},
		"object dst, object src": {
			Dst: obj{
				"foo": false,
				"bar": false,
			},
			Src: obj{
				"bar": true,
				"buz": true,
			},
			Res: obj{
				"foo": false,
				"bar": true,
				"buz": true,
			},
		},
		"nil object dst, object src": {
			Dst: (obj)(nil),
			Src: obj{
				"foo": true,
			},
			Res: obj{
				"foo": true,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			res, err := merge(testCase.Dst, testCase.Src)
			switch testCase.Err {
			case nil, false:
				require.NoError(t, err)
				assert.Equal(t, testCase.Res, res)
			case true:
				require.Error(t, err)
			default:
				require.Equal(t, testCase.Err, err)
			}
		})
	}
}
