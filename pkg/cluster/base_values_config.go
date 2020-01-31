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

package cluster

import (
	"reflect"
	"strings"

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	"github.com/mitchellh/mapstructure"
)

type PostHookConfig struct {
	// ingress controller config
	Ingress IngressControllerConfig
}

type IngressControllerConfig struct {
	Enabled bool
	Values  ValuesConfig
}

type ValuesConfig map[string]interface{}

func ValuesDecodeHook() mapstructure.DecodeHookFunc {
	return func(a reflect.Type, b reflect.Type, d interface{}) (interface{}, error) {

		if a.Kind() == reflect.String && b == reflect.TypeOf(new(ValuesConfig)).Elem() {

			if data, ok := d.(string); ok {
				_, err := yaml.Marshal(data)
				if err != nil {
					return nil, errors.WrapIf(err, "error during marshal yaml")
				}

				output, err := toMap(data)
				if err != nil {
					return nil, errors.WrapIf(err, "failed to convert string to map")
				}

				return output, nil
			}

		}

		return d, nil
	}
}

func toMap(v string) (map[string]interface{}, error) {
	var out = make(map[string]interface{})
	var trimmedStr = strings.TrimSpace(v)
	err := yaml.Unmarshal([]byte(trimmedStr), &out)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal values")
	}

	return out, nil
}
