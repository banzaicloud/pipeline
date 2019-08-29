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

package workflow

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

const ClusterFeatureDeactivateActivityName = "cluster-feature-deactivate"

type ClusterFeatureDeactivateActivityInput struct {
	ClusterID   uint
	FeatureName string
}

type ClusterFeatureDeactivateActivity struct {
	features clusterfeature.FeatureOperatorRegistry
}

func MakeClusterFeatureDeactivateActivity(features clusterfeature.FeatureOperatorRegistry) ClusterFeatureDeactivateActivity {
	return ClusterFeatureDeactivateActivity{
		features: features,
	}
}

func (a ClusterFeatureDeactivateActivity) Execute(ctx context.Context, input ClusterFeatureDeactivateActivityInput) error {
	f, err := a.features.GetFeatureOperator(input.FeatureName)
	if err != nil {
		return err
	}
	return f.Deactivate(ctx, input.ClusterID)
}
