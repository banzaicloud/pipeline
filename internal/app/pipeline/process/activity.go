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
	"context"
	"time"
)

const ProcessActivityName = "process"

const ProcessEventActivityName = "process-event"

type Status string

const (
	Running  Status = "running"
	Failed   Status = "failed"
	Finished Status = "finished"
)

type ProcessActivityInput struct {
	ID           string
	ParentID     string
	OrgID        int32
	Type         string
	Log          string
	ResourceID   string
	ResourceType string
	Status       Status
	StartedAt    time.Time
	FinishedAt   *time.Time
}

type ProcessEventActivityInput struct {
	ProcessID string
	Type      string
	Log       string
	Status    Status
	Timestamp time.Time
}

type ProcessActivity struct {
	service Service
}

func NewProcessActivity(service Service) ProcessActivity {
	return ProcessActivity{service: service}
}

func (a ProcessActivity) ExecuteProcess(ctx context.Context, input ProcessActivityInput) error {
	_, err := a.service.LogProcess(ctx, Process{
		Id:           input.ID,
		ParentId:     input.ParentID,
		OrgId:        input.OrgID,
		Type:         input.Type,
		Log:          input.Log,
		ResourceId:   input.ResourceID,
		ResourceType: input.ResourceType,
		Status:       ProcessStatus(input.Status),
		StartedAt:    input.StartedAt,
		FinishedAt:   input.FinishedAt,
	})

	return err
}

func (a ProcessActivity) ExecuteProcessEvent(ctx context.Context, input ProcessEventActivityInput) error {
	_, err := a.service.LogProcessEvent(ctx, ProcessEvent{
		ProcessId: input.ProcessID,
		Type:      input.Type,
		Log:       input.Log,
		Status:    ProcessStatus(input.Status),
		Timestamp: input.Timestamp,
	})

	return err
}
