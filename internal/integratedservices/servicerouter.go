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

	return s.filterDuplicates(issV1, issV2, clusterID)
}

// Details retrieves the service from the service v1 if not found retrieves it from v2
// Note: an Integrated Service can only be managed by one of the service versions
func (s serviceRouter) Details(ctx context.Context, clusterID uint, serviceName string) (IntegratedService, error) {
	var (
		// cache variables to spare redundant external calls
		detailsV1, detailsV2 IntegratedService
		errV1, errV2         error
	)

	if detailsV1, errV1 = s.serviceV1.Details(ctx, clusterID, serviceName); errV1 == nil {
		if detailsV1.Status != IntegratedServiceStatusInactive {
			// return the legacy integrated service details
			return detailsV1, nil
		}
	} else if !IsUnknownIntegratedServiceError(errV1) {
		return IntegratedService{}, errors.Wrapf(errV1, "failed to retrieve legacy integrated service details")
	}

	if detailsV2, errV2 = s.serviceV2.Details(ctx, clusterID, serviceName); errV2 != nil {
		if IsUnknownIntegratedServiceError(errV2) {
			// fallback to the legacy implementation
			return detailsV1, errV1
		}

		return IntegratedService{}, errors.Wrapf(errV2, "failed to retrieve legacy integrated service details")
	}
	// delegate to the new version of the service
	return detailsV2, errV2
}

// Activate delegates the activation request to the appropriate service version
// New services are always activated with the version 2 service
func (s serviceRouter) Activate(ctx context.Context, clusterID uint, serviceName string, spec IntegratedServiceSpec) error {
	// check the status of the service before triggering the activation
	if isSvc, err := s.Details(ctx, clusterID, serviceName); err != nil || isSvc.Status != IntegratedServiceStatusInactive {
		if err != nil {
			return errors.WrapIf(err, "failed to activate integrated service")
		}

		// the integrated service has already been activated (it might be in error or pending state here)
		return errors.WithStackIf(serviceAlreadyActiveError{
			ServiceName: serviceName,
		})
	}

	if applies, err := s.appliesToLegacy(ctx, clusterID, serviceName); err == nil {
		if applies {
			return s.serviceV1.Activate(ctx, clusterID, serviceName, spec)
		}
	} else if !IsUnknownIntegratedServiceError(err) {
		return errors.WrapIf(err, "failed to retrieve integrated service from the legacy service")
	}

	if err := s.serviceV2.Activate(ctx, clusterID, serviceName, spec); err != nil {
		if IsUnknownIntegratedServiceError(err) {
			// fallback to the legacy !
			return s.serviceV1.Activate(ctx, clusterID, serviceName, spec)
		}

		return err
	}

	// delegate to the new implementation
	return nil
}

func (s serviceRouter) Deactivate(ctx context.Context, clusterID uint, serviceName string) error {
	if applies, err := s.appliesToLegacy(ctx, clusterID, serviceName); err == nil {
		// if found on the legacy service, delegate to it
		if applies {
			return s.serviceV1.Deactivate(ctx, clusterID, serviceName)
		}
	} else if !IsIntegratedServiceNotFoundError(err) {
		return errors.WrapIf(err, "failed to retrieve integrated service from the legacy service")
	}

	// delegate to the new implementation
	return s.serviceV2.Deactivate(ctx, clusterID, serviceName)
}

func (s serviceRouter) Update(ctx context.Context, clusterID uint, serviceName string, spec IntegratedServiceSpec) error {
	if applies, err := s.appliesToLegacy(ctx, clusterID, serviceName); err == nil {
		// if found on the legacy service, delegate to
		if applies {
			return s.serviceV1.Update(ctx, clusterID, serviceName, spec)
		}
	} else if !IsIntegratedServiceNotFoundError(err) {
		return errors.WrapIf(err, "failed to retrieve integrated service from the legacy service")
	}

	// delegate to the new implementation
	return s.serviceV2.Update(ctx, clusterID, serviceName, spec)
}

// filterDuplicates identifies integrated services seen by both the legacy and the new service implementations;
// legacy services take precedence over v2 services in the returned list
func (s serviceRouter) filterDuplicates(v1Services []IntegratedService, v2Services []IntegratedService, clusterID uint) ([]IntegratedService, error) {
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
			s.log.Warn("Integrated service exists on both versions. Version 1 will be returned.", map[string]interface{}{"service": s2.Name, "clusterID": clusterID})
		}
	}

	return deduped, nil
}

// appliesToLegacy checks whether the legacy service should be used
func (s serviceRouter) appliesToLegacy(ctx context.Context, clusterID uint, serviceName string) (bool, error) {
	// the legacy service return with an inactive service instance in case the service is not found
	details, err := s.serviceV1.Details(ctx, clusterID, serviceName)
	if err != nil {
		return false, err
	}

	if details.Status == IntegratedServiceStatusInactive {
		// this means the service is not found
		return false, nil
	}

	return true, nil
}
