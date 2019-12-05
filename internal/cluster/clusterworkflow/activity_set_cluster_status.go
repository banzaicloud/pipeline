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

package clusterworkflow

import (
	"context"
	"time"

	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/cadence"
)

const SetClusterStatusActivityName = "set-cluster-status"

type SetClusterStatusActivity struct {
	store cluster.Store
}

func NewSetClusterStatusActivity(store cluster.Store) SetClusterStatusActivity {
	return SetClusterStatusActivity{
		store: store,
	}
}

type SetClusterStatusActivityInput struct {
	ClusterID     uint
	Status        string
	StatusMessage string
}

func (a SetClusterStatusActivity) Execute(ctx context.Context, input SetClusterStatusActivityInput) error {
	err := a.store.SetStatus(ctx, input.ClusterID, input.Status, input.StatusMessage)
	if err != nil {
		return cadence.WrapClientError(err)
	}

	return nil
}

func setClusterStatus(ctx workflow.Context, clusterID uint, status, statusMessage string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    2 * time.Minute,
		WaitForCancellation:    true,
	})

	return workflow.ExecuteActivity(ctx, SetClusterStatusActivityName, SetClusterStatusActivityInput{
		ClusterID:     clusterID,
		Status:        status,
		StatusMessage: statusMessage,
	}).Get(ctx, nil)
}
