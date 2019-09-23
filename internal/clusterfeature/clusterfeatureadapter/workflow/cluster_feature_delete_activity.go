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

const ClusterFeatureDeleteActivityName = "cluster-feature-delete"

type ClusterFeatureDeleteActivityInput struct {
	ClusterID   uint
	FeatureName string
}

type ClusterFeatureDeleteActivity struct {
	features clusterfeature.FeatureRepository
}

func MakeClusterFeatureDeleteActivity(features clusterfeature.FeatureRepository) ClusterFeatureDeleteActivity {
	return ClusterFeatureDeleteActivity{
		features: features,
	}
}

func (a ClusterFeatureDeleteActivity) Execute(ctx context.Context, input ClusterFeatureDeleteActivityInput) error {
	return a.features.DeleteFeature(ctx, input.ClusterID, input.FeatureName)
}
