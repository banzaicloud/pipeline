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
)

func max(x, y int) int {
	if x > y {
		return x
	}

	return y
}

func elementOrNil(arr []interface{}, i int) interface{} {
	if i < len(arr) {
		return arr[i]
	}

	return nil
}

func getKeys(obj map[string]interface{}) []string {
	keys := []string{}
	for key := range obj {
		keys = append(keys, key)
	}

	return keys
}

func mergeKeys(a, b []string) []string {
	keys := append(a, b...)

	seen := make(map[string]bool)
	uniqKeys := []string{}
	for _, entry := range keys {
		if _, value := seen[entry]; !value {
			seen[entry] = true
			uniqKeys = append(uniqKeys, entry)
		}
	}

	return uniqKeys
}

func merge(dst, src interface{}) (interface{}, error) {
	if reflect.DeepEqual(dst, src) {
		return dst, nil
	}

	switch dstV := dst.(type) {
	// string
	case string:
		if src == nil {
			return dst, nil
		}

		return src, nil

	// number
	case float64:
		if src == nil {
			return dst, nil
		}

		return src, nil

	// boolean
	case bool:
		if src == nil {
			return dst, nil
		}

		return src, nil

	// null
	case nil:
		return src, nil

	// object
	case map[string]interface{}:
		if src == nil {
			return dst, nil
		}

		switch srcV := src.(type) {
		case map[string]interface{}:
			for _, key := range mergeKeys(getKeys(dstV), getKeys(srcV)) {
				val, err := merge(dstV[key], srcV[key])
				if err != nil {
					return dst, err
				}

				dstV[key] = val
			}

			return dstV, nil

		default:
			return src, nil
		}

	// array
	case []interface{}:
		if src == nil {
			return dst, nil
		}

		switch srcV := src.(type) {
		case []interface{}:
			length := max(len(dstV), len(srcV))
			for i := 0; i < length; i++ {
				val, err := merge(elementOrNil(dstV, i), elementOrNil(srcV, i))
				if err != nil {
					return dst, err
				}

				dstV[i] = val
			}

			return dstV, nil
		default:
			return src, nil
		}

	default:
		if src == nil {
			return dst, nil
		}

		return src, nil
	}
}
