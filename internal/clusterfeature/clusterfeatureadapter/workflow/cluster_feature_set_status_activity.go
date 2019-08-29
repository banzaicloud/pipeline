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

const ClusterFeatureSetStatusActivityName = "cluster-feature-set-status"

type ClusterFeatureSetStatusActivityInput struct {
	ClusterID   uint
	FeatureName string
	Status      string
}

type ClusterFeatureSetStatusActivity struct {
	features clusterfeature.FeatureRepository
}

func MakeClusterFeatureSetStatusActivity(features clusterfeature.FeatureRepository) ClusterFeatureSetStatusActivity {
	return ClusterFeatureSetStatusActivity{
		features: features,
	}
}

func (a ClusterFeatureSetStatusActivity) Execute(ctx context.Context, input ClusterFeatureSetStatusActivityInput) error {
	return a.features.UpdateFeatureStatus(ctx, input.ClusterID, input.FeatureName, input.Status)
}
