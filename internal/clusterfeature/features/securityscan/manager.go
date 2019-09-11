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

package securityscan

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

type featureManager struct {
}

//MakeFeatureManager creates asecurity scan feature manager instance
func MakeFeatureManager() featureManager {
	return featureManager{}
}

func (f featureManager) GetOutput(ctx context.Context, clusterID uint) (clusterfeature.FeatureOutput, error) {
	// todo implement this
	panic("implement me")
}

func (f featureManager) ValidateSpec(ctx context.Context, spec clusterfeature.FeatureSpec) error {
	securityScanSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: FeatureName,
			Problem:     err.Error(),
		}
	}

	if err := securityScanSpec.Validate(); err != nil {
		return clusterfeature.InvalidFeatureSpecError{
			FeatureName: FeatureName,
			Problem:     err.Error(),
		}
	}

	return nil
}

func (f featureManager) PrepareSpec(ctx context.Context, spec clusterfeature.FeatureSpec) (clusterfeature.FeatureSpec, error) {
	// todo implement this
	panic("implement me")
}

func (f featureManager) Name() string {
	return FeatureName
}
