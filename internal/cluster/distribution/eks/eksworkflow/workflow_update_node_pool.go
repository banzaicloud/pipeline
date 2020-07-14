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

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
	"github.com/banzaicloud/pipeline/pkg/sdk/cadence/lib/pipeline/processlog"
)

const UpdateNodePoolWorkflowName = "eks-update-node-pool"

type UpdateNodePoolWorkflow struct {
	processLogger processlog.ProcessLogger
}

// NewUpdateNodePoolWorkflow returns a new UpdateNodePoolWorkflow.
func NewUpdateNodePoolWorkflow(processLogger processlog.ProcessLogger) UpdateNodePoolWorkflow {
	return UpdateNodePoolWorkflow{
		processLogger: processLogger,
	}
}

type UpdateNodePoolWorkflowInput struct {
	ProviderSecretID string
	Region           string

	StackName string

	OrganizationID  uint
	ClusterID       uint
	ClusterSecretID string
	ClusterName     string
	NodePoolName    string

	NodeImage string

	Options eks.NodePoolUpdateOptions

	ClusterTags map[string]string
}

func (w UpdateNodePoolWorkflow) Register() {
	workflow.RegisterWithOptions(w.Execute, workflow.RegisterOptions{Name: UpdateNodePoolWorkflowName})
}

func (w UpdateNodePoolWorkflow) Execute(ctx workflow.Context, input UpdateNodePoolWorkflowInput) (err error) {
	activityOptions := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Duration(workflow.GetInfo(ctx).ExecutionStartToCloseTimeoutSeconds) * time.Second,
	}

	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	clusterID := brn.New(input.OrganizationID, brn.ClusterResourceType, fmt.Sprint(input.ClusterID))

	process := w.processLogger.StartProcess(ctx, clusterID.String())
	defer func() {
		process.Finish(ctx, err)
	}()
	defer func() {
		status := cluster.Running
		statusMessage := cluster.RunningMessage

		if err != nil {
			if cadence.IsCanceledError(err) {
				ctx, _ = workflow.NewDisconnectedContext(ctx)
			}

			status = cluster.Warning
			statusMessage = fmt.Sprintf("failed to update node pool: %s", err.Error())
		}

		_ = setClusterStatus(ctx, input.ClusterID, status, statusMessage)
	}()

	var nodePoolVersion string
	{
		activityInput := CalculateNodePoolVersionActivityInput{
			Image: input.NodeImage,
		}

		activityOptions := activityOptions
		activityOptions.StartToCloseTimeout = 30 * time.Second
		activityOptions.RetryPolicy = &cadence.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 1.01,
			MaximumAttempts:    10,
			MaximumInterval:    10 * time.Minute,
		}

		var output CalculateNodePoolVersionActivityOutput

		err = workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, activityOptions),
			CalculateNodePoolVersionActivityName,
			activityInput,
		).Get(ctx, &output)
		if err != nil {
			return
		}

		nodePoolVersion = output.Version
	}

	{
		activityInput := UpdateNodeGroupActivityInput{
			SecretID:        input.ProviderSecretID,
			Region:          input.Region,
			ClusterName:     input.ClusterName,
			StackName:       input.StackName,
			NodePoolName:    input.NodePoolName,
			NodePoolVersion: nodePoolVersion,
			NodeImage:       input.NodeImage,
			MaxBatchSize:    input.Options.MaxBatchSize,
			ClusterTags:     input.ClusterTags,
		}

		activityOptions := activityOptions
		activityOptions.StartToCloseTimeout = 5 * time.Minute
		activityOptions.RetryPolicy = &cadence.RetryPolicy{
			InitialInterval:          20 * time.Second,
			BackoffCoefficient:       1.1,
			MaximumAttempts:          10,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic", ErrReasonStackFailed},
		}

		var output UpdateNodeGroupActivityOutput

		processActivity := process.StartActivity(ctx, UpdateNodeGroupActivityName)
		err = workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, activityOptions),
			UpdateNodeGroupActivityName,
			activityInput,
		).Get(ctx, &output)
		processActivity.Finish(ctx, err)
		if err != nil || !output.NodePoolChanged {
			return
		}
	}

	{
		activityInput := WaitCloudFormationStackUpdateActivityInput{
			SecretID:  input.ProviderSecretID,
			Region:    input.Region,
			StackName: input.StackName,
		}

		activityOptions := activityOptions
		activityOptions.StartToCloseTimeout = 100 * 10 * time.Minute // TODO: calculate based on desired node count (limited to around 100 nodes now)
		activityOptions.HeartbeatTimeout = time.Minute
		activityOptions.RetryPolicy = &cadence.RetryPolicy{
			InitialInterval:          20 * time.Second,
			BackoffCoefficient:       1.1,
			MaximumAttempts:          20,
			NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
		}

		processActivity := process.StartActivity(ctx, WaitCloudFormationStackUpdateActivityName)
		err = decodeCloudFormationError(workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, activityOptions),
			WaitCloudFormationStackUpdateActivityName,
			activityInput,
		).Get(ctx, nil))
		processActivity.Finish(ctx, err)
		if err != nil {
			return err
		}
	}

	return nil
}
