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
	"time"

	"emperror.dev/errors"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	appkitendpoint "github.com/sagikazarmark/appkit/endpoint"
	"github.com/sagikazarmark/kitx/correlation"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"
	kitxtransport "github.com/sagikazarmark/kitx/transport"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"
	"golang.org/x/oauth2"

	"github.com/banzaicloud/pipeline/internal/app/frontend/issue"
	"github.com/banzaicloud/pipeline/internal/app/frontend/issue/issueadapter"
	"github.com/banzaicloud/pipeline/internal/app/frontend/issue/issuedriver"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification/notificationadapter"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification/notificationdriver"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
	"github.com/banzaicloud/pipeline/internal/platform/buildinfo"
)

// RegisterApp returns a new HTTP application.
func RegisterApp(
	router *mux.Router,
	config Config,
	db *gorm.DB,
	buildInfo buildinfo.BuildInfo,
	userExtractor issue.UserExtractor,
	logger Logger,
	errorHandler ErrorHandler,
) error {
	endpointMiddleware := []endpoint.Middleware{
		correlation.Middleware(),
		appkitendpoint.LoggingMiddleware(logger),
		appkitendpoint.ClientErrorMiddleware,
	}

	httpServerOptions := []kithttp.ServerOption{
		kithttp.ServerErrorHandler(kitxtransport.NewErrorHandler(errorHandler)),
		kithttp.ServerErrorEncoder(kitxhttp.NewJSONProblemErrorEncoder(apphttp.NewDefaultProblemConverter())),
		kithttp.ServerBefore(correlation.HTTPToContext()),
	}

	{
		store := notificationadapter.NewGormStore(db)
		service := notification.NewService(store)
		endpoints := notificationdriver.TraceEndpoints(notificationdriver.MakeEndpoints(
			service,
			kitxendpoint.Combine(endpointMiddleware...),
		))

		notificationdriver.RegisterHTTPHandlers(
			endpoints,
			router.PathPrefix("/notifications").Subrouter(),
			kitxhttp.ServerOptions(httpServerOptions),
		)
	}

	if config.Issue.Enabled {
		formatter := issue.NewMarkdownFormatter(issue.VersionInformation{
			Version:    buildInfo.Version,
			CommitHash: buildInfo.CommitHash,
			BuildDate:  buildInfo.BuildDate,
		})

		var reporter issue.Reporter

		switch config.Issue.Driver {
		case "github":
			config := config.Issue.Github

			httpClient := oauth2.NewClient(
				context.Background(),
				oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.Token}),
			)
			httpClient.Timeout = time.Second * 10

			reporter = issueadapter.NewGitHubReporter(github.NewClient(httpClient), config.Owner, config.Repository)

		default:
			return errors.NewWithDetails("unknown issue driver", "driver", config.Issue.Driver)
		}

		service := issue.NewService(
			config.Issue.Labels,
			userExtractor,
			formatter,
			reporter,
			logger,
		)
		endpoints := issuedriver.TraceEndpoints(issuedriver.MakeEndpoints(
			service,
			kitxendpoint.Combine(endpointMiddleware...),
		))

		issuedriver.RegisterHTTPHandlers(
			endpoints,
			router.PathPrefix("/issues").Subrouter(),
			kitxhttp.ServerOptions(httpServerOptions),
		)
	}

	return nil
}
