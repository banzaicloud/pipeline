// Copyright © 2018 Banzai Cloud
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
	"reflect"

	"emperror.dev/errors"
)

func min(x, y int) int {
	if x < y {
		return x
	}

	return y
}

func merge(dst, src interface{}) (interface{}, error) {
	if reflect.DeepEqual(dst, src) {
		return dst, nil
	}

	type (
		array   = []interface{}
		boolean = bool
		number  = float64
		object  = map[string]interface{}
	)

	switch dstV := dst.(type) {
	case nil: // null
		return src, nil

	case boolean, number, string:
		if src == nil {
			return dst, nil
		}

		return src, nil

	case object:
		switch srcV := src.(type) {
		case nil: // null
			return dst, nil

		case array, boolean, number, string:
			return src, nil

		case object:
			if dstV == nil { // this is not the same as dst == nil
				return src, nil
			}

			for key := range srcV {
				val, err := merge(dstV[key], srcV[key])
				if err != nil {
					return dst, err
				}

				dstV[key] = val
			}

			return dstV, nil
		}

	case array:
		switch srcV := src.(type) {
		case nil: // null
			return dst, nil

		case boolean, number, object, string:
			return src, nil

		case array:
			// merge elements at common indices
			length := min(len(dstV), len(srcV))
			for i := 0; i < length; i++ {
				val, err := merge(dstV[i], srcV[i])
				if err != nil {
					return dst, err
				}

				dstV[i] = val
			}

			// append surplus from src (if any)
			dstV = append(dstV, srcV[length:]...)

			return dstV, nil
		}

	default:
		return nil, errors.NewWithDetails("unsupported dst value", "dst", dst)
	}

	return nil, errors.NewWithDetails("unsupported src value", "src", src)
}
