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

package process

import (
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
)

const ProcessActivityName = "process"

const ProcessEventActivityName = "process-event"

type Status string

const (
	Running  Status = "running"
	Failed   Status = "failed"
	Finished Status = "finished"
	Canceled Status = "canceled"
)

type ProcessActivityInput struct {
	ID         string
	ParentID   string
	OrgID      int32
	Type       string
	Log        string
	ResourceID string
	Status     Status
	StartedAt  time.Time
	FinishedAt *time.Time
}

type ProcessEventActivityInput struct {
	ProcessID string
	Type      string
	Log       string
	Status    Status
	Timestamp time.Time
}

type Process interface {
	RecordEnd(error)
}

type Event interface {
	RecordEnd(error)
}

type process struct {
	ctx           workflow.Context
	activityInput ProcessActivityInput
}

func Start(ctx workflow.Context, orgID uint, resourceID string) Process {
	ctx = workflow.WithTaskList(ctx, "pipeline")

	winfo := workflow.GetInfo(ctx)
	parentID := ""
	if winfo.ParentWorkflowExecution != nil {
		parentID = winfo.ParentWorkflowExecution.ID
	}
	activityInput := ProcessActivityInput{
		ID:         winfo.WorkflowExecution.ID,
		ParentID:   parentID,
		Type:       winfo.WorkflowType.Name,
		StartedAt:  workflow.Now(ctx),
		Status:     Running,
		OrgID:      int32(orgID),
		ResourceID: resourceID,
	}
	err := workflow.ExecuteActivity(ctx, ProcessActivityName, activityInput).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Sugar().Warnf("failed to log process: %s", err)
	}

	return &process{ctx: ctx, activityInput: activityInput}
}

func (p *process) RecordEnd(err error) {
	finishedAt := workflow.Now(p.ctx)
	p.activityInput.FinishedAt = &finishedAt
	if err != nil {
		if cadence.IsCanceledError(err) {
			p.ctx, _ = workflow.NewDisconnectedContext(p.ctx)
			p.activityInput.Status = Canceled
		} else {
			p.activityInput.Status = Failed
		}
		p.activityInput.Log = err.Error()
	} else {
		p.activityInput.Status = Finished
	}

	err = workflow.ExecuteActivity(p.ctx, ProcessActivityName, p.activityInput).Get(p.ctx, nil)
	if err != nil {
		workflow.GetLogger(p.ctx).Sugar().Warnf("failed to log process end: %s", err)
	}
}

type processEvent struct {
	ctx           workflow.Context
	activityInput ProcessEventActivityInput
}

func NewEvent(ctx workflow.Context, activityName string) Event {
	ctx = workflow.WithTaskList(ctx, "pipeline")

	winfo := workflow.GetInfo(ctx)

	activityInput := ProcessEventActivityInput{
		ProcessID: winfo.WorkflowExecution.ID,
		Type:      activityName,
		Timestamp: workflow.Now(ctx),
		Status:    Running,
	}

	err := workflow.ExecuteActivity(ctx, ProcessEventActivityName, activityInput).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Sugar().Warnf("failed to log process event: %s", err)
	}

	return &processEvent{ctx: ctx, activityInput: activityInput}
}

func (p *processEvent) RecordEnd(err error) {
	p.activityInput.Timestamp = workflow.Now(p.ctx)
	if err != nil {
		if cadence.IsCanceledError(err) {
			p.ctx, _ = workflow.NewDisconnectedContext(p.ctx)
			p.activityInput.Status = Canceled
		} else {
			p.activityInput.Status = Failed
		}
		p.activityInput.Log = err.Error()
	} else {
		p.activityInput.Status = Finished
	}

	err = workflow.ExecuteActivity(p.ctx, ProcessEventActivityName, p.activityInput).Get(p.ctx, nil)
	if err != nil {
		workflow.GetLogger(p.ctx).Sugar().Warnf("failed to log process event end: %s", err.Error())
	}
}
