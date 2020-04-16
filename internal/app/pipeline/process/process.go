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

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/src/auth"
)

// Process represents an pipeline process.
type Process = pipeline.Process

// ProcessEvent represents an pipeline process event.
type ProcessEvent = pipeline.ProcessEvent

// ProcessStatus represents an pipeline process/event status.
type ProcessStatus = pipeline.ProcessStatus

//go:generate mga gen mockery --name Service --inpkg
// +kit:endpoint:errorStrategy=service

// Service provides access to pipeline processes.
type Service interface {
	// LogProcess create a process entry
	LogProcess(ctx context.Context, proc Process) (process Process, err error)

	// LogProcessEvent create a process event
	LogProcessEvent(ctx context.Context, proc ProcessEvent) (processEvent ProcessEvent, err error)

	// ListProcesses lists access processes visible for a user.
	ListProcesses(ctx context.Context, query Process) (processes []Process, err error)

	// GetProcess returns a single process.
	GetProcess(ctx context.Context, org auth.Organization, id string) (process Process, err error)
}

// NewService returns a new Service.
func NewService(store Store) Service {
	return service{store: store}
}

type service struct {
	store Store
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

func (s service) GetProcess(ctx context.Context, org auth.Organization, id string) (Process, error) {
	return s.store.GetProcess(ctx, id)
}

func (s service) LogProcess(ctx context.Context, p Process) (Process, error) {
	return p, s.store.LogProcess(ctx, p)
}

func (s service) LogProcessEvent(ctx context.Context, p ProcessEvent) (ProcessEvent, error) {
	return p, s.store.LogProcessEvent(ctx, p)
}
