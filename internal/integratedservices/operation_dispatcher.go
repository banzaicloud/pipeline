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

package integratedservices

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
)

// NewLocalIntegratedServiceOperationDispatcher dispatches integrated service operations via goroutines
// This dispatcher implementation should not be used in production, only for development and testing.
func NewLocalIntegratedServiceOperationDispatcher(
	jobQueueSize uint,
	integratedServiceOperatorRegistry IntegratedServiceOperatorRegistry,
	integratedServiceRepository IntegratedServiceRepository,
	logger common.Logger,
	results chan<- error,
) LocalIntegratedServiceOperationDispatcher {
	if logger == nil {
		logger = common.NoopLogger{}
	}

	jobQueue := make(chan job, jobQueueSize)

	jobProcessor := localJobProcessor{
		integratedServiceOperatorRegistry: integratedServiceOperatorRegistry,
		integratedServiceRepository:       integratedServiceRepository,
		jobQueue:                          jobQueue,
		logger:                            logger,
		results:                           results,
	}

	go jobProcessor.ProcessJobs()

	return LocalIntegratedServiceOperationDispatcher{
		jobQueue: jobQueue,
		logger:   logger,
	}
}

// LocalIntegratedServiceOperationDispatcher implements an IntegratedServiceOperationDispatcher using goroutines
type LocalIntegratedServiceOperationDispatcher struct {
	jobQueue chan<- job
	logger   common.Logger
}

// Terminate prevents the dispatcher from processing further requests
func (d LocalIntegratedServiceOperationDispatcher) Terminate() {
	close(d.jobQueue)
}

// DispatchApply dispatches an Apply request to a integrated service manager asynchronously
func (d LocalIntegratedServiceOperationDispatcher) DispatchApply(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec) error {
	d.logger.Debug("starting integrated service spec application", map[string]interface{}{
		"clusterID": clusterID,
		"spec":      spec,
	})
	select {
	case d.jobQueue <- job{
		Operation:             operationApply,
		ClusterID:             clusterID,
		IntegratedServiceName: integratedServiceName,
		Spec:                  spec,
	}:
		return nil
	default:
		return errors.New("job queue is full")
	}
}

// DispatchDeactivate dispatches a Deactivate request to a integrated service manager asynchronously
func (d LocalIntegratedServiceOperationDispatcher) DispatchDeactivate(ctx context.Context, clusterID uint, integratedServiceName string) error {
	d.logger.Debug("starting integrated service deactivation", map[string]interface{}{
		"clusterID": clusterID,
	})
	select {
	case d.jobQueue <- job{
		Operation:             operationDeactivate,
		ClusterID:             clusterID,
		IntegratedServiceName: integratedServiceName,
	}:
		return nil
	default:
		return errors.New("job queue is full")
	}
}

type localJobProcessor struct {
	integratedServiceOperatorRegistry IntegratedServiceOperatorRegistry
	integratedServiceRepository       IntegratedServiceRepository
	jobQueue                          <-chan job
	logger                            common.Logger
	results                           chan<- error
}

func (p localJobProcessor) ProcessJobs() {
	ctx := context.Background()
	logger := p.logger.WithContext(ctx)

	defer func() {
		logger.Debug("job processor terminating")
		close(p.results)
	}()

	var lastJob job
	var lastResult error
	for lastJob = range p.jobQueue {
		logger.Debug("received job", map[string]interface{}{"job": lastJob})
		lastResult = p.ProcessJob(lastJob)
		p.sendResult(lastResult)
		if lastResult != nil {
			logger.Debug("updating integrated service status")
			if err := p.integratedServiceRepository.UpdateIntegratedServiceStatus(ctx, lastJob.ClusterID, lastJob.IntegratedServiceName, IntegratedServiceStatusError); err != nil {
				logger.Error("failed to update integrated service status", map[string]interface{}{"error": err.Error()})
			}
			logger.Error(lastResult.Error())
			return
		}
	}

	switch lastJob.Operation {
	case operationApply:
		logger.Debug("updating integrated service status")
		if err := p.integratedServiceRepository.UpdateIntegratedServiceStatus(ctx, lastJob.ClusterID, lastJob.IntegratedServiceName, IntegratedServiceStatusActive); err != nil {
			logger.Error("failed to update integrated service status", map[string]interface{}{"error": err.Error()})
		}
	case operationDeactivate:
		logger.Debug("deleting integrated service")
		if err := p.integratedServiceRepository.DeleteIntegratedService(ctx, lastJob.ClusterID, lastJob.IntegratedServiceName); err != nil {
			logger.Error("failed to delete integrated service", map[string]interface{}{"error": err.Error()})
		}
	}
}

func (p localJobProcessor) sendResult(result error) {
	if p.results != nil {
		p.results <- result
	}
}

func (p localJobProcessor) ProcessJob(j job) error {
	ctx := context.Background()
	logger := p.logger.WithContext(ctx)

	logger.Debug("retrieving integrated service manager")
	integratedServiceOperator, err := p.integratedServiceOperatorRegistry.GetIntegratedServiceOperator(j.IntegratedServiceName)
	if err != nil {
		const msg = "failed to retrieve integrated service operator"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	switch j.Operation {
	case operationApply:
		logger.Debug("executing Apply operation")
		if err := integratedServiceOperator.Apply(ctx, j.ClusterID, j.Spec); err != nil {
			const msg = "failed to execute Apply operation"
			logger.Debug(msg)
			return errors.WrapIf(err, msg)
		}

	case operationDeactivate:
		logger.Debug("executing Deactivate operation")
		if err := integratedServiceOperator.Deactivate(ctx, j.ClusterID, j.Spec); err != nil {
			const msg = "failed to execute Deactivate operation"
			logger.Debug(msg)
			return errors.WrapIf(err, msg)
		}

	default:
		logger.Error("unsupported operation")
	}

	logger.Debug("processed job")
	return nil
}

type job struct {
	Operation             operation
	ClusterID             uint
	IntegratedServiceName string
	Spec                  IntegratedServiceSpec
}

type operation string

const (
	operationApply      operation = "apply"
	operationDeactivate operation = "deactivate"
)
