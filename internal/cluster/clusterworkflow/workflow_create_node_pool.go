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

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
)

const CreateNodePoolWorkflowName = "create-node-pool"

type CreateNodePoolWorkflowInput struct {
	ClusterID   uint
	UserID      uint
	RawNodePool cluster.NewRawNodePool
}

func CreateNodePoolWorkflow(ctx workflow.Context, input CreateNodePoolWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          15 * time.Second,
			BackoffCoefficient:       1.0,
			MaximumAttempts:          30,
			NonRetriableErrorReasons: []string{pkgCadence.ClientErrorReason, "cadenceInternal:Panic"},
		},
	}
	_ctx := ctx
	ctx = workflow.WithActivityOptions(ctx, ao)

	{
		input := CreateNodePoolLabelSetActivityInput{
			ClusterID:   input.ClusterID,
			RawNodePool: input.RawNodePool,
		}

		err := workflow.ExecuteActivity(ctx, CreateNodePoolLabelSetActivityName, input).Get(ctx, nil)
		if err != nil {
			_ = setClusterStatus(_ctx, input.ClusterID, cluster.Warning, pkgCadence.UnwrapError(err).Error())

			return err
		}
	}

	{
		input := CreateNodePoolActivityInput{
			ClusterID:   input.ClusterID,
			UserID:      input.UserID,
			RawNodePool: input.RawNodePool,
		}

		err := workflow.ExecuteActivity(ctx, CreateNodePoolActivityName, input).Get(ctx, nil)
		if err != nil {
			_ = setClusterStatus(_ctx, input.ClusterID, cluster.Warning, pkgCadence.UnwrapError(err).Error())

			return err
		}
	}

	{
		input := SetClusterStatusActivityInput{
			ClusterID:     input.ClusterID,
			Status:        cluster.Running,
			StatusMessage: cluster.RunningMessage,
		}

		err := workflow.ExecuteActivity(ctx, SetClusterStatusActivityName, input).Get(ctx, nil)
		if err != nil {
			_ = setClusterStatus(_ctx, input.ClusterID, cluster.Warning, pkgCadence.UnwrapError(err).Error())

			return err
		}
	}

	return nil
}
