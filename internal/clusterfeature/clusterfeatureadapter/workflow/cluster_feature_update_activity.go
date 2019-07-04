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

const ClusterFeatureUpdateActivityName = "cluster-feature-update"

type ClusterFeatureUpdateActivityInput struct {
	ClusterID   uint
	FeatureName string
	FeatureSpec clusterfeature.FeatureSpec
}

type ClusterFeatureUpdateActivity struct {
	features clusterfeature.FeatureRegistry
}

func MakeClusterFeatureUpdateActivity(features clusterfeature.FeatureRegistry) ClusterFeatureUpdateActivity {
	return ClusterFeatureUpdateActivity{
		features: features,
	}
}

func (a ClusterFeatureUpdateActivity) Execute(ctx context.Context, input ClusterFeatureUpdateActivityInput) error {
	f, err := a.features.GetFeatureManager(input.FeatureName)
	if err != nil {
		return err
	}
	return f.Update(ctx, input.ClusterID, input.FeatureSpec)
}
