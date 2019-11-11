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
	testCases := map[string]struct {
		Dst interface{}
		Src interface{}
		Res interface{}
		Err interface{}
	}{
		"shorter dst array, longer src array": {
			Dst: []interface{}{
				1, 2, 3,
			},
			Src: []interface{}{
				4, 5, 6, 7,
			},
			Res: []interface{}{
				4, 5, 6, 7,
			},
		},
		"longer dst array, shorter src array": {
			Dst: []interface{}{
				1, 2, 3, 4,
			},
			Src: []interface{}{
				5, 6, 7,
			},
			Res: []interface{}{
				5, 6, 7, 4,
			},
		},
		"object dst, object src": {
			Dst: map[string]interface{}{
				"foo": 1,
				"bar": 2,
			},
			Src: map[string]interface{}{
				"bar": 3,
				"buz": 4,
			},
			Res: map[string]interface{}{
				"foo": 1,
				"bar": 3,
				"buz": 4,
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
