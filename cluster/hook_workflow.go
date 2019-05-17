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

package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

const RunPostHooksWorkflowName = "run-posthooks"

type RunPostHooksWorkflowInput struct {
	ClusterID uint
	PostHooks []RunPostHooksWorkflowInputPostHook
}

type RunPostHooksWorkflowInputPostHook struct {
	Name  string
	Param interface{}
}

func RunPostHooksWorkflow(ctx workflow.Context, input RunPostHooksWorkflowInput) error {
	retryPolicy := &cadence.RetryPolicy{
		InitialInterval:    time.Second * 3,
		BackoffCoefficient: 2,
		ExpirationInterval: time.Minute * 3,
		MaximumAttempts:    5,
	}
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    30 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy:            retryPolicy,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	for _, hook := range input.PostHooks {
		activityInput := RunPostHookActivityInput{
			ClusterID: input.ClusterID,
			HookName:  hook.Name,
			HookParam: hook.Param,
		}

		err := workflow.ExecuteActivity(ctx, RunPostHookActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	// Update cluster status
	{
		activityInput := UpdateClusterStatusActivityInput{
			ClusterID:     input.ClusterID,
			Status:        pkgCluster.Running,
			StatusMessage: pkgCluster.RunningMessage,
		}

		err := workflow.ExecuteActivity(ctx, UpdateClusterStatusActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

const RunPostHookActivityName = "run-posthook"

type RunPostHookActivityInput struct {
	ClusterID uint
	HookName  string
	HookParam interface{}
	Status    string
}

type RunPostHookActivity struct {
	manager *Manager
}

func NewRunPostHookActivity(manager *Manager) *RunPostHookActivity {
	return &RunPostHookActivity{
		manager: manager,
	}
}
func (a *RunPostHookActivity) Execute(ctx context.Context, input RunPostHookActivityInput) error {
	hook, ok := HookMap[input.HookName]
	if !ok {
		return errors.New("hook function not found")
	}

	if hookWithParam, ok := hook.(*PostFunctionWithParam); ok {
		hookWithParamCopy := *hookWithParam // This is to avoid bugs caused by the global nature of posthooks
		hookWithParamCopy.SetParams(input.HookParam)
		hook = &hookWithParamCopy
	}

	cluster, err := a.manager.GetClusterByIDOnly(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	info := activity.GetInfo(ctx)
	logger := activity.GetLogger(ctx).Sugar().With(
		"clusterID", input.ClusterID,
		"postHook", input.HookName,
		"workflowID", info.WorkflowExecution.ID,
		"workflowRunID", info.WorkflowExecution.RunID,
	)

	logger.Infow("starting posthook function", "param", input.HookParam)

	statusMsg := fmt.Sprintf("running %s", hook)
	if err := hook.Do(cluster); err != nil {
		err := emperror.Wrap(err, "posthook failed")
		hook.Error(cluster, err)

		return err
	}

	status := input.Status
	if status == "" {
		status = pkgCluster.Creating
	}
	if err := cluster.SetStatus(status, statusMsg); err != nil {
		return emperror.Wrap(err, "failed to write status to db")
	}

	return nil
}
