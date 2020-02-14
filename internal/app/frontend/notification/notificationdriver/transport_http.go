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

package notificationdriver

import (
	"context"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.GetNotifications,
		kithttp.NopRequestDecoder,
		kitxhttp.ErrorResponseEncoder(encodeGetNotificationsHTTPResponse, errorEncoder),
		options...,
	))
}

func encodeGetNotificationsHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(GetNotificationsResponse)

	return kitxhttp.JSONResponseEncoder(ctx, w, resp.Notifications)
}
