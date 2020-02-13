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
	"emperror.dev/errors/match"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	appkithttp "github.com/sagikazarmark/appkit/transport/http"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter(
		appkithttp.WithProblemMatchers(
			appkithttp.NewStatusProblemMatcher(http.StatusForbidden, match.Is(CannotCreateVirtualUser).MatchError),
		),
	))

	router.Methods(http.MethodPost).Path("").Handler(kithttp.NewServer(
		endpoints.CreateToken,
		decodeCreateTokenHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeCreateTokenHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.ListTokens,
		kithttp.NopRequestDecoder,
		kitxhttp.ErrorResponseEncoder(encodeListTokensHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("/{id}").Handler(kithttp.NewServer(
		endpoints.GetToken,
		decodeGetTokenHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeGetTokenHTTPResponse, errorEncoder),
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

	return CreateTokenRequest{TokenRequest: newTokenRequest}, nil
}

func encodeCreateTokenHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(CreateTokenResponse)

	return kitxhttp.JSONResponseEncoder(ctx, w, resp.NewToken)
}

func encodeListTokensHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(ListTokensResponse)

	return kitxhttp.JSONResponseEncoder(ctx, w, resp.Tokens)
}

func decodeGetTokenHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "id")
	}

	return GetTokenRequest{Id: id}, nil
}

func encodeGetTokenHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(GetTokenResponse)

	return kitxhttp.JSONResponseEncoder(ctx, w, resp.Token)
}

func decodeDeleteTokenHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "id")
	}

	return DeleteTokenRequest{Id: id}, nil
}
