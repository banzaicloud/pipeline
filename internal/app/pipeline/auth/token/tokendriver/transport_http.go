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
	"encoding/json"
	"net/http"

	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/moogar0880/problems"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	router.Methods(http.MethodPost).Path("").Handler(kithttp.NewServer(
		endpoints.CreateToken,
		decodeCreateTokenHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.JSONResponseEncoder, errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.ListTokens,
		kithttp.NopRequestDecoder,
		kitxhttp.JSONResponseEncoder,
		options...,
	))

	router.Methods(http.MethodGet).Path("/{id}").Handler(kithttp.NewServer(
		endpoints.GetToken,
		decodeGetTokenHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.JSONResponseEncoder, errorEncoder),
		options...,
	))

	router.Methods(http.MethodDelete).Path("/{id}").Handler(kithttp.NewServer(
		endpoints.DeleteToken,
		decodeDeleteTokenHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusNoContent), errorEncoder),
		options...,
	))
}

func decodeCreateTokenHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var newTokenRequest token.NewTokenRequest

	err := json.NewDecoder(r.Body).Decode(&newTokenRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	return newTokenRequest, nil
}

func decodeGetTokenHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "id")
	}

	return getTokenRequest{ID: id}, nil
}

func decodeDeleteTokenHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "id")
	}

	return deleteTokenRequest{ID: id}, nil
}

func errorEncoder(_ context.Context, w http.ResponseWriter, e error) error {
	problem := problems.NewDetailedProblem(http.StatusInternalServerError, "something went wrong")

	switch {
	case errors.Is(e, CannotCreateVirtualUser):
		problem.Status = http.StatusForbidden
		problem.Detail = e.Error()

	case errors.As(e, &token.NotFoundError{}):
		problem.Status = http.StatusNotFound
		problem.Detail = e.Error()
	}

	w.Header().Set("Content-Type", problems.ProblemMediaType)
	w.WriteHeader(problem.Status)

	err := json.NewEncoder(w).Encode(problem)
	if err != nil {
		return errors.Wrap(err, "failed to encode error response")
	}

	return nil
}
