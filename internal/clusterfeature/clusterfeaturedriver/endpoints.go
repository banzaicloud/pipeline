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

	"github.com/banzaicloud/pipeline/client"
	"github.com/go-kit/kit/endpoint"
	kitoc "github.com/go-kit/kit/tracing/opencensus"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

// FeatureService manages features on Kubernetes clusters.
type FeatureService interface {
	// List lists the activated features and their details.
	List(ctx context.Context, clusterID uint) ([]clusterfeature.Feature, error)

	// Details returns the details of an activated feature.
	Details(ctx context.Context, clusterID uint, featureName string) (clusterfeature.Feature, error)

	// Activate activates a feature.
	Activate(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error

	// Deactivate deactivates a feature.
	Deactivate(ctx context.Context, clusterID uint, featureName string) error

	// Update updates a feature.
	Update(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error
}

// Endpoints collects all of the endpoints that compose the cluster feature service.
// It's meant to be used as a helper struct, to collect all of the endpoints into a
// single parameter.
type Endpoints struct {
	List       endpoint.Endpoint
	Details    endpoint.Endpoint
	Activate   endpoint.Endpoint
	Deactivate endpoint.Endpoint
	Update     endpoint.Endpoint
}

// MakeEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the provided service.
func MakeEndpoints(s FeatureService) Endpoints {
	return Endpoints{
		List:       kitoc.TraceEndpoint("clusterfeature.List")(MakeListEndpoint(s)),
		Details:    kitoc.TraceEndpoint("clusterfeature.Details")(MakeDetailsEndpoint(s)),
		Activate:   kitoc.TraceEndpoint("clusterfeature.Activate")(MakeActivateEndpoint(s)),
		Deactivate: kitoc.TraceEndpoint("clusterfeature.Deactivate")(MakeDeactivateEndpoint(s)),
		Update:     kitoc.TraceEndpoint("clusterfeature.Update")(MakeUpdateEndpoint(s)),
	}
}

type ListClusterFeaturesRequest struct {
	ClusterID uint
}

// MakeListEndpoint returns an endpoint for the matching method of the underlying service.
func MakeListEndpoint(s FeatureService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListClusterFeaturesRequest)
		result, err := s.List(ctx, req.ClusterID)
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
func MakeDetailsEndpoint(s FeatureService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ClusterFeatureDetailsRequest)
		result, err := s.Details(ctx, req.ClusterID, req.FeatureName)
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
func MakeActivateEndpoint(s FeatureService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ActivateClusterFeatureRequest)
		err := s.Activate(ctx, req.ClusterID, req.FeatureName, req.Spec)
		return nil, err
	}
}

type DeactivateClusterFeatureRequest struct {
	ClusterID   uint
	FeatureName string
}

// MakeDeactivateEndpoint returns an endpoint for the matching method of the underlying service.
func MakeDeactivateEndpoint(s FeatureService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeactivateClusterFeatureRequest)
		err := s.Deactivate(ctx, req.ClusterID, req.FeatureName)
		return nil, err
	}
}

type UpdateClusterFeatureRequest struct {
	ClusterID   uint
	FeatureName string
	Spec        map[string]interface{}
}

// MakeUpdateEndpoint returns an endpoint for the matching method of the underlying service.
func MakeUpdateEndpoint(s FeatureService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateClusterFeatureRequest)
		err := s.Update(ctx, req.ClusterID, req.FeatureName, req.Spec)
		return nil, err
	}
}

func transformDetails(feature clusterfeature.Feature) client.ClusterFeatureDetails {
	return client.ClusterFeatureDetails{
		Spec:   feature.Spec,
		Output: feature.Output,
		Status: feature.Status,
	}
}

func transformList(features []clusterfeature.Feature) map[string]client.ClusterFeatureDetails {
	featureDetails := make(map[string]client.ClusterFeatureDetails, len(features))

	for _, f := range features {
		featureDetails[f.Name] = transformDetails(f)
	}

	return featureDetails

}
