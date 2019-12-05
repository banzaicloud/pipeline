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

	"github.com/go-kit/kit/endpoint"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

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
