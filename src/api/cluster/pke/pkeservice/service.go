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

package pkeservice

import (
	"context"
	"fmt"
	"path"
	"time"

	"emperror.dev/errors"
	"github.com/gofrs/uuid"
	"logur.dev/logur"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/process"
	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/sdk/brn"
)

type (
	Cluster       = cluster.Cluster
	Process       = process.Process
	ProcessEvent  = process.ProcessEvent
	ProcessStatus = process.ProcessStatus
)

// +testify:mock:testOnly=true
type logger logur.Logger

// serviceError is an error that can be shown to the user and be marshalled (for cadence)
type serviceError struct {
	error
	Message string
}

func (*serviceError) ServiceError() bool {
	return true
}

func wrapServiceError(err error) serviceError {
	return serviceError{err, err.Error()}
}

func newServiceError(text string, args ...interface{}) serviceError {
	return wrapServiceError(errors.Errorf(text, args...))
}

type NodeStatus struct {
	// name of node
	Name string

	// name of nodepool
	NodePool string

	// ip address of node (where the other nodes can reach it)
	Ip string

	// detailed description about the current bootstrapping status (including the cause of the failure)
	Message string

	// the current phase of the bootstrap process
	Phase string

	// if this is the final status report, that describes the conclusion of the whole process
	Final bool

	Status ProcessStatus

	// exact time of event
	Timestamp time.Time

	// ID of the process registered earlier (register new process if empty)
	ProcessID string
}

const (
	Running  ProcessStatus = "running"
	Failed   ProcessStatus = "failed"
	Finished ProcessStatus = "finished"
	Canceled ProcessStatus = "canceled"
)

// +kit:endpoint:errorStrategy=service
// +testify:mock

// Service provides an interface to PKE specific operations (i.e. [currently some of] those called by the pke installer)
type Service interface {
	// RegisterNodeStatus registers status reported by a node
	RegisterNodeStatus(ctx context.Context, clusterIdentifier cluster.Identifier, nodeStatus NodeStatus) (resp RegisterNodeStatusResponse, err error)
}

type RegisterNodeStatusResponse struct {
	ProcessID string
}

type service struct {
	clusters    Store
	processes   processService
	idGenerator idGenerator
	logger      logger
}

// +testify:mock:testOnly=true
type idGenerator interface {
	New() string
}

type uuidGenerator struct{}

func (uuidGenerator) New() string {
	return uuid.Must(uuid.NewV4()).String()
}

func nodeBRN(organizationID uint, clusterID uint, hostname string) brn.ResourceName {
	id := path.Join(fmt.Sprint(clusterID), hostname)
	return brn.New(organizationID, brn.NodeResourceType, id)
}

func clusterBRN(organizationID uint, clusterID uint) brn.ResourceName {
	return brn.New(organizationID, brn.ClusterResourceType, fmt.Sprint(clusterID))
}

func brnProcess(p Process, brn brn.ResourceName) Process {
	p.ResourceId = brn.ResourceID
	p.OrgId = int32(brn.OrganizationID)
	p.ResourceType = brn.ResourceType
	return p
}

func (s service) RegisterNodeStatus(ctx context.Context, clusterIdentifier cluster.Identifier, nodeStatus NodeStatus) (resp RegisterNodeStatusResponse, err error) {
	brn := nodeBRN(clusterIdentifier.OrganizationID, clusterIdentifier.ClusterID, nodeStatus.Name)

	s.logger.Info("node status update", map[string]interface{}{
		"clusterID":  clusterIdentifier.ClusterID,
		"nodeName":   nodeStatus.Name,
		"nodeIP":     nodeStatus.Ip,
		"nodePool":   nodeStatus.NodePool,
		"remoteTime": nodeStatus.Timestamp,
		"phase":      nodeStatus.Phase,
		"message":    nodeStatus.Message,
	})

	clusterBRN := clusterBRN(clusterIdentifier.OrganizationID, clusterIdentifier.ClusterID)
	clusterProcesses, err := s.processes.ListProcesses(ctx, brnProcess(Process{Status: pipeline.RUNNING}, clusterBRN))
	if err != nil {
		return resp, errors.WrapIf(err, "failed to list running cluster processes")
	}

	proc := brnProcess(process.Process{
		Id:     nodeStatus.ProcessID,
		Type:   "pke-bootstrap",
		Log:    fmt.Sprintf("%s: %s", nodeStatus.Phase, nodeStatus.Message),
		Status: Running,
	}, brn)

	if proc.Id == "" {
		proc.Id = s.idGenerator.New()
		proc.StartedAt = nodeStatus.Timestamp

		for _, cp := range clusterProcesses {
			proc.ParentId = cp.Id
		}
	} else {
		existing, err := s.processes.GetProcess(ctx, proc.Id)
		if err != nil {
			return resp, wrapServiceError(errors.WrapIf(err, "failed to get existing process"))
		}

		if proc.OrgId != existing.OrgId || proc.ResourceId != existing.ResourceId || proc.ResourceType != existing.ResourceType || proc.Type != existing.Type {
			return resp, newServiceError("invalid process ID")
		}
		proc = existing
	}

	if nodeStatus.Final {
		proc.Status = nodeStatus.Status
		proc.FinishedAt = &nodeStatus.Timestamp
		proc.Log = nodeStatus.Message
	}

	proc, err = s.processes.LogProcess(ctx, proc)
	if err != nil {
		return resp, errors.WrapIf(err, "failed to log process")
	}

	event := pipeline.ProcessEvent{
		ProcessId: proc.Id,
		Type:      fmt.Sprintf("pke-%s", nodeStatus.Phase),
		Log:       nodeStatus.Message,
		Status:    nodeStatus.Status,
		Timestamp: nodeStatus.Timestamp,
	}
	_, err = s.processes.LogProcessEvent(ctx, event)
	if err != nil {
		return resp, err
	}

	resp.ProcessID = proc.Id

	if nodeStatus.Final && nodeStatus.Status == Failed {
		signalValue := newServiceError("%s failed: %s", nodeStatus.Name, nodeStatus.Message)
		for _, cp := range clusterProcesses {
			// send signals with best effort
			_ = s.processes.SignalProcess(ctx, cp.Id, "node-bootstrap-failed", signalValue)
		}
	}

	return resp, err
}

// Store allows looking up clusters form persistent storage
type Store interface {
	// GetCluster returns a generic Cluster.
	// Returns an error with the NotFound behavior when the cluster cannot be found.
	GetCluster(ctx context.Context, id uint) (Cluster, error)
}

// processService provides access to pipeline processes.
// +testify:mock:testOnly=true
type processService interface {
	// LogProcess create a process entry
	LogProcess(ctx context.Context, proc Process) (process Process, err error)

	// LogProcessEvent create a process event
	LogProcessEvent(ctx context.Context, proc ProcessEvent) (processEvent ProcessEvent, err error)

	// ListProcesses lists access processes visible for a user.
	ListProcesses(ctx context.Context, query Process) (processes []Process, err error)

	// GetProcess returns a single process.
	GetProcess(ctx context.Context, id string) (process Process, err error)

	// CancelProcess cancels a single process.
	// CancelProcess(ctx context.Context, id string) (err error)

	// SignalProcess sends a signal to a single process.
	SignalProcess(ctx context.Context, id string, signal string, value interface{}) (err error)
}

// NewService returns a new Service instance
func NewService(
	clusters cluster.Store,
	processes processService,
	logger logger,
	idGenerator idGenerator,
) Service {
	if idGenerator == nil {
		idGenerator = new(uuidGenerator)
	}
	return service{
		clusters:    clusters,
		processes:   processes,
		logger:      logger,
		idGenerator: idGenerator,
	}
}
