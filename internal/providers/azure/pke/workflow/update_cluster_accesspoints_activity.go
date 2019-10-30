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

	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
)

const UpdateClusterAccessPointsActivityName = "pke-azure-upd-cluster-accesspoints"

type UpdateClusterAccessPointsActivity struct {
	store pke.AzurePKEClusterStore
}

func MakeUpdateClusterAccessPointsActivity(store pke.AzurePKEClusterStore) UpdateClusterAccessPointsActivity {
	return UpdateClusterAccessPointsActivity{
		store: store,
	}
}

type UpdateClusterAccessPointsActivityInput struct {
	ClusterID    uint
	AccessPoints pke.AccessPoints
}

func (a UpdateClusterAccessPointsActivity) Execute(ctx context.Context, input UpdateClusterAccessPointsActivityInput) error {
	return a.store.UpdateClusterAccessPoints(input.ClusterID, input.AccessPoints)
}

func updateClusterAccessPoints(ctx workflow.Context, clusterID uint, accessPoints pke.AccessPoints) error {
	return workflow.ExecuteActivity(ctx, UpdateClusterAccessPointsActivityName, UpdateClusterAccessPointsActivityInput{
		ClusterID:    clusterID,
		AccessPoints: accessPoints,
	}).Get(ctx, nil)
}
