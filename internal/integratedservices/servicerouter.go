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
)

// serviceRouter routes api calls to the appropriate integrated service version. Its main role is
// to make integrated service versions transparent to clients
// Generally service version 2 is preferred, but if an active service with v1 exists then it must be used until the service gets deactivated. After deactivation v2 is preferred for these service as well.
type serviceRouter struct {
	serviceV1 Service
	serviceV2 Service

	log Logger
}

// NewServiceRouter creates a new service router instance with the passed in service implementations
func NewServiceRouter(serviceV1 Service, serviceV2 Service, log Logger) Service {
	return serviceRouter{
		serviceV1: serviceV1,
		serviceV2: serviceV2,

		log: log,
	}
}

// List calls both service versions and merges results
func (s serviceRouter) List(ctx context.Context, clusterID uint) ([]IntegratedService, error) {
	issV1, err := s.serviceV1.List(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve integrated services - V1")
	}

	issV2, err := s.serviceV2.List(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve integrated services - V2")
	}

	return s.filterDuplicates(issV1, issV2)
}

// Details retrieves the service from the service v1 if not found retrieves it from v2
// Note: an Integrated Service can only be managed by one of the service versions
func (s serviceRouter) Details(ctx context.Context, clusterID uint, serviceName string) (IntegratedService, error) {
	if detailsV1, err := s.serviceV1.Details(ctx, clusterID, serviceName); err == nil {
		// return the legacy integrated service details
		return detailsV1, nil
	} else if !IsIntegratedServiceNotFoundError(err) {
		return IntegratedService{}, errors.Wrapf(err, "failed to retrieve legacy integrated service details")
	} // ignore the not found error, proceed to the new implementation

	// delegate to the new version of the service
	return s.serviceV2.Details(ctx, clusterID, serviceName)
}

// Activate delegates the activation request to the appropriate service version
// New services are always activated with the version 2 service
func (s serviceRouter) Activate(ctx context.Context, clusterID uint, serviceName string, spec IntegratedServiceSpec) error {
	if _, err := s.serviceV1.Details(ctx, clusterID, serviceName); err == nil {
		// if found on the legacy service, delegate to it
		return s.serviceV1.Activate(ctx, clusterID, serviceName, spec)
	} else if !IsIntegratedServiceNotFoundError(err) {
		return errors.WrapIf(err, "failed to retrieve integrated service from the legacy service")
	}

	// delegate to the new implementation
	return s.serviceV2.Activate(ctx, clusterID, serviceName, spec)
}

func (s serviceRouter) Deactivate(ctx context.Context, clusterID uint, serviceName string) error {
	if _, err := s.serviceV1.Details(ctx, clusterID, serviceName); err == nil {
		// if found on the legacy service, delegate to it
		return s.serviceV1.Deactivate(ctx, clusterID, serviceName)
	} else if !IsIntegratedServiceNotFoundError(err) {
		return errors.WrapIf(err, "failed to retrieve integrated service from the legacy service")
	}

	// delegate to the new implementation
	return s.serviceV2.Deactivate(ctx, clusterID, serviceName)
}

func (s serviceRouter) Update(ctx context.Context, clusterID uint, serviceName string, spec IntegratedServiceSpec) error {
	if _, err := s.serviceV1.Details(ctx, clusterID, serviceName); err == nil {
		// if found on the legacy service, delegate to it
		return s.serviceV1.Update(ctx, clusterID, serviceName, spec)
	} else if !IsIntegratedServiceNotFoundError(err) {
		return errors.WrapIf(err, "failed to retrieve integrated service from the legacy service")
	}

	// delegate to the new implementation
	return s.serviceV2.Update(ctx, clusterID, serviceName, spec)
}

// filterDuplicates identifies integrated services seen by both the legacy and the new service implementations
// and only returns adds ti the returned list the service returned by the legacy service
func (s serviceRouter) filterDuplicates(v1Services []IntegratedService, v2Services []IntegratedService) ([]IntegratedService, error) {
	if len(v1Services) == 0 {
		return v2Services, nil
	}

	if len(v2Services) == 0 {
		return v1Services, nil
	}

	// create a map keyed by the IS name and valued by the IS
	isV1Map := make(map[string]IntegratedService)
	for _, isV1 := range v1Services {
		isV1Map[isV1.Name] = isV1
	}

	deduped := v1Services
	for _, s2 := range v2Services {
		if _, ok := isV1Map[s2.Name]; !ok {
			// the isv2 doesn't exist on v1
			deduped = append(deduped, s2)
		} else {
			s.log.Warn("Integrated service exists on both versions. Version 1 will be returned.", map[string]interface{}{"service": s2.Name})
		}
	}

	return deduped, nil
}
