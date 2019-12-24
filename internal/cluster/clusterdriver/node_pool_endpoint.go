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

package clusterdriver

import (
	"context"

	"emperror.dev/errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/mitchellh/mapstructure"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

type createNodePoolRequest struct {
	ClusterID uint
	Spec      map[string]interface{}
}

func MakeCreateNodePoolEndpoint(service cluster.NodePoolService) endpoint.Endpoint {
	return kitxendpoint.BusinessErrorMiddleware(func(ctx context.Context, req interface{}) (interface{}, error) {
		request := req.(createNodePoolRequest)

		var nodePool cluster.NewNodePool

		err := mapstructure.Decode(request.Spec, &nodePool)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode node pool")
		}

		return nil, service.CreateNodePool(ctx, request.ClusterID, nodePool, request.Spec)
	})
}

type deleteNodePoolRequest struct {
	ClusterID    uint
	NodePoolName string
}

func MakeDeleteNodePoolEndpoint(service cluster.NodePoolService) endpoint.Endpoint {
	return kitxendpoint.BusinessErrorMiddleware(func(ctx context.Context, req interface{}) (interface{}, error) {
		request := req.(deleteNodePoolRequest)

		return service.DeleteNodePool(ctx, request.ClusterID, request.NodePoolName)
	})
}
