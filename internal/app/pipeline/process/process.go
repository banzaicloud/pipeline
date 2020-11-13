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

	"emperror.dev/errors"
	"go.uber.org/cadence/.gen/go/shared"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
)

// Process represents an pipeline process.
type Process = pipeline.Process

// ProcessEvent represents an pipeline process event.
type ProcessEvent = pipeline.ProcessEvent

// ProcessStatus represents an pipeline process/event status.
type ProcessStatus = pipeline.ProcessStatus

// Service provides access to pipeline processes.
type Service interface {
	// LogProcess create a process entry
	LogProcess(ctx context.Context, proc Process) (process Process, err error)

	// LogProcessEvent create a process event
	LogProcessEvent(ctx context.Context, proc ProcessEvent) (processEvent ProcessEvent, err error)

	// ListProcesses lists access processes visible for a user.
	ListProcesses(ctx context.Context, query Process) (processes []Process, err error)

	// GetProcess returns a single process.
	GetProcess(ctx context.Context, id string) (process Process, err error)
}

// +kit:endpoint:errorStrategy=service
// +testify:mock

type WorkflowService interface {
	Service

	// CancelProcess cancels a single process.
	CancelProcess(ctx context.Context, id string) (err error)

	// SignalProcess sends a signal to a single process.
	SignalProcess(ctx context.Context, id string, signal string, value interface{}) (err error)
}

// NewService returns a new Service.
func NewService(store Store) Service {
	return service{store: store}
}

// NewWorkflowService returns a new WorkflowService.
func NewWorkflowService(store Store, workflowClient workflowClient) WorkflowService {
	return service{store: store, workflowClient: workflowClient}
}

type service struct {
	store          Store
	workflowClient workflowClient
}

// Store persists access processes in a persistent store.
type Store interface {
	// ListProcesses lists the process in the for a given organization.
	ListProcesses(ctx context.Context, query Process) ([]Process, error)

	// LogProcess adds a process entry.
	LogProcess(ctx context.Context, p Process) error

	// GetProcess gets a process entry.
	GetProcess(ctx context.Context, id string) (Process, error)

	// LogProcessEvent adds a process event to a process.
	LogProcessEvent(ctx context.Context, p ProcessEvent) error
}

// NotFoundError is returned if a process cannot be found.
type NotFoundError struct {
	ID string
}

// Error implements the error interface.
func (NotFoundError) Error() string {
	return "process not found"
}

// Details returns error details.
func (e NotFoundError) Details() []interface{} {
	return []interface{}{"processId", e.ID}
}

// NotFound tells a client that this error is related to a resource being not found.
// Can be used to translate the error to eg. status code.
func (NotFoundError) NotFound() bool {
	return true
}

// ServiceError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
func (NotFoundError) ServiceError() bool {
	return true
}

func (s service) ListProcesses(ctx context.Context, query Process) ([]Process, error) {
	return s.store.ListProcesses(ctx, query)
}

func (s service) GetProcess(ctx context.Context, id string) (Process, error) {
	return s.store.GetProcess(ctx, id)
}

func (s service) LogProcess(ctx context.Context, p Process) (Process, error) {
	return p, s.store.LogProcess(ctx, p)
}

func (s service) LogProcessEvent(ctx context.Context, p ProcessEvent) (ProcessEvent, error) {
	return p, s.store.LogProcessEvent(ctx, p)
}

type workflowClient interface {
	CancelWorkflow(ctx context.Context, workflowID string, runID string) error
	SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error
}

func (s service) CancelProcess(ctx context.Context, id string) error {
	if s.workflowClient == nil {
		return errors.New("workflow client not available")
	}

	err := s.workflowClient.CancelWorkflow(ctx, id, "")
	if _, ok := err.(*shared.EntityNotExistsError); ok {
		return NotFoundError{ID: id}
	}

	return err
}

func (s service) SignalProcess(ctx context.Context, id string, signal string, value interface{}) error {
	if s.workflowClient == nil {
		return errors.New("workflow client not available")
	}

	err := s.workflowClient.SignalWorkflow(ctx, id, "", signal, value)
	if _, ok := err.(*shared.EntityNotExistsError); ok {
		return NotFoundError{ID: id}
	}
	return err
}
