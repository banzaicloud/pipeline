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

package values

import (
	"reflect"
	"strings"

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	"github.com/mitchellh/mapstructure"
)

func StringToMapStringInterface() mapstructure.DecodeHookFunc {
	return func(srcType, dstType reflect.Type, srcVal interface{}) (interface{}, error) {
		var output map[string]interface{}
		if srcType.Kind() == reflect.String && dstType == reflect.TypeOf(output) {
			if str, ok := srcVal.(string); ok {
				if err := yaml.Unmarshal([]byte(strings.TrimSpace(str)), &output); err != nil {
					return nil, errors.WrapIf(err, "failed to convert string to map")
				}

				return output, nil
			}

		}

		return srcVal, nil
	}
}
