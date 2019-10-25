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

package projectdriver

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/cloud/google/project"
)

type listProjectsRequest struct {
	SecretID string
}

type listProjectsResponse struct {
	Projects []project.Project `json:"projects"`
}

func MakeListProjectsEndpoint(service project.Service) endpoint.Endpoint {
	return kitxendpoint.BusinessErrorMiddleware(func(ctx context.Context, req interface{}) (interface{}, error) {
		r := req.(listProjectsRequest)

		projects, err := service.ListProjects(ctx, r.SecretID)
		if err != nil {
			return nil, err
		}

		return listProjectsResponse{Projects: projects}, nil
	})
}
