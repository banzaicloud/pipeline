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

package secrettypedriver

import (
	"context"
	"net/http"

	"emperror.dev/errors"
	"emperror.dev/errors/match"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	appkithttp "github.com/sagikazarmark/appkit/transport/http"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/secrettype"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter(
		appkithttp.WithProblemMatchers(
			appkithttp.NewStatusProblemMatcher(http.StatusNotFound, match.Is(secrettype.ErrNotSupportedSecretType).MatchError),
		),
	))

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.ListSecretTypes,
		kithttp.NopRequestDecoder,
		kitxhttp.ErrorResponseEncoder(encodeListSecretTypesHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("/{type}").Handler(kithttp.NewServer(
		endpoints.GetSecretType,
		decodeGetSecretTypeHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeGetSecretTypeHTTPResponse, errorEncoder),
		options...,
	))
}

func encodeListSecretTypesHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(ListSecretTypesResponse)

	return kitxhttp.JSONResponseEncoder(ctx, w, resp.SecretTypes)
}

func decodeGetSecretTypeHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)

	t, ok := vars["type"]
	if !ok || t == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "type")
	}

	return GetSecretTypeRequest{SecretType: t}, nil
}

func encodeGetSecretTypeHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(GetSecretTypeResponse)

	return kitxhttp.JSONResponseEncoder(ctx, w, resp.SecretTypeDef)
}
