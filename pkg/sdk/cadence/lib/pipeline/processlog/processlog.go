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

package processlog

import (
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
)

// ProcessLogger keeps track of long-running processes.
type ProcessLogger interface {
	// StartProcess records the beginning of a process.
	StartProcess(ctx workflow.Context, resourceID string) Process
}

// Process is a long-running job/workflow/whatever that includes activities.
type Process interface {
	// Finish records the end of a process.
	Finish(ctx workflow.Context, err error)

	// StartActivity records a new activity of a process.
	StartActivity(ctx workflow.Context, typ string) Activity
}

// Activity is a short lived part of a Process.
type Activity interface {
	// Finish records the end of a process.
	Finish(ctx workflow.Context, err error)
}

// New returns a new ProcessLogger.
func New() ProcessLogger {
	return processLogger{}
}

type processLogger struct{}

func (p processLogger) StartProcess(ctx workflow.Context, resourceID string) Process {
	ctx = withContext(ctx)

	winfo := workflow.GetInfo(ctx)
	parentID := ""

	if winfo.ParentWorkflowExecution != nil {
		parentID = winfo.ParentWorkflowExecution.ID
	}

	resourceName, err := brn.Parse(resourceID)
	if err != nil {
		workflow.GetLogger(ctx).Sugar().Errorf("failed to parse resource ID: %s", err)

		panic(err)
	}

	activityInput := processActivityInput{
		ID:           winfo.WorkflowExecution.ID,
		ParentID:     parentID,
		Type:         winfo.WorkflowType.Name,
		StartedAt:    workflow.Now(ctx),
		Status:       running,
		OrgID:        int32(resourceName.OrganizationID),
		ResourceID:   resourceName.ResourceID,
		ResourceType: resourceName.ResourceType,
	}

	err = workflow.ExecuteActivity(ctx, processActivityName, activityInput).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Sugar().Warnf("failed to log process: %s", err)
	}

	return &process{activityInput: activityInput}
}

const processActivityName = "process"

type processActivityInput struct {
	ID           string
	ParentID     string
	OrgID        int32
	Type         string
	Log          string
	ResourceID   string
	ResourceType string
	Status       status
	StartedAt    time.Time
	FinishedAt   *time.Time
}

type status string

const (
	running  status = "running"
	failed   status = "failed"
	finished status = "finished"
	canceled status = "canceled"
)

type process struct {
	activityInput processActivityInput
}

func (p process) Finish(ctx workflow.Context, err error) {
	ctx = withContext(ctx)

	finishedAt := workflow.Now(ctx)

	activityInput := p.activityInput

	activityInput.FinishedAt = &finishedAt
	if err != nil {
		if cadence.IsCanceledError(err) {
			ctx, _ = workflow.NewDisconnectedContext(ctx)

			activityInput.Status = canceled
		} else {
			activityInput.Status = failed
		}

		activityInput.Log = err.Error()
	} else {
		activityInput.Status = finished
	}

	err = workflow.ExecuteActivity(ctx, processActivityName, activityInput).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Sugar().Warnf("failed to log process end: %s", err)
	}
}

func (p process) StartActivity(ctx workflow.Context, typ string) Activity {
	ctx = withContext(ctx)

	winfo := workflow.GetInfo(ctx)

	activityInput := processActivityActivityInput{
		ProcessID: winfo.WorkflowExecution.ID,
		Type:      typ,
		Timestamp: workflow.Now(ctx),
		Status:    running,
	}

	err := workflow.ExecuteActivity(ctx, processActivityActivityName, activityInput).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Sugar().Warnf("failed to log process activity: %s", err)
	}

	return &processActivity{activityInput: activityInput}
}

const processActivityActivityName = "process-event"

type processActivityActivityInput struct {
	ProcessID string
	Type      string
	Log       string
	Status    status
	Timestamp time.Time
}

type processActivity struct {
	activityInput processActivityActivityInput
}

func (a processActivity) Finish(ctx workflow.Context, err error) {
	ctx = withContext(ctx)

	activityInput := a.activityInput

	activityInput.Timestamp = workflow.Now(ctx)
	if err != nil {
		if cadence.IsCanceledError(err) {
			ctx, _ = workflow.NewDisconnectedContext(ctx)

			activityInput.Status = canceled
		} else {
			activityInput.Status = failed
		}

		activityInput.Log = err.Error()
	} else {
		activityInput.Status = finished
	}

	err = workflow.ExecuteActivity(ctx, processActivityActivityName, activityInput).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Sugar().Warnf("failed to log process activity end: %s", err)
	}
}

func withContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskList:               "pipeline",
		ScheduleToStartTimeout: time.Duration(workflow.GetInfo(ctx).ExecutionStartToCloseTimeoutSeconds) * time.Second,
		StartToCloseTimeout:    30 * time.Second,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 1.4,
			MaximumInterval:    3 * time.Minute,
			MaximumAttempts:    10,
		},
	})
}
