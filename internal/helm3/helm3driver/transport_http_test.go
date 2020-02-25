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

package helm3driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"emperror.dev/errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/banzaicloud/pipeline/internal/helm3"
)

func TestRegisterHTTPHandlers_ListRepositories(t *testing.T) {
	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			ListRepositories: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return ListRepositoriesResponse{
					Repos: []helm3.Repository{
						{
							Name:             "test-repo-name",
							URL:              "https: //kubernetes-charts.banzaicloud.com",
							PasswordSecretID: "0f54013dc29a52560599613be8d67e64bf903ddaaca55d467776c47eea6b4f59",
						},
					},
					Err: nil,
				}, nil
			},
		},
		handler.PathPrefix("/orgs/{orgId}/helmrepos").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := ts.Client().Get(fmt.Sprintf("%s/orgs/%d/helmrepos", ts.URL, 1))

	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRegisterHTTPHandlers_DeleteRepositories(t *testing.T) {
	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			DeleteRepository: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return DeleteRepositoryResponse{}, nil
			},
		},
		handler.PathPrefix("/orgs/{orgId}/helmrepos").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/orgs/%d/helmrepos/%s", ts.URL, 1, "test-repo"),
		nil,
	)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

}

func TestRegisterHTTPHandlers_AddRepository(t *testing.T) {

	tests := []struct {
		name               string
		endpoint           endpoint.Endpoint
		expectedStatusCode int
	}{
		{
			name: "helm repository successfully added",
			endpoint: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return AddRepositoryResponse{}, nil
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "failed to add helm repository (business error / validation)",
			endpoint: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				// response encoded by the response encoder
				return AddRepositoryResponse{
					Err: helm3.NewValidationError("testing", []string{"testing"}),
				}, nil
			},
			expectedStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "failed to add helm repository (internal server error)",
			endpoint: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return AddRepositoryResponse{}, errors.New("testing")
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// GIVEN
			handler := mux.NewRouter()

			RegisterHTTPHandlers(
				Endpoints{
					AddRepository: tt.endpoint,
				},
				handler.PathPrefix("/orgs/{orgId}/helmrepos").Subrouter(), )

			ts := httptest.NewServer(handler)
			defer ts.Close()

			// WHEN
			addRepoReq := helm3.Repository{
				Name:             "test-helm-repository",
				URL:              "https: //kubernetes-charts.banzaicloud.com",
				PasswordSecretID: "0f54013dc29a52560599613be8d67e64bf903ddaaca55d467776c47eea6b4f59",
			}

			body, err := json.Marshal(addRepoReq)
			require.NoError(t, err)

			resp, err := ts.Client().Post(fmt.Sprintf("%s/orgs/%d/helmrepos", ts.URL, 1), "application/json", bytes.NewReader(body))

			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatusCode, resp.StatusCode)
		})
	}
}
