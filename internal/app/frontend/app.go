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
	"net/http"
	"time"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/sagikazarmark/ocmux"
	"golang.org/x/oauth2"

	"github.com/banzaicloud/pipeline/internal/app/frontend/issue"
	"github.com/banzaicloud/pipeline/internal/app/frontend/issue/issueadapter"
	"github.com/banzaicloud/pipeline/internal/app/frontend/issue/issuedriver"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification/notificationadapter"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification/notificationdriver"
	"github.com/banzaicloud/pipeline/internal/platform/buildinfo"
)

// NewApp returns a new HTTP application.
func NewApp(
	config Config,
	db *gorm.DB,
	buildInfo buildinfo.BuildInfo,
	userExtractor issue.UserExtractor,
	logger Logger,
	errorHandler ErrorHandler,
) (http.Handler, error) {
	router := mux.NewRouter()
	router.Use(ocmux.Middleware())
	frontend := router.PathPrefix("/frontend").Subrouter()

	{
		store := notificationadapter.NewGormStore(db)
		service := notification.NewService(store)
		endpoints := notificationdriver.MakeEndpoints(service)
		subrouter := frontend.PathPrefix("/notifications").Subrouter()
		errorHandler := emperror.WithDetails(errorHandler, "module", "notification")

		notificationdriver.RegisterHTTPHandlers(endpoints, subrouter, errorHandler)

		// Compatibility routes
		notificationdriver.RegisterHTTPHandlers(endpoints, router.PathPrefix("/notifications").Subrouter(), errorHandler)
	}

	{
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
			return nil, errors.NewWithDetails("unknown issue driver", "driver", config.Issue.Driver)
		}

		service := issue.NewService(
			config.Issue.Labels,
			userExtractor,
			formatter,
			reporter,
			logger.WithFields(map[string]interface{}{"module": "issue"}),
		)
		endpoints := issuedriver.MakeEndpoints(service)
		subrouter := frontend.PathPrefix("/issues").Subrouter()
		errorHandler := emperror.WithDetails(errorHandler, "module", "issue")

		issuedriver.RegisterHTTPHandlers(endpoints, subrouter, errorHandler)

		// Compatibility routes
		issuedriver.RegisterHTTPHandlers(endpoints, router.PathPrefix("/issues").Subrouter(), errorHandler)
	}

	return router, nil
}
