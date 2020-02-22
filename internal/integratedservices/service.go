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
	"fmt"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
)

// +kit:endpoint:errorStrategy=service
// +testify:mock

// Service manages integrated services on Kubernetes clusters.
type Service interface {
	// List lists the activated integrated services and their details.
	List(ctx context.Context, clusterID uint) (services []IntegratedService, err error)

	// Details returns the details of an activated integrated service.
	Details(ctx context.Context, clusterID uint, serviceName string) (service IntegratedService, err error)

	// Activate activates a integrated service.
	Activate(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error

	// Deactivate deactivates a integrated service.
	Deactivate(ctx context.Context, clusterID uint, serviceName string) error

	// Update updates a integrated service.
	Update(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error
}

// MakeIntegratedServiceService returns a new IntegratedServiceService instance.
func MakeIntegratedServiceService(
	integratedServiceOperationDispatcher IntegratedServiceOperationDispatcher,
	integratedServiceManagerRegistry IntegratedServiceManagerRegistry,
	integratedServiceRepository IntegratedServiceRepository,
	logger common.Logger,
) IntegratedServiceService {
	return IntegratedServiceService{
		integratedServiceOperationDispatcher: integratedServiceOperationDispatcher,
		integratedServiceManagerRegistry:     integratedServiceManagerRegistry,
		integratedServiceRepository:          integratedServiceRepository,
		logger:                               logger,
	}
}

// IntegratedServiceService implements a cluster integrated service service
type IntegratedServiceService struct {
	integratedServiceOperationDispatcher IntegratedServiceOperationDispatcher
	integratedServiceManagerRegistry     IntegratedServiceManagerRegistry
	integratedServiceRepository          IntegratedServiceRepository
	logger                               common.Logger
}

// List returns non-inactive integrated services and their status.
func (s IntegratedServiceService) List(ctx context.Context, clusterID uint) ([]IntegratedService, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID})
	logger.Info("listing integrated services")

	integratedServices, err := s.integratedServiceRepository.GetIntegratedServices(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to retrieve integrated services", "clusterId", clusterID)
	}

	// only keep integrated service name and status
	for i := range integratedServices {
		integratedServices[i].Spec = nil
		integratedServices[i].Output = nil
	}

	logger.Info("integrated services successfully listed")

	return integratedServices, nil
}

// Details returns the details of an activated integrated service.
func (s IntegratedServiceService) Details(ctx context.Context, clusterID uint, integratedServiceName string) (IntegratedService, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "integrated service": integratedServiceName})
	logger.Info("processing integrated service details request")

	// TODO: check cluster ID?

	logger.Debug("retrieving integrated service manager")
	integratedServiceManager, err := s.integratedServiceManagerRegistry.GetIntegratedServiceManager(integratedServiceName)
	if err != nil {
		const msg = "failed to retrieve integrated service manager"
		logger.Debug(msg)
		return IntegratedService{}, errors.WrapIf(err, msg)
	}

	logger.Debug("retrieving integrated service from repository")
	integratedService, err := s.integratedServiceRepository.GetIntegratedService(ctx, clusterID, integratedServiceName)
	if err != nil {
		if IsIntegratedServiceNotFoundError(err) {
			return IntegratedService{
				Name:   integratedServiceName,
				Status: IntegratedServiceStatusInactive,
			}, nil
		}

		const msg = "failed to retrieve integrated service from repository"
		logger.Debug(msg)
		return integratedService, errors.WrapIf(err, msg)
	}

	logger.Debug("retrieving integrated service output")
	output, err := integratedServiceManager.GetOutput(ctx, clusterID, integratedService.Spec)
	if err != nil {
		const msg = "failed to retrieve integrated service output"
		logger.Debug(msg)
		return integratedService, errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "integrated service", integratedServiceName)
	}

	integratedService.Output = merge(integratedService.Output, output)

	logger.Info("integrated service details request processed successfully")

	return integratedService, nil
}

// Activate activates an integrated service.
func (s IntegratedServiceService) Activate(ctx context.Context, clusterID uint, integratedServiceName string, spec map[string]interface{}) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "integrated service": integratedServiceName})
	logger.Info("processing integrated service activation request")

	// TODO: check cluster ID?

	logger.Debug("retrieving integrated service manager")
	integratedServiceManager, err := s.integratedServiceManagerRegistry.GetIntegratedServiceManager(integratedServiceName)
	if err != nil {
		const msg = "failed to retrieve integrated service manager"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	if _, err := s.integratedServiceRepository.GetIntegratedService(ctx, clusterID, integratedServiceName); err == nil {
		return errors.WithStackIf(serviceAlreadyActiveError{
			ServiceName: integratedServiceName,
		})
	} else if !IsIntegratedServiceNotFoundError(err) { // unexpected error
		const msg = "failed to get integrated service from repository"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("validating integrated service specification")
	if err := integratedServiceManager.ValidateSpec(ctx, spec); err != nil {
		logger.Debug("integrated service specification validation failed")
		return InvalidIntegratedServiceSpecError{IntegratedServiceName: integratedServiceName, Problem: err.Error()}
	}

	logger.Debug("preparing integrated service specification")
	preparedSpec, err := integratedServiceManager.PrepareSpec(ctx, clusterID, spec)
	if err != nil {
		const msg = "failed to prepare integrated service specification"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("starting integrated service activation")
	if err := s.integratedServiceOperationDispatcher.DispatchApply(ctx, clusterID, integratedServiceName, preparedSpec); err != nil {
		const msg = "failed to start integrated service activation"
		logger.Debug(msg)
		return errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "integrated service", integratedServiceName)
	}

	logger.Debug("persisting integrated service")
	if err := s.integratedServiceRepository.SaveIntegratedService(ctx, clusterID, integratedServiceName, spec, IntegratedServiceStatusPending); err != nil {
		const msg = "failed to persist integrated service"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Info("integrated service activation request processed successfully")

	return nil
}

// Deactivate deactivates a integrated service.
func (s IntegratedServiceService) Deactivate(ctx context.Context, clusterID uint, integratedServiceName string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "integrated service": integratedServiceName})
	logger.Info("processing integrated service deactivation request")

	// TODO: check cluster ID?

	logger.Debug("checking integrated service name")
	if _, err := s.integratedServiceManagerRegistry.GetIntegratedServiceManager(integratedServiceName); err != nil {
		const msg = "failed to retrieve integrated service manager"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("get integrated service details")
	f, err := s.integratedServiceRepository.GetIntegratedService(ctx, clusterID, integratedServiceName)
	if err != nil {
		const msg = "failed to retrieve integrated service details"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("starting integrated service deactivation")
	if err := s.integratedServiceOperationDispatcher.DispatchDeactivate(ctx, clusterID, integratedServiceName, f.Spec); err != nil {
		const msg = "failed to start integrated service deactivation"
		logger.Debug(msg)
		return errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "integrated service", integratedServiceName)
	}

	logger.Debug("updating integrated service status")
	if err := s.integratedServiceRepository.UpdateIntegratedServiceStatus(ctx, clusterID, integratedServiceName, IntegratedServiceStatusPending); err != nil {
		if !IsIntegratedServiceNotFoundError(err) {
			const msg = "failed to update integrated service status"
			logger.Debug(msg)
			return errors.WrapIf(err, msg)
		}

		logger.Info("integrated service is already inactive")
	}

	logger.Info("integrated service deactivation request processed successfully")

	return nil
}

// Update updates a integrated service.
func (s IntegratedServiceService) Update(ctx context.Context, clusterID uint, integratedServiceName string, spec map[string]interface{}) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterID": clusterID, "integrated service": integratedServiceName})
	logger.Info("processing integrated service update request")

	// TODO: check cluster ID?

	logger.Debug("retieving integrated service manager")
	integratedServiceManager, err := s.integratedServiceManagerRegistry.GetIntegratedServiceManager(integratedServiceName)
	if err != nil {
		const msg = "failed to retrieve integrated service manager"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("validating integrated service specification")
	if err := integratedServiceManager.ValidateSpec(ctx, spec); err != nil {
		logger.Debug("integrated service specification validation failed")
		return InvalidIntegratedServiceSpecError{IntegratedServiceName: integratedServiceName, Problem: err.Error()}
	}

	logger.Debug("preparing integrated service specification")
	preparedSpec, err := integratedServiceManager.PrepareSpec(ctx, clusterID, spec)
	if err != nil {
		const msg = "failed to prepare integrated service specification"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Debug("starting integrated service update")
	if err := s.integratedServiceOperationDispatcher.DispatchApply(ctx, clusterID, integratedServiceName, preparedSpec); err != nil {
		const msg = "failed to start integrated service update"
		logger.Debug(msg)
		return errors.WrapIfWithDetails(err, msg, "clusterID", clusterID, "integrated service", integratedServiceName)
	}

	logger.Debug("persisting integrated service")
	if err := s.integratedServiceRepository.SaveIntegratedService(ctx, clusterID, integratedServiceName, spec, IntegratedServiceStatusPending); err != nil {
		const msg = "failed to persist integrated service"
		logger.Debug(msg)
		return errors.WrapIf(err, msg)
	}

	logger.Info("integrated service updated successfully")

	return nil
}

func merge(this map[string]interface{}, that map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(this)+len(that))
	for k, v := range this {
		result[k] = v
	}
	for k, v := range that {
		result[k] = v
	}
	return result
}

type serviceAlreadyActiveError struct {
	ServiceName string
}

func (e serviceAlreadyActiveError) Error() string {
	return fmt.Sprintf("Service %q is already active.", e.ServiceName)
}

func (e serviceAlreadyActiveError) Details() []interface{} {
	return []interface{}{
		"integratedServiceName", e.ServiceName,
	}
}

func (serviceAlreadyActiveError) ServiceError() bool {
	return true
}

// Conflict tells a client that this error is related to a conflicting request.
// Can be used to translate the error to status codes for example.
func (serviceAlreadyActiveError) Conflict() bool {
	return true
}
