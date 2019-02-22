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

package pkeworkflow

import (
	"context"
)

const UpdateClusterStatusActivityName = "pke-update-cluster-status-activity"

type UpdateClusterStatusActivity struct {
	clusters Clusters
}

func NewUpdateClusterStatusActivity(clusters Clusters) *UpdateClusterStatusActivity {
	return &UpdateClusterStatusActivity{
		clusters: clusters,
	}
}

type UpdateClusterStatusActivityInput struct {
	ClusterID     uint
	Status        string
	StatusMessage string
}

func (a *UpdateClusterStatusActivity) Execute(ctx context.Context, input UpdateClusterStatusActivityInput) error {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	return c.UpdateStatus(input.Status, input.StatusMessage)
}
