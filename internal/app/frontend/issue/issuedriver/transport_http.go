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

package issuedriver

import (
	"context"
	"encoding/json"
	"net/http"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/moogar0880/problems"

	"github.com/banzaicloud/pipeline/internal/app/frontend/issue"
	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, errorHandler notification.ErrorHandler) {
	options := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeHTTPError),
		kithttp.ServerErrorHandler(emperror.MakeContextAware(errorHandler)),
	}

	router.Methods(http.MethodPost).Path("").Handler(kithttp.NewServer(
		endpoints.ReportIssue,
		decodeReportIssueHTTPRequest,
		encodeReportIssueHTTPResponse,
		options...,
	))
}

func encodeHTTPError(_ context.Context, _ error, w http.ResponseWriter) {
	problem := problems.NewDetailedProblem(http.StatusInternalServerError, "something went wrong")

	w.Header().Set("Content-Type", problems.ProblemMediaType)
	w.WriteHeader(problem.Status)

	_ = json.NewEncoder(w).Encode(problem)
}

func decodeReportIssueHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var newIssue issue.NewIssue

	err := json.NewDecoder(r.Body).Decode(&newIssue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	return newIssue, nil
}

func encodeReportIssueHTTPResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
	w.WriteHeader(http.StatusCreated)

	return nil
}
