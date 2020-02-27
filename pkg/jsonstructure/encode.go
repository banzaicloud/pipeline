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
	"encoding/base64"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/banzaicloud/pipeline/pkg/mirror"
)

// Encode returns a transformation of v that is equivalent with JSON encoding and decoding v.
func Encode(v interface{}, options ...EncodeOption) (interface{}, error) {
	var e encoder
	EncodeOptions(options).apply(&e)
	return e.encode(reflect.ValueOf(v))
}

// EncodeOption represents an encoding option.
type EncodeOption interface {
	apply(*encoder)
}

// EncodeOptions is a list of encoding options.
type EncodeOptions []EncodeOption

func (opts EncodeOptions) apply(e *encoder) {
	for _, opt := range opts {
		opt.apply(e)
	}
}

type zeroStructIsEmptyOption bool

// WithZeroStructsAsEmpty is an encoding option that makes the encoder treat struct values equal to their default (zero) value as empty.
const WithZeroStructsAsEmpty zeroStructIsEmptyOption = true

func (o zeroStructIsEmptyOption) apply(e *encoder) {
	e.zeroStructIsEmpty = bool(o)
}

type encoder struct {
	zeroStructIsEmpty bool
}

func (e encoder) encode(v reflect.Value) (interface{}, error) {
	numberType := reflect.TypeOf(float64(0))

	switch v.Kind() {
	// null
	case reflect.Invalid:
		return nil, nil

	// boolean, string
	case reflect.Bool:
		return v.Bool(), nil
	case reflect.String:
		return v.String(), nil

	// number
	case reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Convert(numberType).Interface(), nil

	// array
	case reflect.Slice:
		if v.IsNil() {
			return nil, nil
		}
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return base64.StdEncoding.EncodeToString(v.Bytes()), nil
		}
		fallthrough
	case reflect.Array:
		arr := make(Array, v.Len())
		for i := range arr {
			val, err := e.encode(v.Index(i))
			if err != nil {
				return nil, err
			}
			arr[i] = val
		}
		return arr, nil

	// indirection
	case reflect.Interface, reflect.Ptr:
		return e.encode(deref(v))

	case reflect.Map:
		if v.Type().Key().Kind() == reflect.String {
			if v.IsNil() {
				return nil, nil
			}

			obj := make(Object, v.Len())
			for it := v.MapRange(); it.Next(); {
				key := it.Key().String()
				val, err := e.encode(it.Value())
				if err != nil {
					return nil, err
				}
				obj[key] = val
			}
			return obj, nil
		}

	case reflect.Struct:
		obj := make(Object, v.NumField())
		if err := e.encodeStruct(obj, v); err != nil {
			return nil, err
		}
		return obj, nil

	default:
	}
	return nil, encodeError{v}
}

func (e encoder) encodeStruct(obj Object, value reflect.Value) error {
	for it := mirror.NewStructIter(value); it.Next(); {
		if err := e.encodeField(obj, it.Field(), it.Value()); err != nil {
			return err
		}
	}
	return nil
}

func (e encoder) encodeField(obj Object, field reflect.StructField, value reflect.Value) error {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "-" {
		// omit field
		return nil
	}

	jsonTags := strings.Split(jsonTag, ",")

	var (
		omitempty bool
	)

	if len(jsonTags) > 1 {
		for _, t := range jsonTags[1:] {
			switch t {
			case "omitempty":
				omitempty = true
			}
		}
	}

	if omitempty && e.isEmpty(value) {
		// omit empty field
		return nil
	}

	var key string
	if name := jsonTags[0]; name != "" {
		key = name
	}

	if field.Anonymous && key == "" {
		switch value := derefPtr(value); value.Kind() {
		case reflect.Invalid:
			// omit nil pointer
			return nil
		case reflect.Struct:
			if err := e.encodeStruct(obj, value); err != nil {
				return err
			}
			return nil
		}
	}

	val, err := e.encode(value)
	if err != nil {
		return err
	}

	if key == "" {
		key = field.Name
	}

	obj[key] = val

	return nil
}

func (e encoder) isEmpty(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Bool:
		return value.Bool() == false
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return math.Float64bits(value.Float()) == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	case reflect.Struct:
		if e.zeroStructIsEmpty {
			return value.IsZero()
		}
		fallthrough
	default:
		return false
	}
}

func indirect(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Interface, reflect.Ptr:
		return true
	default:
		return false
	}
}

func deref(value reflect.Value) reflect.Value {
	if indirect(value) {
		return deref(value.Elem())
	}
	return value
}

func derefPtr(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Ptr {
		return value.Elem()
	}
	return value
}

type encodeError struct {
	value reflect.Value
}

func (e encodeError) Error() string {
	return fmt.Sprintf("cannot encode value: %#v", e.value)
}
