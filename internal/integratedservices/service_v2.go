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
	managerRegistry IntegratedServiceManagerRegistry
	dispatcher      IntegratedServiceOperationDispatcher
	repository      IntegratedServiceRepository
	logger          common.Logger
}

// NewISServiceV2 creates a new service instance using the provided collaborators
func NewISServiceV2(
	integratedServiceManagerRegistry IntegratedServiceManagerRegistry,
	integratedServiceOperationDispatcher IntegratedServiceOperationDispatcher,
	repository IntegratedServiceRepository,
	logger common.Logger,
) *ISServiceV2 {
	return &ISServiceV2{
		managerRegistry: integratedServiceManagerRegistry,
		dispatcher:      integratedServiceOperationDispatcher,
		repository:      repository,
		logger:          logger,
	}
}

// Activate initiates the activation of an integrated service
func (i ISServiceV2) Activate(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error {
	manager, err := i.managerRegistry.GetIntegratedServiceManager(serviceName)
	if err != nil {
		return errors.WrapIf(err, "unsupported integrated service")
	}

	if err := manager.ValidateSpec(ctx, spec); err != nil {
		return InvalidIntegratedServiceSpecError{IntegratedServiceName: serviceName, Problem: err.Error()}
	}

	preparedSpec, err := manager.PrepareSpec(ctx, clusterID, spec)
	if err != nil {
		return errors.WrapIf(err, "failed to prepare the integrated service specification")
	}

	if err := i.dispatcher.DispatchApply(ctx, clusterID, serviceName, preparedSpec); err != nil {
		return errors.WrapIfWithDetails(err, "failed to dispatch the apply operation", "clusterID", clusterID, "serviceName", serviceName)
	}

	return nil
}

func (i ISServiceV2) List(ctx context.Context, clusterID uint) ([]IntegratedService, error) {
	integratedServices, err := i.repository.GetIntegratedServices(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to retrieve integrated services", "clusterId", clusterID)
	}

	// Some services may be disabled, we only want enabled ones
	supportedServiceNames := i.managerRegistry.GetIntegratedServiceNames()

	servicesToReturn := make([]IntegratedService, len(supportedServiceNames))
	for j, serviceName := range supportedServiceNames {
		status := IntegratedServiceStatusInactive
		// Take the status of existing integrated service instance if it exists
		for _, service := range integratedServices {
			if service.Name == serviceName {
				status = service.Status
				break
			}
		}
		// Check whether there is an active workflow running for the service
		dispatched, err := i.dispatcher.IsBeingDispatched(ctx, clusterID, serviceName)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to retrieve integrated service dispatched status")
		}
		// Services where a job is currently being dispatched should be treated as pending
		if dispatched {
			status = IntegratedServiceStatusPending
		}
		servicesToReturn[j] = IntegratedService{
			Name:   serviceName,
			Status: status,
		}
	}

	return servicesToReturn, nil
}

func (i ISServiceV2) Details(ctx context.Context, clusterID uint, serviceName string) (IntegratedService, error) {
	isDispatched, err := i.dispatcher.IsBeingDispatched(ctx, clusterID, serviceName)
	if err != nil {
		return IntegratedService{}, errors.WrapIfWithDetails(err, "failed to check workflow",
			"clusterID", clusterID, "serviceName", serviceName)
	}
	if isDispatched {
		return IntegratedService{
			Name:   serviceName,
			Status: IntegratedServiceStatusPending,
		}, nil
	}

	integratedService, err := i.repository.GetIntegratedService(ctx, clusterID, serviceName)
	if err != nil {
		if IsIntegratedServiceNotFoundError(err) {
			return IntegratedService{
				Name:   serviceName,
				Status: IntegratedServiceStatusInactive,
			}, nil
		}

		return integratedService, errors.WrapIf(err, "failed to retrieve integrated service")
	}

	return integratedService, nil
}

func (i ISServiceV2) Deactivate(ctx context.Context, clusterID uint, serviceName string) error {
	if err := i.checkManagedByPipeline(ctx, clusterID, serviceName); err != nil {
		return errors.WrapIf(err, "service is not managed by pipeline")
	}

	if _, err := i.managerRegistry.GetIntegratedServiceManager(serviceName); err != nil {
		return errors.WrapIf(err, "failed to retrieve integrated service manager")
	}

	f, err := i.repository.GetIntegratedService(ctx, clusterID, serviceName)
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve integrated service details")
	}

	if err := i.dispatcher.DispatchDeactivate(ctx, clusterID, serviceName, f.Spec); err != nil {
		return errors.WrapIf(err, "failed to start integrated service deactivation")
	}

	return nil
}

func (i ISServiceV2) Update(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error {
	if err := i.checkManagedByPipeline(ctx, clusterID, serviceName); err != nil {
		return errors.WrapIf(err, "service is not managed by pipeline")
	}

	integratedServiceManager, err := i.managerRegistry.GetIntegratedServiceManager(serviceName)
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve integrated service manager")
	}

	if err := integratedServiceManager.ValidateSpec(ctx, spec); err != nil {
		return InvalidIntegratedServiceSpecError{IntegratedServiceName: serviceName, Problem: err.Error()}
	}

	preparedSpec, err := integratedServiceManager.PrepareSpec(ctx, clusterID, spec)
	if err != nil {
		return errors.WrapIf(err, "failed to prepare integrated service specification")
	}

	if err := i.dispatcher.DispatchApply(ctx, clusterID, serviceName, preparedSpec); err != nil {
		return errors.WrapIf(err, "failed to start integrated service update")
	}

	return nil
}

func (i ISServiceV2) checkManagedByPipeline(ctx context.Context, clusterID uint, serviceName string) error {
	integratedService, err := i.repository.GetIntegratedService(ctx, clusterID, serviceName)
	if err != nil {
		if IsIntegratedServiceNotFoundError(err) {
			return nil
		}
		return errors.WrapIf(err, "failed to retrieve the integrated service")
	}

	managedBy, ok := integratedService.Output["managed-by"]
	if !ok {
		// the managed-by flag is not set
		return NotManagedIntegratedServiceError{serviceName}
	}
	if managedBy != "pipeline" {
		return NotManagedIntegratedServiceError{serviceName}
	}

	return nil
}
