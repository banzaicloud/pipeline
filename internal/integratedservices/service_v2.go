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

package integratedservices

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
)

// ISServiceV2 integrated service service implementation - V2
type ISServiceV2 struct {
	managerRegistry   IntegratedServiceManagerRegistry
	dispatcher        IntegratedServiceOperationDispatcher
	repository        IntegratedServiceRepository
	serviceNameMapper ServiceNameMapper
	logger            common.Logger
	metrics           ApiMetrics
}

// NewISServiceV2 creates a new service instance using the provided collaborators
func NewISServiceV2(
	integratedServiceManagerRegistry IntegratedServiceManagerRegistry,
	integratedServiceOperationDispatcher IntegratedServiceOperationDispatcher,
	repository IntegratedServiceRepository,
	serviceNameMapper ServiceNameMapper,
	logger common.Logger,
	metrics ApiMetrics,
) *ISServiceV2 {
	return &ISServiceV2{
		managerRegistry:   integratedServiceManagerRegistry,
		dispatcher:        integratedServiceOperationDispatcher,
		repository:        repository,
		serviceNameMapper: serviceNameMapper,
		logger:            logger,
		metrics:           metrics,
	}
}

// Activate initiates the activation of an integrated service
func (i ISServiceV2) Activate(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error {
	timer := i.metrics.RequestTimer(clusterID, serviceName, "activate")
	defer timer.ObserveDuration()

	errorCounter := i.metrics.ErrorCounter(clusterID, serviceName, "activate")

	manager, err := i.managerRegistry.GetIntegratedServiceManager(serviceName)
	if err != nil {
		errorCounter.Increment("unsupported")
		return errors.WrapIf(err, "unsupported integrated service")
	}

	if err := manager.ValidateSpec(ctx, spec); err != nil {
		errorCounter.Increment("validate_spec")
		return InvalidIntegratedServiceSpecError{IntegratedServiceName: serviceName, Problem: err.Error()}
	}

	preparedSpec, err := manager.PrepareSpec(ctx, clusterID, spec)
	if err != nil {
		errorCounter.Increment("prepare_spec")
		return errors.WrapIf(err, "failed to prepare the integrated service specification")
	}

	if err := i.dispatcher.DispatchApply(ctx, clusterID, serviceName, preparedSpec); err != nil {
		errorCounter.Increment("apply")
		return errors.WrapIfWithDetails(err, "failed to dispatch the apply operation", "clusterID", clusterID, "integrated service", serviceName)
	}

	return nil
}

func (i ISServiceV2) List(ctx context.Context, clusterID uint) ([]IntegratedService, error) {
	timer := i.metrics.RequestTimer(clusterID, "", "list")
	defer timer.ObserveDuration()

	errorCounter := i.metrics.ErrorCounter(clusterID, "", "list")

	integratedServices, err := i.repository.GetIntegratedServices(ctx, clusterID)
	if err != nil {
		errorCounter.Increment("repository_list")
		return nil, errors.WrapIfWithDetails(err, "failed to retrieve integrated services", "clusterId", clusterID)
	}

	// only keep integrated service name and status
	for j := range integratedServices {
		integratedServices[j].Name = i.serviceNameMapper.MapServiceName(integratedServices[j].Name)
		integratedServices[j].Spec = nil
		integratedServices[j].Output = nil
	}

	return integratedServices, nil
}

func (i ISServiceV2) Details(ctx context.Context, clusterID uint, serviceName string) (IntegratedService, error) {
	timer := i.metrics.RequestTimer(clusterID, serviceName, "details")
	defer timer.ObserveDuration()

	errorCounter := i.metrics.ErrorCounter(clusterID, serviceName, "details")

	manager, err := i.managerRegistry.GetIntegratedServiceManager(serviceName)
	if err != nil {
		errorCounter.Increment("unsupported")
		return IntegratedService{}, errors.WrapIf(err, "failed to get integrated service manager")
	}

	integratedService, err := i.repository.GetIntegratedService(ctx, clusterID, i.serviceNameMapper.MapServiceName(serviceName))
	if err != nil {
		if IsIntegratedServiceNotFoundError(err) {
			errorCounter.Increment("notfound")
			return IntegratedService{
				Name:   serviceName,
				Status: IntegratedServiceStatusInactive,
			}, nil
		}

		errorCounter.Increment("repo_get")
		return integratedService, errors.WrapIf(err, "failed to retrieve integrated service")
	}

	output, err := manager.GetOutput(ctx, clusterID, integratedService.Spec)
	if err != nil {
		errorCounter.Increment("output")
		return integratedService, errors.WrapIfWithDetails(err, "failed to retrieve integrated service output", "clusterID", clusterID, "integrated service", serviceName)
	}

	integratedService.Output = merge(integratedService.Output, output)

	return integratedService, nil
}

func (i ISServiceV2) Deactivate(ctx context.Context, clusterID uint, serviceName string) error {
	timer := i.metrics.RequestTimer(clusterID, serviceName, "deactivate")
	defer timer.ObserveDuration()

	errorCounter := i.metrics.ErrorCounter(clusterID, serviceName, "deactivate")

	if _, err := i.managerRegistry.GetIntegratedServiceManager(serviceName); err != nil {
		errorCounter.Increment("unsupported")
		return errors.WrapIf(err, "failed to retrieve integrated service manager")
	}

	f, err := i.repository.GetIntegratedService(ctx, clusterID, i.serviceNameMapper.MapServiceName(serviceName))
	if err != nil {
		errorCounter.Increment("repo_get")
		return errors.WrapIf(err, "failed to retrieve integrated service details")
	}

	if err := i.dispatcher.DispatchDeactivate(ctx, clusterID, serviceName, f.Spec); err != nil {
		errorCounter.Increment("deactivate")
		return errors.WrapIf(err, "failed to start integrated service deactivation")
	}

	return nil
}

func (i ISServiceV2) Update(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error {
	timer := i.metrics.RequestTimer(clusterID, serviceName, "update")
	defer timer.ObserveDuration()

	errorCounter := i.metrics.ErrorCounter(clusterID, serviceName, "update")

	integratedServiceManager, err := i.managerRegistry.GetIntegratedServiceManager(serviceName)
	if err != nil {
		errorCounter.Increment("unsupported")
		return errors.WrapIf(err, "failed to retrieve integrated service manager")
	}

	if err := integratedServiceManager.ValidateSpec(ctx, spec); err != nil {
		errorCounter.Increment("validate_spec")
		return InvalidIntegratedServiceSpecError{IntegratedServiceName: serviceName, Problem: err.Error()}
	}

	preparedSpec, err := integratedServiceManager.PrepareSpec(ctx, clusterID, spec)
	if err != nil {
		errorCounter.Increment("prepare_spec")
		return errors.WrapIf(err, "failed to prepare integrated service specification")
	}

	if err := i.dispatcher.DispatchApply(ctx, clusterID, serviceName, preparedSpec); err != nil {
		errorCounter.Increment("apply")
		return errors.WrapIf(err, "failed to start integrated service update")
	}

	return nil
}
