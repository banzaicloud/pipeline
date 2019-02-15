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
	"time"

	"go.uber.org/cadence/workflow"
)

const DeleteClusterWorkflowName = "pke-delete-cluster"

type DeleteClusterWorkflowInput struct {
	ClusterID uint
}

func DeleteClusterWorkflow(ctx workflow.Context, input DeleteClusterWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	var nodePools []NodePool
	listNodePoolsActivityInput := ListNodePoolsActivityInput{
		ClusterID: input.ClusterID,
	}

	if err := workflow.ExecuteActivity(ctx, ListNodePoolsActivityName, listNodePoolsActivityInput).Get(ctx, &nodePools); err != nil {
		return err
	}

	var poolActivities []workflow.Future

	for _, np := range nodePools {
		if !np.Master && np.Worker {
			deleteWorkerPoolActivityInput := DeleteWorkerPoolActivityInput{
				ClusterID: input.ClusterID,
				Pool:      np,
			}

			future := workflow.ExecuteActivity(ctx, DeleteWorkerPoolActivityName, deleteWorkerPoolActivityInput)
			poolActivities = append(poolActivities, future)
		}
	}

	for _, future := range poolActivities {
		if err := future.Get(ctx, nil); err != nil {
			return err
		}
	}

	return nil
}
