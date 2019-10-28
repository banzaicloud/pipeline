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
	"encoding/json"
	"net/http"

	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	"github.com/banzaicloud/pipeline/pkg/problems"
)

const secretIDParam = "secretId"

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.ListProjects,
		decodeListProjectsHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.JSONResponseEncoder, errorEncoder),
		options...,
	))
}

func decodeListProjectsHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	secretID := r.Header.Get(secretIDParam)
	if secretID == "" {
		secretID = r.URL.Query().Get(secretIDParam)
	}

	return listProjectsRequest{SecretID: secretID}, nil
}

func errorEncoder(_ context.Context, w http.ResponseWriter, e error) error {
	problem := problems.NewDetailedProblem(http.StatusInternalServerError, "something went wrong")

	w.Header().Set("Content-Type", problems.ProblemMediaType)
	w.WriteHeader(problem.Status)

	err := json.NewEncoder(w).Encode(problem)
	if err != nil {
		return errors.Wrap(err, "failed to encode error response")
	}

	return nil
}
