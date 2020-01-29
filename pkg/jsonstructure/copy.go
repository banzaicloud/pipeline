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
	"fmt"

	"emperror.dev/errors"
)

// Copy returns a deep copy of a JSON structure value.
//
// A JSON structure value is one of:
// - nil
// - a bool
// - a number (float64)
// - a string
// - a []interface{}
// - a map[string]interface{}
// where the last two can only contain JSON structure values.
//
func Copy(v interface{}) (interface{}, error) {
	switch val := v.(type) {
	case nil, Boolean, Number, String:
		return val, nil
	case Array:
		return CopyArray(val)
	case Object:
		return CopyObject(val)
	default:
		return nil, errors.WithStackIf(unsupportedValueError{
			value: val,
		})
	}
}

// CopyArray returns a deep copy of a JSON array.
// Elements of the array must be valid JSON structure values.
func CopyArray(arr Array) (Array, error) {
	copy := make(Array, len(arr))
	for i, v := range arr {
		var err error
		copy[i], err = Copy(v)
		if err != nil {
			return nil, err
		}
	}
	return copy, nil
}

// CopyObject returns a deep copy of a JSON object.
// Members of the object must be valid JSON structure values
func CopyObject(obj Object) (Object, error) {
	copy := make(Object, len(obj))
	for k, v := range obj {
		var err error
		copy[k], err = Copy(v)
		if err != nil {
			return nil, err
		}
	}
	return copy, nil
}

type unsupportedValueError struct {
	value interface{}
}

func (e unsupportedValueError) Error() string {
	return fmt.Sprintf("not a JSON structure value: %v (%T)", e.value, e.value)
}
