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
	activityOptions := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Duration(workflow.GetInfo(ctx).ExecutionStartToCloseTimeoutSeconds) * time.Second,
	}

	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	processLog := process.NewProcessLog(
		workflow.WithStartToCloseTimeout(ctx, 10*time.Minute),
		input.OrganizationID,
		fmt.Sprint(input.ClusterID),
	)
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

		activityOptions := activityOptions
		activityOptions.StartToCloseTimeout = 5 * time.Minute
		activityOptions.RetryPolicy = &cadence.RetryPolicy{
			InitialInterval:          20 * time.Second,
			BackoffCoefficient:       1.1,
			MaximumAttempts:          10,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
		}

		processEvent := process.NewProcessEvent(workflow.WithStartToCloseTimeout(ctx, 10*time.Minute), UpdateNodeGroupActivityName)
		err = workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, activityOptions),
			UpdateNodeGroupActivityName,
			activityInput,
		).Get(ctx, nil)
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

		activityOptions := activityOptions
		activityOptions.StartToCloseTimeout = 40 * time.Minute // TODO: calculate based on desired node count
		activityOptions.HeartbeatTimeout = time.Minute
		activityOptions.RetryPolicy = &cadence.RetryPolicy{
			InitialInterval:          20 * time.Second,
			BackoffCoefficient:       1.1,
			MaximumAttempts:          20,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
		}

		processEvent := process.NewProcessEvent(workflow.WithStartToCloseTimeout(ctx, 10*time.Minute), WaitCloudFormationStackUpdateActivityName)

		err = workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, activityOptions),
			WaitCloudFormationStackUpdateActivityName,
			activityInput,
		).Get(ctx,nil)
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
