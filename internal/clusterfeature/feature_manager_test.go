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
	"context"

	"emperror.dev/errors"
)

type dummyFeatureManager struct {
}

func (m *dummyFeatureManager) Deactivate(ctx context.Context, clusterID uint) error {
	switch clusterID {
	case 1:
		return errors.NewWithDetails("failed to deactivate", "clistetID", clusterID)

	default:
		return nil
	}

}

func (*dummyFeatureManager) Name() string {
	return "myFeature"
}

func (*dummyFeatureManager) Activate(ctx context.Context, clusterID uint, spec FeatureSpec) error {
	if spec["fail"] == true {
		return errors.NewWithDetails("failed to activate feature", "clusterId", clusterID, "feature", "myFeature")
	}

	return nil
}

func (m *dummyFeatureManager) ValidateSpec(ctx context.Context, spec FeatureSpec) error {
	if spec == nil {
		return InvalidFeatureSpecError{FeatureName: "myFeature", Problem: "empty spec"}
	}

	if spec["key"] != "value" {
		return InvalidFeatureSpecError{FeatureName: "myFeature", Problem: "key should have value"}
	}

	return nil
}

func (*dummyFeatureManager) Update(ctx context.Context, clusterID uint, spec FeatureSpec) error {
	return nil
}

func (*dummyFeatureManager) Details(ctx context.Context, clusterID uint) (*Feature, error) {
	return &Feature{
		Name: "myFeature",
		Spec: FeatureSpec{
			"key": "value",
		},
		Output: map[string]interface{}{},
		Status: FeatureStatusActive,
	}, nil
}
