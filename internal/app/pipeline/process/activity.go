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

	"github.com/banzaicloud/pipeline-sdk/process"
)

const ProcessLogActivityName = "process-log"

const ProcessEventActivityName = "process-event"

type ProcessLogActivity struct {
	service Service
}

func NewProcessLogActivity(service Service) ProcessLogActivity {
	return ProcessLogActivity{service: service}
}

func (a ProcessLogActivity) ExecuteProcessLog(ctx context.Context, input process.ProcessLogActivityInput) error {
	_, err := a.service.LogProcess(ctx, Process{
		Id:         input.ID,
		ParentId:   input.ParentID,
		OrgId:      input.OrgID,
		Type:       input.Type,
		Log:        input.Log,
		ResourceId: input.ResourceID,
		Status:     ProcessStatus(input.Status),
		StartedAt:  input.StartedAt,
		FinishedAt: input.FinishedAt,
	})

	return err
}

func (a ProcessLogActivity) ExecuteProcessEvent(ctx context.Context, input process.ProcessEventActivityInput) error {
	_, err := a.service.LogProcessEvent(ctx, ProcessEvent{
		ProcessId: input.ProcessID,
		Type:      input.Type,
		Log:       input.Log,
		Status:    ProcessStatus(input.Status),
		Timestamp: input.Timestamp,
	})

	return err
}
