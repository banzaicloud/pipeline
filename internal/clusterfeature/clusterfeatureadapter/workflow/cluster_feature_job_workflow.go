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
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

// ClusterFeatureJobWorkflowName is the name the ClusterFeatureJobWorkflow is registered under
const ClusterFeatureJobWorkflowName = "cluster-feature-job"

// ClusterFeatureJobSignalName is the name of signal with which jobs can be sent to the workflow
const ClusterFeatureJobSignalName = "job"

const (
	// OperationApply identifies the cluster feature apply operation
	OperationApply = "apply"
	// OperationDeactivate identifies the cluster feature deactivation operation
	OperationDeactivate = "deactivate"
)

// ClusterFeatureJobWorkflowInput defines the fixed inputs of the ClusterFeatureJobWorkflow
type ClusterFeatureJobWorkflowInput struct {
	ClusterID   uint
	FeatureName string
}

// ClusterFeatureJobSignalInput defines the dynamic inputs of the ClusterFeatureJobWorkflow
type ClusterFeatureJobSignalInput struct {
	Operation     string
	FeatureSpec   clusterfeature.FeatureSpec
	RetryInterval time.Duration
}

// ClusterFeatureJobWorkflow executes cluster feature jobs
func ClusterFeatureJobWorkflow(ctx workflow.Context, input ClusterFeatureJobWorkflowInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: 15 * time.Minute,
		StartToCloseTimeout:    15 * time.Minute,
		WaitForCancellation:    true,
	})

	jobsChannel := workflow.GetSignalChannel(ctx, ClusterFeatureJobSignalName)

	var signalInput ClusterFeatureJobSignalInput
	jobsChannel.Receive(ctx, &signalInput) // wait until the first job arrives

	if err := setClusterFeatureStatus(ctx, input, clusterfeature.FeatureStatusPending); err != nil {
		return err
	}

	for {
		if err := executeJob(ctx, input, signalInput, jobsChannel); err != nil {
			if err := setClusterFeatureStatus(ctx, input, clusterfeature.FeatureStatusError); err != nil {
				workflow.GetLogger(ctx).Error("failed to set cluster feature status", zap.Error(err))
			}
			return err
		}

		if !jobsChannel.ReceiveAsync(&signalInput) {
			break
		}
	}

	switch op := signalInput.Operation; op {
	case OperationApply:
		if err := setClusterFeatureStatus(ctx, input, clusterfeature.FeatureStatusActive); err != nil {
			return err
		}
	case OperationDeactivate:
		if err := deleteClusterFeature(ctx, input); err != nil {
			return err
		}
	default:
		workflow.GetLogger(ctx).Error("unsupported operation", zap.String("operation", op))
	}

	return nil
}

func getActivity(workflowInput ClusterFeatureJobWorkflowInput, signalInput ClusterFeatureJobSignalInput) (string, interface{}, error) {
	switch op := signalInput.Operation; op {
	case OperationApply:
		return ClusterFeatureApplyActivityName, ClusterFeatureApplyActivityInput{
			ClusterID:   workflowInput.ClusterID,
			FeatureName: workflowInput.FeatureName,
			FeatureSpec: signalInput.FeatureSpec,
		}, nil
	case OperationDeactivate:
		return ClusterFeatureDeactivateActivityName, ClusterFeatureDeactivateActivityInput{
			ClusterID:   workflowInput.ClusterID,
			FeatureName: workflowInput.FeatureName,
		}, nil
	default:
		return "", nil, errors.NewWithDetails("unsupported operation", "operation", op)
	}
}

func tryExecuteActivity(ctx workflow.Context, activityName string, activityInput interface{}) (bool, error) {
	err := workflow.ExecuteActivity(ctx, activityName, activityInput).Get(ctx, nil)
	return shouldRetry(err), errors.WrapIfWithDetails(err, "activity execution failed", "activityName", activityName, "activityInput", activityInput)
}

func executeActivity(ctx workflow.Context, activityName string, activityInput interface{}, jobsChannel workflow.Channel, signalInputPtr *ClusterFeatureJobSignalInput) (bool, error) {
	for {
		retry, err := tryExecuteActivity(ctx, activityName, activityInput)
		if retry {
			again := false

			// wait for retry and listen for new jobs
			workflow.NewSelector(ctx).AddFuture(workflow.NewTimer(ctx, signalInputPtr.RetryInterval), func(f workflow.Future) {
				again = true
			}).AddReceive(jobsChannel, func(c workflow.Channel, more bool) {
				c.Receive(ctx, signalInputPtr)
			}).Select(ctx)

			if again {
				continue
			}

			return true, nil
		}

		return false, err
	}
}

func executeJob(ctx workflow.Context, workflowInput ClusterFeatureJobWorkflowInput, signalInput ClusterFeatureJobSignalInput, jobsChannel workflow.Channel) error {
	for {
		activityName, activityInput, err := getActivity(workflowInput, signalInput)
		if err != nil {
			return err
		}

		newJob, err := executeActivity(ctx, activityName, activityInput, jobsChannel, &signalInput)
		if newJob {
			continue
		}

		return err
	}
}

func setClusterFeatureStatus(ctx workflow.Context, input ClusterFeatureJobWorkflowInput, status string) error {
	activityInput := ClusterFeatureSetStatusActivityInput{
		ClusterID:   input.ClusterID,
		FeatureName: input.FeatureName,
		Status:      status,
	}
	return workflow.ExecuteActivity(ctx, ClusterFeatureSetStatusActivityName, activityInput).Get(ctx, nil)
}

func deleteClusterFeature(ctx workflow.Context, input ClusterFeatureJobWorkflowInput) error {
	activityInput := ClusterFeatureDeleteActivityInput{
		ClusterID:   input.ClusterID,
		FeatureName: input.FeatureName,
	}
	return workflow.ExecuteActivity(ctx, ClusterFeatureDeleteActivityName, activityInput).Get(ctx, nil)
}

func shouldRetry(err error) bool {
	var sh interface {
		ShouldRetry() bool
	}
	if errors.As(err, &sh) {
		return sh.ShouldRetry()
	}
	return false
}
