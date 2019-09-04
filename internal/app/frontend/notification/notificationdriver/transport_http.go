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
	"encoding/json"
	"net/http"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/moogar0880/problems"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, errorHandler notification.ErrorHandler) {
	options := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeHTTPError),
		kithttp.ServerErrorHandler(emperror.MakeContextAware(errorHandler)),
	}

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.GetNotifications,
		decodeGetNotificationsHTTPRequest,
		encodeGetNotificationsHTTPResponse,
		options...,
	))
}

func encodeHTTPError(_ context.Context, _ error, w http.ResponseWriter) {
	problem := problems.NewDetailedProblem(http.StatusInternalServerError, "something went wrong")

	w.Header().Set("Content-Type", problems.ProblemMediaType)
	_ = json.NewEncoder(w).Encode(problem)
}

func decodeGetNotificationsHTTPRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

func encodeGetNotificationsHTTPResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(response)

	return errors.Wrap(err, "failed to send response")
}
