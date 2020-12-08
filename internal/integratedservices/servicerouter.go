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

// serviceRouter component that routes api calls to the appropriate integrated service version. It's main role is
// to make integrated service versions transparent to clients
// Generally service version 2 is preferred, if not applicable the router falls back to version v1
type serviceRouter struct {
	serviceV1 Service
	serviceV2 Service
}

// NewServiceRouter creates a new service router instance with the passed in service implementations
func NewServiceRouter(serviceV1 Service, serviceV2 Service) Service {
	return serviceRouter{
		serviceV1: serviceV1,
		serviceV2: serviceV2,
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

	return append(issV1, issV2...), nil
}

// Details retrieves the service from the service v2, if not found retrieves it from v1
// Note: an Integrated Service can only be managed by one of the service versions
func (s serviceRouter) Details(ctx context.Context, clusterID uint, serviceName string) (IntegratedService, error) {
	var combined error
	if detailsV2, err := s.serviceV2.Details(ctx, clusterID, serviceName); err == nil {
		return detailsV2, nil
	} else {
		combined = errors.Append(combined, err)
	}

	// fallback to v1
	if detailsV1, err := s.serviceV1.Details(ctx, clusterID, serviceName); err == nil {
		return detailsV1, nil
	} else {
		combined = errors.Append(combined, err)
	}

	return IntegratedService{}, combined
}

// Activate delegates the activation request to the appropriate service version
// New services are always activated with the version 2 service
func (s serviceRouter) Activate(ctx context.Context, clusterID uint, serviceName string, spec IntegratedServiceSpec) error {
	if _, err := s.serviceV1.Details(ctx, clusterID, serviceName); err == nil {
		// if found on service v1 delegate to the service v1
		return s.serviceV1.Activate(ctx, clusterID, serviceName, spec)
	}

	// new service
	return s.serviceV2.Activate(ctx, clusterID, serviceName, spec)
}

func (s serviceRouter) Deactivate(ctx context.Context, clusterID uint, serviceName string) error {
	panic("implement me")
}

func (s serviceRouter) Update(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error {
	panic("implement me")
}
