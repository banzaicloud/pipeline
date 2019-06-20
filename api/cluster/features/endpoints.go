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

package features

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/banzaicloud/pipeline/client"
)

type clusterFeatureDetails = client.ClusterFeatureDetails
type clusterFeatureList map[string]client.ClusterFeatureDetails
type clusterFeatureSpec map[string]interface{}

type Features interface {
	List(ctx context.Context, clusterID uint) (clusterFeatureList, error)
	Activate(ctx context.Context, clusterID uint, featureName string, spec clusterFeatureSpec) error
	Deactivate(ctx context.Context, clusterID uint, featureName string) error
	Details(ctx context.Context, clusterID uint, featureName string) (clusterFeatureDetails, error)
	Update(ctx context.Context, clusterID uint, featureName string, spec clusterFeatureSpec) error
}

type Endpoints struct {
	ListClusterFeatures endpoint.Endpoint

	ActivateClusterFeature   endpoint.Endpoint
	ClusterFeatureDetails    endpoint.Endpoint
	DeactivateClusterFeature endpoint.Endpoint
	UpdateClusterFeature     endpoint.Endpoint
}

func MakeEndpoints(f Features) Endpoints {
	return Endpoints{
		ListClusterFeatures:      MakeListClusterFeaturesEndpoint(f),
		ActivateClusterFeature:   MakeActivateClusterFeatureEndpoint(f),
		ClusterFeatureDetails:    MakeClusterFeatureDetailsEndpoint(f),
		DeactivateClusterFeature: MakeDeactivateClusterFeatureEndpoint(f),
		UpdateClusterFeature:     MakeUpdateClusterFeatureEndpoint(f),
	}
}

type ListClusterFeaturesRequest struct {
	OrganizationID uint
	ClusterID      uint
}

func MakeListClusterFeaturesEndpoint(f Features) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListClusterFeaturesRequest)
		result, err := f.List(ctx, req.ClusterID)
		return result, err
	}
}

type ActivateClusterFeatureRequest struct {
	OrganizationID uint
	ClusterID      uint
	FeatureName    string
	Spec           clusterFeatureSpec
}

func MakeActivateClusterFeatureEndpoint(f Features) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ActivateClusterFeatureRequest)
		err := f.Activate(ctx, req.ClusterID, req.FeatureName, req.Spec)
		return nil, err
	}
}

type ClusterFeatureDetailsRequest struct {
	OrganizationID uint
	ClusterID      uint
	FeatureName    string
}

func MakeClusterFeatureDetailsEndpoint(f Features) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ClusterFeatureDetailsRequest)
		result, err := f.Details(ctx, req.ClusterID, req.FeatureName)
		return result, err
	}
}

type DeactivateClusterFeatureRequest struct {
	OrganizationID uint
	ClusterID      uint
	FeatureName    string
}

func MakeDeactivateClusterFeatureEndpoint(f Features) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeactivateClusterFeatureRequest)
		err := f.Deactivate(ctx, req.ClusterID, req.FeatureName)
		return nil, err
	}
}

type UpdateClusterFeatureRequest struct {
	OrganizationID uint
	ClusterID      uint
	FeatureName    string
	Spec           clusterFeatureSpec
}

func MakeUpdateClusterFeatureEndpoint(f Features) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateClusterFeatureRequest)
		err := f.Update(ctx, req.ClusterID, req.FeatureName, req.Spec)
		return nil, err
	}
}
