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
	"time"

	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
)

const DeleteClusterWorkflowName = "delete-cluster"

type DeleteClusterWorkflowInput struct {
	ClusterID uint
	Force     bool
}

func DeleteClusterWorkflow(ctx workflow.Context, input DeleteClusterWorkflowInput) error {
	{
		ctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			ScheduleToStartTimeout: 5 * time.Minute,
			StartToCloseTimeout:    10 * time.Minute,
		})

		activityInput := RemoveClusterFromGroupActivityInput{
			ClusterID: input.ClusterID,
		}
		err := workflow.ExecuteActivity(ctx, RemoveClusterFromGroupActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			_ = setClusterStatus(ctx, input.ClusterID, cluster.Error, pkgCadence.UnwrapError(err).Error())
			return err
		}
	}

	{
		ctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			ScheduleToStartTimeout: 5 * time.Minute,
			StartToCloseTimeout:    30 * time.Minute,
		})

		activityInput := DeleteClusterActivityInput{
			ClusterID: input.ClusterID,
			Force:     input.Force,
		}
		err := workflow.ExecuteActivity(ctx, DeleteClusterActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			_ = setClusterStatus(ctx, input.ClusterID, cluster.Error, pkgCadence.UnwrapError(err).Error())
			return err
		}
	}

	return nil
}
