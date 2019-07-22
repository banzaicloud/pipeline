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

type featureRegistry struct {
	registry map[string]FeatureManager
}

// NewFeatureRegistry returns a new FeatureRegistry.
func NewFeatureRegistry(features map[string]FeatureManager) FeatureRegistry {
	return &featureRegistry{
		registry: features,
	}
}

// UnknownFeatureError is returned when there is no feature manager registered for a feature.
type UnknownFeatureError struct {
	FeatureName string
}

func (UnknownFeatureError) Error() string {
	return "unknown feature"
}

func (e UnknownFeatureError) Details() []interface{} {
	return []interface{}{"feature", e.FeatureName}
}

func (r *featureRegistry) GetFeatureManager(featureName string) (FeatureManager, error) {
	featureManager, ok := r.registry[featureName]
	if !ok {
		return nil, errors.WithStack(UnknownFeatureError{FeatureName: featureName})
	}

	return featureManager, nil
}
