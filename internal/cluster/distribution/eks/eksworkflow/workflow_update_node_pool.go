// Copyright © 2020 Banzai Cloud
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

package eksworkflow

import (
	"fmt"
	"time"

	"github.com/banzaicloud/pipeline-sdk/process"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

const UpdateNodePoolWorkflowName = "eks-update-node-pool"

type UpdateNodePoolWorkflowInput struct {
	SecretID string
	Region   string

	StackName string

	OrganizationID uint
	ClusterID      uint
	ClusterName    string
	NodePoolName   string

	NodeImage string
}

func UpdateNodePoolWorkflow(ctx workflow.Context, input UpdateNodePoolWorkflowInput) (err error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Duration(workflow.GetInfo(ctx).ExecutionStartToCloseTimeoutSeconds) * time.Second,
		StartToCloseTimeout:    time.Duration(workflow.GetInfo(ctx).ExecutionStartToCloseTimeoutSeconds) * time.Second,
	})

	processLog := process.NewProcessLog(ctx, input.OrganizationID, fmt.Sprint(input.ClusterID))
	defer processLog.End(err)

	{
		activityInput := UpdateNodeGroupActivityInput{
			SecretID:     input.SecretID,
			Region:       input.Region,
			ClusterName:  input.ClusterName,
			NodePoolName: input.NodePoolName,
			StackName:    input.StackName,
			NodeImage:    input.NodeImage,
		}

		// TODO: improve
		ctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			ScheduleToStartTimeout: 20 * time.Minute,
			StartToCloseTimeout:    40 * time.Minute,
			RetryPolicy: &cadence.RetryPolicy{
				InitialInterval:          2 * time.Second,
				BackoffCoefficient:       1.5,
				MaximumInterval:          30 * time.Second,
				MaximumAttempts:          5,
				NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
			},
		})

		processEvent := process.NewProcessEvent(ctx, UpdateNodeGroupActivityName)
		err = workflow.ExecuteActivity(ctx, UpdateNodeGroupActivityName, activityInput).Get(ctx, nil)
		processEvent.End(err)
		if err != nil {
			_ = setClusterStatus(ctx, input.ClusterID, cluster.Warning, fmt.Sprintf("failed to update node pool: %s", err.Error()))

			return err
		}
	}

	// TODO: get current count of the ASG to calculate a timeout
	{
		activityInput := WaitCloudFormationStackUpdateActivityInput{
			SecretID:  input.SecretID,
			Region:    input.Region,
			StackName: input.StackName,
		}

		// TODO: improve
		ctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			ScheduleToStartTimeout: 20 * time.Minute,
			StartToCloseTimeout:    40 * time.Minute,
			WaitForCancellation:    true,
			HeartbeatTimeout:       45 * time.Second,
			RetryPolicy: &cadence.RetryPolicy{
				InitialInterval:          2 * time.Second,
				BackoffCoefficient:       1.5,
				MaximumInterval:          30 * time.Second,
				MaximumAttempts:          5,
				NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
			},
		})

		processEvent := process.NewProcessEvent(ctx, WaitCloudFormationStackUpdateActivityName)
		err = workflow.ExecuteActivity(ctx, WaitCloudFormationStackUpdateActivityName, activityInput).Get(ctx, nil)
		processEvent.End(err)
		if err != nil {
			_ = setClusterStatus(ctx, input.ClusterID, cluster.Warning, fmt.Sprintf("failed to update node pool: %s", err.Error()))

			return err
		}
	}

	err = setClusterStatus(ctx, input.ClusterID, cluster.Running, cluster.RunningMessage)
	if err != nil {
		return err
	}

	return nil
}
