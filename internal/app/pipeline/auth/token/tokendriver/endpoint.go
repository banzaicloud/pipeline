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

package tokendriver

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
)

// MakeCreateTokenEndpoint returns an endpoint for the matching method of the underlying service.
func MakeCreateTokenEndpoint(service token.Service) endpoint.Endpoint {
	return kitxendpoint.BusinessErrorMiddleware(func(ctx context.Context, req interface{}) (interface{}, error) {
		return service.CreateToken(ctx, req.(token.NewTokenRequest))
	})
}

// MakeListTokensEndpoint returns an endpoint for the matching method of the underlying service.
func MakeListTokensEndpoint(service token.Service) endpoint.Endpoint {
	return kitxendpoint.BusinessErrorMiddleware(func(ctx context.Context, _ interface{}) (interface{}, error) {
		return service.ListTokens(ctx)
	})
}

type getTokenRequest struct {
	ID string
}

// MakeGetTokenEndpoint returns an endpoint for the matching method of the underlying service.
func MakeGetTokenEndpoint(service token.Service) endpoint.Endpoint {
	return kitxendpoint.BusinessErrorMiddleware(func(ctx context.Context, req interface{}) (interface{}, error) {
		r := req.(getTokenRequest)

		return service.GetToken(ctx, r.ID)
	})
}

type deleteTokenRequest struct {
	ID string
}

// MakeDeleteTokenEndpoint returns an endpoint for the matching method of the underlying service.
func MakeDeleteTokenEndpoint(service token.Service) endpoint.Endpoint {
	return kitxendpoint.BusinessErrorMiddleware(func(ctx context.Context, req interface{}) (interface{}, error) {
		r := req.(deleteTokenRequest)

		return nil, service.DeleteToken(ctx, r.ID)
	})
}
