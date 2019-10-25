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

package clusterfeaturedriver

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

type ListClusterFeaturesRequest struct {
	ClusterID uint
}

// MakeListEndpoint returns an endpoint for the matching method of the underlying service.
func MakeListEndpoint(service clusterfeature.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListClusterFeaturesRequest)
		result, err := service.List(ctx, req.ClusterID)
		if err != nil {
			return nil, err
		}

		return transformList(result), nil
	}
}

type ClusterFeatureDetailsRequest struct {
	ClusterID   uint
	FeatureName string
}

// MakeDetailsEndpoint returns an endpoint for the matching method of the underlying service.
func MakeDetailsEndpoint(service clusterfeature.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ClusterFeatureDetailsRequest)
		result, err := service.Details(ctx, req.ClusterID, req.FeatureName)
		if err != nil {
			return nil, err
		}

		return transformDetails(result), nil
	}
}

type ActivateClusterFeatureRequest struct {
	ClusterID   uint
	FeatureName string
	Spec        map[string]interface{}
}

// MakeActivateEndpoint returns an endpoint for the matching method of the underlying service.
func MakeActivateEndpoint(service clusterfeature.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ActivateClusterFeatureRequest)
		err := service.Activate(ctx, req.ClusterID, req.FeatureName, req.Spec)
		return nil, err
	}
}

type DeactivateClusterFeatureRequest struct {
	ClusterID   uint
	FeatureName string
}

// MakeDeactivateEndpoint returns an endpoint for the matching method of the underlying service.
func MakeDeactivateEndpoint(service clusterfeature.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeactivateClusterFeatureRequest)
		err := service.Deactivate(ctx, req.ClusterID, req.FeatureName)
		return nil, err
	}
}

type UpdateClusterFeatureRequest struct {
	ClusterID   uint
	FeatureName string
	Spec        map[string]interface{}
}

// MakeUpdateEndpoint returns an endpoint for the matching method of the underlying service.
func MakeUpdateEndpoint(service clusterfeature.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateClusterFeatureRequest)
		err := service.Update(ctx, req.ClusterID, req.FeatureName, req.Spec)
		return nil, err
	}
}

func transformDetails(feature clusterfeature.Feature) pipeline.ClusterFeatureDetails {
	return pipeline.ClusterFeatureDetails{
		Spec:   feature.Spec,
		Output: feature.Output,
		Status: feature.Status,
	}
}

func transformList(features []clusterfeature.Feature) map[string]pipeline.ClusterFeatureDetails {
	featureDetails := make(map[string]pipeline.ClusterFeatureDetails, len(features))

	for _, f := range features {
		featureDetails[f.Name] = transformDetails(f)
	}

	return featureDetails
}
