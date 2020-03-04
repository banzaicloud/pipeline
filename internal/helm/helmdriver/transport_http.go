// Copyright © 2020 Banzai Cloud
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

package helmdriver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/helm"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
)

func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodPost).Path("").Handler(kithttp.NewServer(
		endpoints.AddRepository,
		decodeAddRepositoryHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.ListRepositories,
		decodeListRepositoriesHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeListRepositoriesHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodDelete).Path("/{repoName}").Handler(kithttp.NewServer(
		endpoints.DeleteRepository,
		decodeDeleteRepositoryHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeDeleteRepositoryHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodPatch).Path("/{repoName}").Handler(kithttp.NewServer(
		endpoints.UpdateRepository,
		decodeUpdateRepositoryHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))
}

func decodeAddRepositoryHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, e := extractOrgID(r)
	if e != nil {
		return nil, errors.WrapIf(e, "failed to decode add repository request")
	}

	var request pipeline.HelmReposAddRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode request")
	}

	return AddRepositoryRequest{
		OrganizationID: orgID,
		Repository: helm.Repository{
			Name:             request.Name,
			URL:              request.Url,
			PasswordSecretID: request.PasswordSecretRef,
			TlsSecretID:      request.TlsSecretRef,
		}}, nil
}

func decodeUpdateRepositoryHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, e := extractOrgID(r)
	if e != nil {
		return nil, errors.WrapIf(e, "failed to decode patch repository request")
	}

	repoName, err := extractHelmRepoName(r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode list request")
	}

	var request pipeline.HelmReposAddRequest

	dErr := json.NewDecoder(r.Body).Decode(&request)
	if dErr != nil {
		return nil, errors.WrapIf(dErr, "failed to decode request")
	}

	return UpdateRepositoryRequest{
		OrganizationID: orgID,
		Repository: helm.Repository{
			Name:             repoName,
			URL:              request.Url,
			PasswordSecretID: request.PasswordSecretRef,
			TlsSecretID:      request.TlsSecretRef,
		}}, nil
}

func decodeListRepositoriesHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractOrgID(r)
	if err != nil {
		return 0, errors.WrapIf(err, "failed to decode list request")
	}

	return ListRepositoriesRequest{OrganizationID: orgID}, nil
}

func encodeListRepositoriesHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(ListRepositoriesResponse)
	list := make([]pipeline.HelmRepoListItem, 0, len(resp.Repos))
	for _, repo := range resp.Repos {
		list = append(list, pipeline.HelmRepoListItem{
			Name:              repo.Name,
			Url:               repo.URL,
			PasswordSecretRef: repo.PasswordSecretID,
			TlsSecretRef:      repo.TlsSecretID,
		})
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, list)
}

func decodeDeleteRepositoryHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	orgID, err := extractOrgID(r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode list request")
	}

	repoName, err := extractHelmRepoName(r)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode list request")
	}

	return DeleteRepositoryRequest{OrganizationID: orgID, RepoName: repoName}, nil
}

func encodeDeleteRepositoryHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp, ok := response.(DeleteRepositoryResponse)
	if ok && resp.Err == nil {
		w.WriteHeader(http.StatusNoContent)

		return nil
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, resp)
}

func extractOrgID(r *http.Request) (uint, error) {
	vars := mux.Vars(r)

	id, ok := vars["orgId"]
	if !ok || id == "" {
		return 0, errors.NewWithDetails("missing path parameter", "param", "orgId")
	}

	orgID, e := strconv.ParseUint(id, 10, 32)
	if e != nil {
		return 0, errors.WrapIff(e, "failed to parse path param: %s, value:  %s", "id", id)
	}

	return uint(orgID), nil
}

func extractHelmRepoName(r *http.Request) (string, error) {
	vars := mux.Vars(r)

	repoName, ok := vars["repoName"]
	if !ok || repoName == "" {
		return "", errors.NewWithDetails("missing path parameter", "param", "repoName")
	}

	return repoName, nil
}
