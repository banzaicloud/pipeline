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
		return errors.WrapIfWithDetails(err, "failed to dispatch the apply operation", "clusterID", clusterID, "integrated service", serviceName)
	}

	return nil
}

func (i ISServiceV2) List(ctx context.Context, clusterID uint) ([]IntegratedService, error) {
	integratedServices, err := i.repository.GetIntegratedServices(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to retrieve integrated services", "clusterId", clusterID)
	}

	// only keep integrated service name and status
	for i := range integratedServices {
		integratedServices[i].Spec = nil
		integratedServices[i].Output = nil
	}

	return integratedServices, nil
}

func (i ISServiceV2) Details(ctx context.Context, clusterID uint, serviceName string) (IntegratedService, error) {
	manager, err := i.managerRegistry.GetIntegratedServiceManager(serviceName)
	if err != nil {
		return IntegratedService{}, errors.WrapIf(err, "failed to get integrated service manager")
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

	output, err := manager.GetOutput(ctx, clusterID, integratedService.Spec)
	if err != nil {
		return integratedService, errors.WrapIfWithDetails(err, "failed to retrieve integrated service output", "clusterID", clusterID, "integrated service", serviceName)
	}

	integratedService.Output = merge(integratedService.Output, output)

	return integratedService, nil
}

func (i ISServiceV2) Deactivate(ctx context.Context, clusterID uint, serviceName string) error {
	// TODO implement me!
	return errors.NewWithDetails("Operation not, yet implemented!", "clusterID", clusterID)
}

func (i ISServiceV2) Update(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error {
	// TODO implement me!
	return errors.NewWithDetails("Operation not, yet implemented!", "clusterID", clusterID)
}
