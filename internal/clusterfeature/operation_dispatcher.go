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

package clusterfeature

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
)

// NewLocalFeatureOperationDispatcher dispatches feature operations via goroutines
// This dispatcher implementation should not be used in production, only for development and testing.
func NewLocalFeatureOperationDispatcher(
	jobQueueSize uint,
	featureOperatorRegistry FeatureOperatorRegistry,
	featureRepository FeatureRepository,
	logger common.Logger,
	results chan<- error,
) LocalFeatureOperationDispatcher {
	if logger == nil {
		logger = commonadapter.NewNoopLogger()
	}

	jobQueue := make(chan job, jobQueueSize)

	jobProcessor := localJobProcessor{
		featureOperatorRegistry: featureOperatorRegistry,
		featureRepository:       featureRepository,
		jobQueue:                jobQueue,
		logger:                  logger,
		results:                 results,
	}

	go jobProcessor.ProcessJobs()

	return LocalFeatureOperationDispatcher{
		jobQueue: jobQueue,
		logger:   logger,
	}
}

// LocalFeatureOperationDispatcher implements an FeatureOperationDispatcher using goroutines
type LocalFeatureOperationDispatcher struct {
	jobQueue chan<- job
	logger   common.Logger
}

// Terminate prevents the dispatcher from processing further requests
func (d LocalFeatureOperationDispatcher) Terminate() {
	close(d.jobQueue)
}

// DispatchApply dispatches an Apply request to a feature manager asynchronously
func (d LocalFeatureOperationDispatcher) DispatchApply(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error {
	d.logger.Debug("starting feature spec application", map[string]interface{}{
		"clusterID": clusterID,
		"spec":      spec,
	})
	select {
	case d.jobQueue <- job{
		Operation:   operationApply,
		ClusterID:   clusterID,
		FeatureName: featureName,
		Spec:        spec,
	}:
		return nil
	default:
		return errors.New("job queue is full")
	}
}

// DispatchDeactivate dispatches a Deactivate request to a feature manager asynchronously
func (d LocalFeatureOperationDispatcher) DispatchDeactivate(ctx context.Context, clusterID uint, featureName string) error {
	d.logger.Debug("starting feature deactivation", map[string]interface{}{
		"clusterID": clusterID,
	})
	select {
	case d.jobQueue <- job{
		Operation:   operationDeactivate,
		ClusterID:   clusterID,
		FeatureName: featureName,
	}:
		return nil
	default:
		return errors.New("job queue is full")
	}
}

type localJobProcessor struct {
	featureOperatorRegistry FeatureOperatorRegistry
	featureRepository       FeatureRepository
	jobQueue                <-chan job
	logger                  common.Logger
	results                 chan<- error
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
			logger.Debug("updating feature status")
			if err := p.featureRepository.UpdateFeatureStatus(ctx, lastJob.ClusterID, lastJob.FeatureName, FeatureStatusError); err != nil {
				logger.Error("failed to update feature status", map[string]interface{}{"error": err.Error()})
			}
			logger.Error(lastResult.Error())
			return
		}
	}

	switch lastJob.Operation {
	case operationApply:
		logger.Debug("updating feature status")
		if err := p.featureRepository.UpdateFeatureStatus(ctx, lastJob.ClusterID, lastJob.FeatureName, FeatureStatusActive); err != nil {
			logger.Error("failed to update feature status", map[string]interface{}{"error": err.Error()})
		}
	case operationDeactivate:
		logger.Debug("deleting feature")
		if err := p.featureRepository.DeleteFeature(ctx, lastJob.ClusterID, lastJob.FeatureName); err != nil {
			logger.Error("failed to delete feature", map[string]interface{}{"error": err.Error()})
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

	logger.Debug("retrieving feature manager")
	featureOperator, err := p.featureOperatorRegistry.GetFeatureOperator(j.FeatureName)
	if err != nil {
		const msg = "failed to retrieve feature operator"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	switch j.Operation {
	case operationApply:
		logger.Debug("executing Apply operation")
		if err := featureOperator.Apply(ctx, j.ClusterID, j.Spec); err != nil {
			const msg = "failed to execute Apply operation"
			logger.Debug(msg)
			return errors.WrapIf(err, msg)
		}

	case operationDeactivate:
		logger.Debug("executing Deactivate operation")
		if err := featureOperator.Deactivate(ctx, j.ClusterID); err != nil {
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
	Operation   operation
	ClusterID   uint
	FeatureName string
	Spec        FeatureSpec
}

type operation string

const (
	operationApply      operation = "apply"
	operationDeactivate operation = "deactivate"
)
