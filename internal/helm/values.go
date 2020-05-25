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

package helm

import (
	"emperror.dev/errors"
	ghodss "github.com/ghodss/yaml"
	"sigs.k8s.io/yaml"
)

func ConvertStructure(in interface{}) (map[string]interface{}, error) {
	valuesOverride, err := ghodss.Marshal(in)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to marshal values")
	}

	// convert back to map[string]interface{}
	var mapStringValues map[string]interface{}
	err = yaml.UnmarshalStrict(valuesOverride, &mapStringValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal values")
	}
	return mapStringValues, nil
}

func ConvertBytes(b []byte) (map[string]interface{}, error) {
	// convert back to map[string]interface{}
	var mapStringValues map[string]interface{}
	err := yaml.UnmarshalStrict(b, &mapStringValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal values to map[string]interface{}")
	}
	return mapStringValues, nil
}
