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

package integratedservicedriver

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

type ListIntegratedServicesRequest struct {
	ClusterID uint
}

// MakeListEndpoint returns an endpoint for the matching method of the underlying service.
func MakeListEndpoint(service integratedservices.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListIntegratedServicesRequest)
		result, err := service.List(ctx, req.ClusterID)
		if err != nil {
			return nil, err
		}

		return transformList(result), nil
	}
}

type IntegratedServiceDetailsRequest struct {
	ClusterID             uint
	IntegratedServiceName string
}

// MakeDetailsEndpoint returns an endpoint for the matching method of the underlying service.
func MakeDetailsEndpoint(service integratedservices.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(IntegratedServiceDetailsRequest)
		result, err := service.Details(ctx, req.ClusterID, req.IntegratedServiceName)
		if err != nil {
			return nil, err
		}

		return transformDetails(result), nil
	}
}

type ActivateIntegratedServiceRequest struct {
	ClusterID             uint
	IntegratedServiceName string
	Spec                  map[string]interface{}
}

// MakeActivateEndpoint returns an endpoint for the matching method of the underlying service.
func MakeActivateEndpoint(service integratedservices.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ActivateIntegratedServiceRequest)
		err := service.Activate(ctx, req.ClusterID, req.IntegratedServiceName, req.Spec)
		return nil, err
	}
}

type DeactivateIntegratedServiceRequest struct {
	ClusterID             uint
	IntegratedServiceName string
}

// MakeDeactivateEndpoint returns an endpoint for the matching method of the underlying service.
func MakeDeactivateEndpoint(service integratedservices.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeactivateIntegratedServiceRequest)
		err := service.Deactivate(ctx, req.ClusterID, req.IntegratedServiceName)
		return nil, err
	}
}

type UpdateIntegratedServiceRequest struct {
	ClusterID             uint
	IntegratedServiceName string
	Spec                  map[string]interface{}
}

// MakeUpdateEndpoint returns an endpoint for the matching method of the underlying service.
func MakeUpdateEndpoint(service integratedservices.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateIntegratedServiceRequest)
		err := service.Update(ctx, req.ClusterID, req.IntegratedServiceName, req.Spec)
		return nil, err
	}
}

func transformDetails(integratedService integratedservices.IntegratedService) pipeline.IntegratedServiceDetails {
	return pipeline.IntegratedServiceDetails{
		Spec:   integratedService.Spec,
		Output: integratedService.Output,
		Status: integratedService.Status,
	}
}

func transformList(integratedServices []integratedservices.IntegratedService) map[string]pipeline.IntegratedServiceDetails {
	integratedServiceDetails := make(map[string]pipeline.IntegratedServiceDetails, len(integratedServices))

	for _, f := range integratedServices {
		integratedServiceDetails[f.Name] = transformDetails(f)
	}

	return integratedServiceDetails
}
