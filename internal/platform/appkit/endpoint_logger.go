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

package appkit

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"

	"github.com/banzaicloud/pipeline/internal/common"
)

// EndpointLoggerFactory logs trace information about a request.
func EndpointLoggerFactory(logger common.Logger) kitxendpoint.MiddlewareFactory {
	return func(name string) endpoint.Middleware {
		return EndpointLogger(logger.WithFields(map[string]interface{}{"operation": name}))
	}
}

// EndpointLogger logs trace information about a request.
func EndpointLogger(logger common.Logger) endpoint.Middleware {
	return func(e endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			logger := logger.WithContext(ctx)

			logger.Trace("processing request")

			defer func(begin time.Time) {
				logger.Trace("processing request finished", map[string]interface{}{
					"took": time.Since(begin),
				})
			}(time.Now())

			return e(ctx, request)
		}
	}
}
