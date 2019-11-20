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

package jsonstructure

import (
	"encoding/json"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	type (
		NamedBool bool

		NamedInterface interface {
			Method()
		}

		NamedStruct struct {
			Field bool
		}
	)

	var (
		aBool      bool
		aNamedBool NamedBool
	)

	testCases := map[string]struct {
		Input  interface{}
		Result interface{}
		Error  interface{}
	}{
		"nil": {
			Input:  nil,
			Result: nil,
		},

		// simple types

		"false": {
			Input:  false,
			Result: false,
		},
		"true": {
			Input:  true,
			Result: true,
		},
		"float32": {
			Input:  float32(42),
			Result: float64(42),
		},
		"float64": {
			Input:  float64(42),
			Result: float64(42),
		},
		"int": {
			Input:  int(42),
			Result: float64(42),
		},
		"int8": {
			Input:  int8(42),
			Result: float64(42),
		},
		"int16": {
			Input:  int16(42),
			Result: float64(42),
		},
		"int32": {
			Input:  int32(42),
			Result: float64(42),
		},
		"int64": {
			Input:  int64(42),
			Result: float64(42),
		},
		"uint": {
			Input:  uint(42),
			Result: float64(42),
		},
		"uint8": {
			Input:  uint8(42),
			Result: float64(42),
		},
		"uint16": {
			Input:  uint16(42),
			Result: float64(42),
		},
		"uint32": {
			Input:  uint32(42),
			Result: float64(42),
		},
		"uint64": {
			Input:  uint64(42),
			Result: float64(42),
		},
		"uint64 max": {
			Input:  ^uint64(0),
			Result: float64(^uint64(0)),
		},
		"uintptr": {
			Input:  uintptr(42),
			Result: float64(42),
		},
		"unsafe.Pointer": {
			Input: unsafe.Pointer(nil),
			Error: true,
		},
		"complex64": {
			Input: complex64(0 + 0i),
			Error: true,
		},
		"complex128": {
			Input: complex128(0 + 0i),
			Error: true,
		},
		"string": {
			Input:  "42",
			Result: "42",
		},
		"byte": {
			Input:  byte(42),
			Result: float64(42),
		},
		"rune": {
			Input:  rune(42),
			Result: float64(42),
		},

		// composite types

		// pointers
		"nil bool pointer": {
			Input:  (*bool)(nil),
			Result: nil,
		},
		"non-nil bool pointer": {
			Input:  &aBool,
			Result: aBool,
		},
		"nil named bool pointer": {
			Input:  (*NamedBool)(nil),
			Result: nil,
		},
		"non-nil named bool pointer": {
			Input:  &aNamedBool,
			Result: bool(aNamedBool),
		},

		// arrays
		"empty bool array": {
			Input:  [...]bool{},
			Result: []interface{}{},
		},
		"non-empty bool array": {
			Input:  [...]bool{false, true},
			Result: []interface{}{false, true},
		},
		"empty float32 array": {
			Input:  [...]float32{},
			Result: []interface{}{},
		},
		"non-empty float32 array": {
			Input:  [...]float32{42},
			Result: []interface{}{float64(42)},
		},
		"empty float64 array": {
			Input:  [...]float64{},
			Result: []interface{}{},
		},
		"non-empty float64 array": {
			Input:  [...]float64{42},
			Result: []interface{}{float64(42)},
		},
		"empty byte array": {
			Input:  [...]byte{},
			Result: []interface{}{},
		},
		"non-empty byte array": {
			Input:  [...]byte{42},
			Result: []interface{}{float64(42)},
		},
		"empty interface{} array": {
			Input:  [...]interface{}{},
			Result: []interface{}{},
		},
		"non-empy interface{} array": {
			Input: [...]interface{}{
				nil,
				false, true,
				float32(42), float64(42), int(42), uint(42),
				[...]bool{false, true},
				[]bool{false, true},
				[...]interface{}{},
				[]interface{}{},
				map[string]bool{
					"false": false,
					"true":  true,
				},
				map[string]interface{}{},
			},
			Result: []interface{}{
				nil,
				false, true,
				float64(42), float64(42), float64(42), float64(42),
				[]interface{}{false, true},
				[]interface{}{false, true},
				[]interface{}{},
				[]interface{}{},
				map[string]interface{}{
					"false": false,
					"true":  true,
				},
				map[string]interface{}{},
			},
		},

		// slices
		"nil bool slice": {
			Input:  []bool(nil),
			Result: nil,
		},
		"empty bool slice": {
			Input:  []bool{},
			Result: []interface{}{},
		},
		"non-empty bool slice": {
			Input:  []bool{false, true},
			Result: []interface{}{false, true},
		},
		"nil byte slice": {
			Input:  []byte(nil),
			Result: nil,
		},
		"empty byte slice": {
			Input:  []byte{},
			Result: "",
		},
		"non-empty byte slice": {
			Input:  []byte{1, 2, 3},
			Result: "AQID",
		},
		"nil interface{} slice": {
			Input:  []interface{}(nil),
			Result: nil,
		},
		"empty interface{} slice": {
			Input:  []interface{}{},
			Result: []interface{}{},
		},

		// channels
		"nil chan": {
			Input: (chan bool)(nil),
			Error: true,
		},
		"non-nil chan": {
			Input: make(chan bool),
			Error: true,
		},

		// functions
		"nil func": {
			Input: (func())(nil),
			Error: true,
		},
		"non-nil func": {
			Input: func() {},
			Error: true,
		},

		// maps
		"nil map[bool]bool": {
			Input: (map[bool]bool)(nil),
			Error: true,
		},
		"empty map[bool]bool": {
			Input: map[bool]bool{},
			Error: true,
		},
		"non-empty map[bool]bool": {
			Input: map[bool]bool{false: true, true: false},
			Error: true,
		},
		"nil map[string]bool": {
			Input:  map[string]bool(nil),
			Result: nil,
		},
		"empty map[string]bool": {
			Input:  map[string]bool{},
			Result: map[string]interface{}{},
		},
		"non-empty map[string]bool": {
			Input: map[string]bool{
				"false": false,
				"true":  true,
			},
			Result: map[string]interface{}{
				"false": false,
				"true":  true,
			},
		},
		"map[string]interface{}": {
			Input: map[string]interface{}{
				"nil":     nil,
				"false":   false,
				"true":    true,
				"float32": float32(42),
				"float64": float64(42),
				"int":     int(42),
				"uint":    uint(42),
				"array":   [...]interface{}{},
				"slice":   []interface{}{},
			},
			Result: map[string]interface{}{
				"nil":     nil,
				"false":   false,
				"true":    true,
				"float32": float64(42),
				"float64": float64(42),
				"int":     float64(42),
				"uint":    float64(42),
				"array":   []interface{}{},
				"slice":   []interface{}{},
			},
		},

		// structs
		"empty struct": {
			Input:  struct{}{},
			Result: map[string]interface{}{},
		},
		"struct with false bool field, no tag": {
			Input: struct {
				Field bool
			}{
				Field: false,
			},
			Result: map[string]interface{}{
				"Field": false,
			},
		},
		"struct with true bool field, no tag": {
			Input: struct {
				Field bool
			}{
				Field: true,
			},
			Result: map[string]interface{}{
				"Field": true,
			},
		},
		"struct with false bool field, omit tag": {
			Input: struct {
				Field bool `json:"-"`
			}{
				Field: false,
			},
			Result: map[string]interface{}{},
		},
		"struct with true bool field, omit tag": {
			Input: struct {
				Field bool `json:"-"`
			}{
				Field: true,
			},
			Result: map[string]interface{}{},
		},
		"struct with false bool field, empty tag": {
			Input: struct {
				Field bool `json:""`
			}{
				Field: false,
			},
			Result: map[string]interface{}{
				"Field": false,
			},
		},
		"struct with true bool field, empty tag": {
			Input: struct {
				Field bool `json:""`
			}{
				Field: true,
			},
			Result: map[string]interface{}{
				"Field": true,
			},
		},
		"struct with false bool field, rename tag": {
			Input: struct {
				Field bool `json:"field"`
			}{
				Field: false,
			},
			Result: map[string]interface{}{
				"field": false,
			},
		},
		"struct with true bool field, rename tag": {
			Input: struct {
				Field bool `json:"field"`
			}{
				Field: true,
			},
			Result: map[string]interface{}{
				"field": true,
			},
		},
		"struct with false bool field, omitempty tag": {
			Input: struct {
				Field bool `json:",omitempty"`
			}{
				Field: false,
			},
			Result: map[string]interface{}{},
		},
		"struct with true bool field, omitempty tag": {
			Input: struct {
				Field bool `json:",omitempty"`
			}{
				Field: true,
			},
			Result: map[string]interface{}{
				"Field": true,
			},
		},
		"struct with false bool field, rename and omitempty tag": {
			Input: struct {
				Field bool `json:"field,omitempty"`
			}{
				Field: false,
			},
			Result: map[string]interface{}{},
		},
		"struct with true bool field, rename and omitempty tag": {
			Input: struct {
				Field bool `json:"field,omitempty"`
			}{
				Field: true,
			},
			Result: map[string]interface{}{
				"field": true,
			},
		},
		"struct with zero int field, no tag": {
			Input: struct {
				Field int
			}{
				Field: 0,
			},
			Result: map[string]interface{}{
				"Field": float64(0),
			},
		},
		"struct with non-zero int field, no tag": {
			Input: struct {
				Field int
			}{
				Field: 42,
			},
			Result: map[string]interface{}{
				"Field": float64(42),
			},
		},
		"struct with zero int field, omit tag": {
			Input: struct {
				Field int `json:"-"`
			}{
				Field: 0,
			},
			Result: map[string]interface{}{},
		},
		"struct with non-zero int field, omit tag": {
			Input: struct {
				Field int `json:"-"`
			}{
				Field: 42,
			},
			Result: map[string]interface{}{},
		},
		"struct with zero int field, empty tag": {
			Input: struct {
				Field int `json:""`
			}{
				Field: 0,
			},
			Result: map[string]interface{}{
				"Field": float64(0),
			},
		},
		"struct with non-zero int field, empty tag": {
			Input: struct {
				Field int `json:""`
			}{
				Field: 42,
			},
			Result: map[string]interface{}{
				"Field": float64(42),
			},
		},
		"struct with zero int field, rename tag": {
			Input: struct {
				Field int `json:"field"`
			}{
				Field: 0,
			},
			Result: map[string]interface{}{
				"field": float64(0),
			},
		},
		"struct with non-zero int field, rename tag": {
			Input: struct {
				Field int `json:"field"`
			}{
				Field: 42,
			},
			Result: map[string]interface{}{
				"field": float64(42),
			},
		},
		"struct with zero int field, omitempty tag": {
			Input: struct {
				Field int `json:",omitempty"`
			}{
				Field: 0,
			},
			Result: map[string]interface{}{},
		},
		"struct with non-zero int field, omitempty tag": {
			Input: struct {
				Field int `json:",omitempty"`
			}{
				Field: 42,
			},
			Result: map[string]interface{}{
				"Field": float64(42),
			},
		},
		"struct with zero int field, rename and omitempty tag": {
			Input: struct {
				Field int `json:"field,omitempty"`
			}{
				Field: 0,
			},
			Result: map[string]interface{}{},
		},
		"struct with non-zero int field, rename and omitempty tag": {
			Input: struct {
				Field int `json:"field,omitempty"`
			}{
				Field: 42,
			},
			Result: map[string]interface{}{
				"field": float64(42),
			},
		},
		"struct with nil embedded interface field, no tag": {
			Input: struct {
				NamedInterface
			}{
				NamedInterface: nil,
			},
			Result: map[string]interface{}{
				"NamedInterface": nil,
			},
		},
		"struct with nil embedded interface field, omit tag": {
			Input: struct {
				NamedInterface `json:"-"`
			}{
				NamedInterface: nil,
			},
			Result: map[string]interface{}{},
		},
		"struct with nil embedded interface field, empty tag": {
			Input: struct {
				NamedInterface `json:""`
			}{
				NamedInterface: nil,
			},
			Result: map[string]interface{}{
				"NamedInterface": nil,
			},
		},
		"struct with nil embedded interface field, rename tag": {
			Input: struct {
				NamedInterface `json:"interface"`
			}{
				NamedInterface: nil,
			},
			Result: map[string]interface{}{
				"interface": nil,
			},
		},
		"struct with nil embedded interface field, omitempty tag": {
			Input: struct {
				NamedInterface `json:",omitempty"`
			}{
				NamedInterface: nil,
			},
			Result: map[string]interface{}{},
		},
		"struct with nil embedded interface field, rename and omitempty tag": {
			Input: struct {
				NamedInterface `json:"interface,omitempty"`
			}{
				NamedInterface: nil,
			},
			Result: map[string]interface{}{},
		},
		"struct with false named-bool-implemented embedded interface field, no tag": {
			Input: struct {
				NamedInterface
			}{
				NamedInterface: simpleInterfaceImplementer(false),
			},
			Result: map[string]interface{}{
				"NamedInterface": false,
			},
		},
		"struct with false named-bool-implemented embedded interface field, omit tag": {
			Input: struct {
				NamedInterface `json:"-"`
			}{
				NamedInterface: simpleInterfaceImplementer(false),
			},
			Result: map[string]interface{}{},
		},
		"struct with false named-bool-implemented embedded interface field, empty tag": {
			Input: struct {
				NamedInterface `json:""`
			}{
				NamedInterface: simpleInterfaceImplementer(false),
			},
			Result: map[string]interface{}{
				"NamedInterface": false,
			},
		},
		"struct with false named-bool-implemented embedded interface field, rename tag": {
			Input: struct {
				NamedInterface `json:"interface"`
			}{
				NamedInterface: simpleInterfaceImplementer(false),
			},
			Result: map[string]interface{}{
				"interface": false,
			},
		},
		"struct with false named-bool-implemented embedded interface field, omitempty tag": {
			Input: struct {
				NamedInterface `json:",omitempty"`
			}{
				NamedInterface: simpleInterfaceImplementer(false),
			},
			Result: map[string]interface{}{
				"NamedInterface": false,
			},
		},
		"struct with false named-bool-implemented embedded interface field, rename and omitempty tag": {
			Input: struct {
				NamedInterface `json:"interface,omitempty"`
			}{
				NamedInterface: simpleInterfaceImplementer(false),
			},
			Result: map[string]interface{}{
				"interface": false,
			},
		},
		"struct with struct-implemented embedded interface field, no tag": {
			Input: struct {
				NamedInterface
			}{
				NamedInterface: structInterfaceImplementer{},
			},
			Result: map[string]interface{}{
				"NamedInterface": map[string]interface{}{
					"Field": false,
				},
			},
		},
		"struct with nil pointer-implemented embedded interface field, no tag": {
			Input: struct {
				NamedInterface
			}{
				NamedInterface: (*pointerInterfaceImplementer)(nil),
			},
			Result: map[string]interface{}{
				"NamedInterface": nil,
			},
		},
		"struct with non-nil pointer-implemented embedded interface field, no tag": {
			Input: struct {
				NamedInterface
			}{
				NamedInterface: &pointerInterfaceImplementer{},
			},
			Result: map[string]interface{}{
				"NamedInterface": map[string]interface{}{
					"Field": false,
				},
			},
		},
		"struct with embedded struct, no tag": {
			Input: struct {
				NamedStruct
			}{
				NamedStruct: NamedStruct{},
			},
			Result: map[string]interface{}{
				"Field": false,
			},
		},
		"struct with embedded struct, omit tag": {
			Input: struct {
				NamedStruct `json:"-"`
			}{
				NamedStruct: NamedStruct{},
			},
			Result: map[string]interface{}{},
		},
		"struct with embedded struct, rename tag": {
			Input: struct {
				NamedStruct `json:"struct"`
			}{
				NamedStruct: NamedStruct{},
			},
			Result: map[string]interface{}{
				"struct": map[string]interface{}{
					"Field": false,
				},
			},
		},
		"struct with embedded struct, omitempty tag": {
			Input: struct {
				NamedStruct `json:",omitempty"`
			}{
				NamedStruct: NamedStruct{},
			},
			Result: map[string]interface{}{
				"Field": false,
			},
		},
		"struct with embedded struct, rename and omitempty tag": {
			Input: struct {
				NamedStruct `json:"struct,omitempty"`
			}{
				NamedStruct: NamedStruct{},
			},
			Result: map[string]interface{}{
				"struct": map[string]interface{}{
					"Field": false,
				},
			},
		},
		"struct with nil embedded pointer to struct, no tag": {
			Input: struct {
				*NamedStruct
			}{
				NamedStruct: nil,
			},
			Result: map[string]interface{}{},
		},
		"struct with non-nil embedded pointer to struct, no tag": {
			Input: struct {
				*NamedStruct
			}{
				NamedStruct: &NamedStruct{},
			},
			Result: map[string]interface{}{
				"Field": false,
			},
		},
		"struct with nil embedded pointer to struct, omit tag": {
			Input: struct {
				*NamedStruct `json:"-"`
			}{
				NamedStruct: nil,
			},
			Result: map[string]interface{}{},
		},
		"struct with non-nil embedded pointer to struct, omit tag": {
			Input: struct {
				*NamedStruct `json:"-"`
			}{
				NamedStruct: &NamedStruct{},
			},
			Result: map[string]interface{}{},
		},
		"struct with nil embedded pointer to struct, empty tag": {
			Input: struct {
				*NamedStruct `json:""`
			}{
				NamedStruct: nil,
			},
			Result: map[string]interface{}{},
		},
		"struct with non-nil embedded pointer to struct, empty tag": {
			Input: struct {
				*NamedStruct `json:""`
			}{
				NamedStruct: &NamedStruct{},
			},
			Result: map[string]interface{}{
				"Field": false,
			},
		},
		"struct with nil embedded pointer to struct, rename tag": {
			Input: struct {
				*NamedStruct `json:"struct"`
			}{
				NamedStruct: nil,
			},
			Result: map[string]interface{}{
				"struct": nil,
			},
		},
		"struct with non-nil embedded pointer to struct, rename tag": {
			Input: struct {
				*NamedStruct `json:"struct"`
			}{
				NamedStruct: &NamedStruct{},
			},
			Result: map[string]interface{}{
				"struct": map[string]interface{}{
					"Field": false,
				},
			},
		},
		"struct with nil embedded pointer to struct, omitempty tag": {
			Input: struct {
				*NamedStruct `json:",omitempty"`
			}{
				NamedStruct: nil,
			},
			Result: map[string]interface{}{},
		},
		"struct with non-nil embedded pointer to struct, omitempty tag": {
			Input: struct {
				*NamedStruct `json:",omitempty"`
			}{
				NamedStruct: &NamedStruct{},
			},
			Result: map[string]interface{}{
				"Field": false,
			},
		},
		"struct with nil embedded pointer to struct, rename and omitempty tag": {
			Input: struct {
				*NamedStruct `json:"struct,omitempty"`
			}{
				NamedStruct: nil,
			},
			Result: map[string]interface{}{},
		},
		"struct with non-nil embedded pointer to struct, rename and omitempty tag": {
			Input: struct {
				*NamedStruct `json:"struct,omitempty"`
			}{
				NamedStruct: &NamedStruct{},
			},
			Result: map[string]interface{}{
				"struct": map[string]interface{}{
					"Field": false,
				},
			},
		},

		// user-defined types
		"named bool": {
			Input:  aNamedBool,
			Result: bool(aNamedBool),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			jsonRes, jsonErr := jsonMarshalUnmarshal(t, testCase.Input)

			// verify the test case
			switch testCase.Error {
			case nil, false:
				require.NoError(t, jsonErr, "bad test case: JSON returned with error but no error is expected")
				require.Equal(t, jsonRes, testCase.Result, "bad test case: expected and JSON result differ")
			default:
				require.Error(t, jsonErr, "bad test case: JSON returned with success but an error is expected")
			}

			res, err := Encode(testCase.Input)

			switch testCase.Error {
			case nil, false:
				require.NoError(t, err, "Encode expected to succeed but did not")
				assert.Equal(t, testCase.Result, res)
			case true:
				require.Error(t, err, "Encode expected to fail")
			default:
				require.Equal(t, testCase.Error, err, "Encode expected to fail with specific error")
			}
		})
	}
}

type structInterfaceImplementer struct {
	Field bool
}

func (structInterfaceImplementer) Method() {}

type pointerInterfaceImplementer struct {
	Field bool
}

func (*pointerInterfaceImplementer) Method() {}

type simpleInterfaceImplementer bool

func (simpleInterfaceImplementer) Method() {}

func jsonMarshalUnmarshal(t *testing.T, v interface{}) (res interface{}, err error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	t.Logf("json: %q", string(raw))

	err = json.Unmarshal(raw, &res)
	t.Logf("unmarshalled: %#v (%T)", res, res)
	return
}
