// Copyright © 2021 Banzai Cloud
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

package cadence

import (
	"context"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/cadence/client"
)

// CronConfiguration encapsulates information about a cron workflow.
type CronConfiguration struct {
	CadenceClient                client.Client
	CronInstanceType             CronInstanceType
	CronSchedule                 string
	ExecutionStartToCloseTimeout time.Duration
	TaskListName                 string
	Workflow                     string
	WorkflowArguments            []interface{}
}

// NewCronConfiguration instantiates a cron configuration from the specified
// values.
func NewCronConfiguration(
	cadenceClient client.Client,
	cronInstanceType CronInstanceType,
	cronSchedule string,
	executionStartToCloseTimeout time.Duration,
	taskListName string,
	workflow string,
	workflowArguments ...interface{},
) CronConfiguration {
	return CronConfiguration{
		CadenceClient:                cadenceClient,
		CronInstanceType:             cronInstanceType,
		CronSchedule:                 cronSchedule,
		ExecutionStartToCloseTimeout: executionStartToCloseTimeout,
		TaskListName:                 taskListName,
		WorkflowArguments:            workflowArguments,
		Workflow:                     workflow,
	}
}

// CronWorkflowID returns the ID of the cron workflow based on its configuration.
func (cronConfig CronConfiguration) CronWorkflowID() (cronWorkflowID string) {
	switch cronConfig.CronInstanceType {
	case CronInstanceTypeDomain:
		return cronConfig.TaskListName + "-cron-" + cronConfig.Workflow
	}

	return string(cronConfig.CronInstanceType) + "-cron-" + cronConfig.Workflow
}

// StartCronWorkflow initiates a Cadence cron workflow from the specified type.
//
// WARNING: the current implementation allows only single instance crons (one
// per Cadence server).
func (cronConfig CronConfiguration) StartCronWorkflow(ctx context.Context) (err error) {
	state, err := cronConfig.WorkflowState(ctx)
	if err != nil {
		return errors.Wrap(err, "querying workflow state failed")
	}

	switch state {
	case CronWorkflowStateScheduled: // Note: nothing to do.
		return nil
	case CronWorkflowStateScheduledOutdated: // Note: restart required.
		err = cronConfig.CadenceClient.TerminateWorkflow(
			ctx,
			cronConfig.CronWorkflowID(),
			"",
			"cron workflow schedule requires an update",
			nil,
		)
		if err != nil {
			return errors.Wrap(err, "terminating cron workflow failed")
		}
	}

	cronWorkflowOptions := client.StartWorkflowOptions{
		ID:                           cronConfig.CronWorkflowID(),
		TaskList:                     cronConfig.TaskListName,
		ExecutionStartToCloseTimeout: cronConfig.ExecutionStartToCloseTimeout,
		CronSchedule:                 cronConfig.CronSchedule,
		Memo: map[string]interface{}{ // Note: CronSchedule is not directly retrievable (version 0.13.4-0.15.0).
			"CronSchedule": cronConfig.CronSchedule,
		},
	}

	_, err = cronConfig.CadenceClient.StartWorkflow(
		ctx,
		cronWorkflowOptions,
		cronConfig.Workflow,
		cronConfig.WorkflowArguments...,
	)
	if err != nil {
		return errors.Wrap(err, "starting cron workflow failed")
	}

	return nil
}

// WorkflowState queries the state of the cron workflow corresponding to the
// cron configuration.
func (cronConfig CronConfiguration) WorkflowState(ctx context.Context) (workflowState CronWorkflowState, err error) {
	cronWorkflowDescription, err := cronConfig.CadenceClient.DescribeWorkflowExecution(
		ctx,
		cronConfig.CronWorkflowID(),
		"",
	)
	if errors.As(err, new(*shared.EntityNotExistsError)) {
		return CronWorkflowStateNotScheduled, nil
	} else if err != nil {
		return CronWorkflowStateUnknown, errors.Wrap(err, "failed to query cron workflow")
	} else if cronWorkflowDescription == nil ||
		cronWorkflowDescription.WorkflowExecutionInfo == nil {
		return CronWorkflowStateUnknown, errors.New("cron workflow execution information not found")
	}

	executionInfo := cronWorkflowDescription.WorkflowExecutionInfo

	closeStatus := executionInfo.GetCloseStatus()
	// Note: https://cadenceworkflow.io/docs/go-client/distributed-cron cron
	// workflows only stop when cancelled or terminated.
	if closeStatus == shared.WorkflowExecutionCloseStatusCanceled ||
		closeStatus == shared.WorkflowExecutionCloseStatusTerminated {
		return CronWorkflowStateNotScheduled, nil
	}

	activeCronSchedule := ""
	if executionInfo.Memo != nil &&
		executionInfo.Memo.Fields["CronSchedule"] != nil {
		value := client.NewValue(executionInfo.Memo.Fields["CronSchedule"])
		if value.HasValue() {
			err = value.Get(&activeCronSchedule)
			if err != nil {
				return CronWorkflowStateUnknown, errors.Wrap(err, "retrieving cron schedule failed")
			}
		}
	}

	if activeCronSchedule != cronConfig.CronSchedule {
		return CronWorkflowStateScheduledOutdated, nil
	}

	return CronWorkflowStateScheduled, nil
}

// CronInstanceType determines how cron instances are treated in case of a
// multiworker environment.
type CronInstanceType string

const (
	// CronInstanceTypeDomain specifies only one instance of the scheduled cron
	// workflow can exist per Cadence domain.
	CronInstanceTypeDomain CronInstanceType = "domain"
)

// CronWorkflowState describes the state of a cron workflow which can be used
// for cron workflow operations.
type CronWorkflowState string

const (
	// CronWorkflowStateNotScheduled defines the state when no corresponding
	// cron workflow schedule exists.
	CronWorkflowStateNotScheduled CronWorkflowState = "not-scheduled"

	// CronWorkflowStateScheduled defines the state when the corresponding cron
	// workflow is scheduled to run with the latest known schedule.
	CronWorkflowStateScheduled CronWorkflowState = "scheduled"

	// CronWorkflowStateScheduledOutdated defines the state when the
	// corresponding cron workflow is scheduled to run using an outdated
	// schedule.
	CronWorkflowStateScheduledOutdated CronWorkflowState = "scheduled-outdated"

	// CronWorkflowStateUnknown defines the state when no information is
	// available about the corresponding cron workflow.
	CronWorkflowStateUnknown CronWorkflowState = "unknown"
)
