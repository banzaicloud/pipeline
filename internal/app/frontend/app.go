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

package frontend

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/tracing/opencensus"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	appkitendpoint "github.com/sagikazarmark/appkit/endpoint"
	"github.com/sagikazarmark/kitx/correlation"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"
	kitxtransport "github.com/sagikazarmark/kitx/transport"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification/notificationadapter"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification/notificationdriver"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
)

// RegisterApp returns a new HTTP application.
func RegisterApp(
	router *mux.Router,
	config Config,
	db *gorm.DB,
	logger Logger,
	errorHandler ErrorHandler,
) error {
	endpointMiddleware := []endpoint.Middleware{
		correlation.Middleware(),
		opencensus.TraceEndpoint("", opencensus.WithSpanName(func(ctx context.Context, _ string) string {
			name, _ := kitxendpoint.OperationName(ctx)

			return name
		})),
		appkitendpoint.LoggingMiddleware(logger),
	}

	httpServerOptions := []kithttp.ServerOption{
		kithttp.ServerErrorHandler(kitxtransport.NewErrorHandler(errorHandler)),
		kithttp.ServerErrorEncoder(kitxhttp.NewJSONProblemErrorEncoder(apphttp.NewDefaultProblemConverter())),
		kithttp.ServerBefore(correlation.HTTPToContext()),
	}

	{
		store := notificationadapter.NewGormStore(db)
		service := notification.NewService(store)
		endpoints := notificationdriver.MakeEndpoints(
			service,
			kitxendpoint.Combine(endpointMiddleware...),
		)

		notificationdriver.RegisterHTTPHandlers(
			endpoints,
			router.PathPrefix("/notifications").Subrouter(),
			kitxhttp.ServerOptions(httpServerOptions),
		)
	}

	return nil
}
