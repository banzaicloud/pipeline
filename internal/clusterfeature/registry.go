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

package clusterfeature

import (
	"emperror.dev/errors"
)

// MakeFeatureManagerRegistry returns a FeatureManagerRegistry with the specified feature managers registered.
func MakeFeatureManagerRegistry(managers []FeatureManager) FeatureManagerRegistry {
	lookup := make(map[string]FeatureManager, len(managers))
	for _, fm := range managers {
		lookup[fm.Name()] = fm
	}

	return featureManagerRegistry{
		lookup: lookup,
	}
}

type featureManagerRegistry struct {
	lookup map[string]FeatureManager
}

func (r featureManagerRegistry) GetFeatureManager(featureName string) (FeatureManager, error) {

	if featureManager, ok := r.lookup[featureName]; ok {
		return featureManager, nil
	}

	return nil, errors.WithStack(UnknownFeatureError{FeatureName: featureName})
}

// MakeFeatureOperatorRegistry returns a FeatureOperatorRegistry with the specified feature operators registered.
func MakeFeatureOperatorRegistry(operators []FeatureOperator) FeatureOperatorRegistry {
	lookup := make(map[string]FeatureOperator, len(operators))
	for _, fo := range operators {
		lookup[fo.Name()] = fo
	}

	return featureOperatorRegistry{
		lookup: lookup,
	}
}

type featureOperatorRegistry struct {
	lookup map[string]FeatureOperator
}

func (r featureOperatorRegistry) GetFeatureOperator(featureName string) (FeatureOperator, error) {

	if featureOperator, ok := r.lookup[featureName]; ok {
		return featureOperator, nil
	}

	return nil, errors.WithStack(UnknownFeatureError{FeatureName: featureName})
}

// UnknownFeatureError is returned when there is no feature manager registered for a feature.
type UnknownFeatureError struct {
	FeatureName string
}

func (UnknownFeatureError) Error() string {
	return "unknown feature"
}

// Details returns the error's details
func (e UnknownFeatureError) Details() []interface{} {
	return []interface{}{"feature", e.FeatureName}
}
