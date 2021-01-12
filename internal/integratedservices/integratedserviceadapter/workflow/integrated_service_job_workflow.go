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

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

// IntegratedServiceJobWorkflowName is the name the IntegratedServiceJobWorkflow is registered under
const IntegratedServiceJobWorkflowName = "integrated-service-job"

// IntegratedServiceJobWorkflowV2Name name of the v2 integrated service workflow
const IntegratedServiceJobWorkflowV2Name = "integrated-service-job-v2"

// IntegratedServiceJobSignalName is the name of signal with which jobs can be sent to the workflow
const IntegratedServiceJobSignalName = "job"

const (
	// OperationApply identifies the integrated service apply operation
	OperationApply = "apply"
	// OperationDeactivate identifies the integrated service deactivation operation
	OperationDeactivate = "deactivate"
)

// integratedServicesV2 readable flag for signaling the implementation version
const integratedServicesV2 bool = true

// IntegratedServiceJobWorkflowInput defines the fixed inputs of the IntegratedServiceJobWorkflow
type IntegratedServiceJobWorkflowInput struct {
	ClusterID             uint
	IntegratedServiceName string
}

// IntegratedServiceJobSignalInput defines the dynamic inputs of the IntegratedServiceJobWorkflow
type IntegratedServiceJobSignalInput struct {
	Operation              string
	IntegratedServiceSpecs integratedservices.IntegratedServiceSpec
	RetryInterval          time.Duration
}

// IntegratedServiceJobWorkflow executes integrated service jobs
func IntegratedServiceJobWorkflow(ctx workflow.Context, input IntegratedServiceJobWorkflowInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: 15 * time.Minute,
		StartToCloseTimeout:    3 * time.Hour,
		WaitForCancellation:    true,
	})

	jobsChannel := workflow.GetSignalChannel(ctx, IntegratedServiceJobSignalName)

	var signalInput IntegratedServiceJobSignalInput
	jobsChannel.Receive(ctx, &signalInput) // wait until the first job arrives

	if err := setIntegratedServiceStatus(ctx, input, integratedservices.IntegratedServiceStatusPending); err != nil {
		return err
	}

	if err := executeJobs(ctx, input, &signalInput, jobsChannel, !integratedServicesV2); err != nil {
		if err := setIntegratedServiceStatus(ctx, input, integratedservices.IntegratedServiceStatusError); err != nil {
			workflow.GetLogger(ctx).Error("failed to set integrated service status", zap.Error(err))
		}
		return err
	}

	switch op := signalInput.Operation; op {
	case OperationApply:
		if err := setIntegratedServiceStatus(ctx, input, integratedservices.IntegratedServiceStatusActive); err != nil {
			return err
		}
	case OperationDeactivate:
		if err := deleteIntegratedService(ctx, input, false); err != nil {
			return err
		}
	default:
		workflow.GetLogger(ctx).Error("unsupported operation", zap.String("operation", op))
	}

	return nil
}

// IntegratedServiceJobWorkflowV2 workflow that skips status updates (and all database operations related to integrated services)
func IntegratedServiceJobWorkflowV2(ctx workflow.Context, input IntegratedServiceJobWorkflowInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: 15 * time.Minute,
		StartToCloseTimeout:    3 * time.Hour,
		WaitForCancellation:    true,
	})

	jobsChannel := workflow.GetSignalChannel(ctx, IntegratedServiceJobSignalName)

	var signalInput IntegratedServiceJobSignalInput
	jobsChannel.Receive(ctx, &signalInput) // wait until the first job arrives

	if err := executeJobs(ctx, input, &signalInput, jobsChannel, integratedServicesV2); err != nil {
		return err
	}

	return nil
}

func getActivity(workflowInput IntegratedServiceJobWorkflowInput, signalInput IntegratedServiceJobSignalInput, isV2 bool) (string, interface{}, error) {
	switch op := signalInput.Operation; op {
	case OperationApply:
		return GetActivityName(IntegratedServiceApplyActivityName, isV2), IntegratedServiceApplyActivityInput{
			ClusterID:             workflowInput.ClusterID,
			IntegratedServiceName: workflowInput.IntegratedServiceName,
			IntegratedServiceSpec: signalInput.IntegratedServiceSpecs,
			RetryInterval:         signalInput.RetryInterval,
		}, nil
	case OperationDeactivate:
		return GetActivityName(IntegratedServiceDeactivateActivityName, isV2), IntegratedServiceDeactivateActivityInput{
			ClusterID:             workflowInput.ClusterID,
			IntegratedServiceName: workflowInput.IntegratedServiceName,
			IntegratedServiceSpec: signalInput.IntegratedServiceSpecs,
			RetryInterval:         signalInput.RetryInterval,
		}, nil
	default:
		return "", nil, errors.NewWithDetails("unsupported operation", "operation", op)
	}
}

func executeJobs(ctx workflow.Context, workflowInput IntegratedServiceJobWorkflowInput, signalInputPtr *IntegratedServiceJobSignalInput, jobsChannel workflow.Channel, isV2 bool) error {
	for {
		activityName, activityInput, err := getActivity(workflowInput, *signalInputPtr, isV2)
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

func setIntegratedServiceStatus(ctx workflow.Context, input IntegratedServiceJobWorkflowInput, status string) error {
	activityInput := IntegratedServiceSetStatusActivityInput{
		ClusterID:             input.ClusterID,
		IntegratedServiceName: input.IntegratedServiceName,
		Status:                status,
	}
	return workflow.ExecuteActivity(ctx, IntegratedServiceSetStatusActivityName, activityInput).Get(ctx, nil)
}

func deleteIntegratedService(ctx workflow.Context, input IntegratedServiceJobWorkflowInput, isV2 bool) error {
	activityInput := IntegratedServiceDeleteActivityInput{
		ClusterID:             input.ClusterID,
		IntegratedServiceName: input.IntegratedServiceName,
	}
	return workflow.ExecuteActivity(ctx, GetActivityName(IntegratedServiceDeleteActivityName, isV2), activityInput).Get(ctx, nil)
}
