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
		StartToCloseTimeout:    3 * time.Hour,
		WaitForCancellation:    true,
	})

	jobsChannel := workflow.GetSignalChannel(ctx, ClusterFeatureJobSignalName)

	var signalInput ClusterFeatureJobSignalInput
	jobsChannel.Receive(ctx, &signalInput) // wait until the first job arrives

	if err := setClusterFeatureStatus(ctx, input, clusterfeature.FeatureStatusPending); err != nil {
		return err
	}

	if err := executeJobs(ctx, input, &signalInput, jobsChannel); err != nil {
		if err := setClusterFeatureStatus(ctx, input, clusterfeature.FeatureStatusError); err != nil {
			workflow.GetLogger(ctx).Error("failed to set cluster feature status", zap.Error(err))
		}
		return err
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
			ClusterID:     workflowInput.ClusterID,
			FeatureName:   workflowInput.FeatureName,
			FeatureSpec:   signalInput.FeatureSpec,
			RetryInterval: signalInput.RetryInterval,
		}, nil
	case OperationDeactivate:
		return ClusterFeatureDeactivateActivityName, ClusterFeatureDeactivateActivityInput{
			ClusterID:     workflowInput.ClusterID,
			FeatureName:   workflowInput.FeatureName,
			FeatureSpec:   signalInput.FeatureSpec,
			RetryInterval: signalInput.RetryInterval,
		}, nil
	default:
		return "", nil, errors.NewWithDetails("unsupported operation", "operation", op)
	}
}

func executeJobs(ctx workflow.Context, workflowInput ClusterFeatureJobWorkflowInput, signalInputPtr *ClusterFeatureJobSignalInput, jobsChannel workflow.Channel) error {
	for {
		activityName, activityInput, err := getActivity(workflowInput, *signalInputPtr)
		if err != nil {
			return err
		}

		{
			activityCtx, cancelActivity := workflow.WithCancel(ctx)

			activityFuture := workflow.ExecuteActivity(activityCtx, activityName, activityInput)

			selector := workflow.NewSelector(ctx)

			selector.AddFuture(activityFuture, func(f workflow.Future) {})

			selector.AddReceive(jobsChannel, func(c workflow.Channel, more bool) {
				cancelActivity()
			})

			selector.Select(ctx)

			err := activityFuture.Get(ctx, nil)

			if !getLatestValue(jobsChannel, signalInputPtr) {
				return err
			}
		}
	}
}

func getLatestValue(ch workflow.Channel, valuePtr interface{}) bool {
	received := false
	for ch.ReceiveAsync(valuePtr) {
		received = true
	}
	return received
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
