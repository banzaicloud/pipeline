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
	"net/http"

	"emperror.dev/emperror"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/sagikazarmark/ocmux"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification/notificationadapter"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification/notificationdriver"
)

// NewApp returns a new HTTP application.
func NewApp(db *gorm.DB, logger Logger, errorHandler ErrorHandler) http.Handler {
	router := mux.NewRouter().PathPrefix("/frontend").Subrouter()
	router.Use(ocmux.Middleware())

	{
		store := notificationadapter.NewGormStore(db)
		service := notification.NewService(store)
		endpoints := notificationdriver.MakeEndpoints(service)
		errorHandler := emperror.WithDetails(errorHandler, "module", "notification")
		router.Handle("/notifications", notificationdriver.MakeHTTPHandler(endpoints, errorHandler))
	}

	return router
}
