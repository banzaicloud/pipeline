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

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/banzaicloud/pipeline/internal/helm3"
)

func TestRegisterHTTPHandlers_AddRepository(t *testing.T) {

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			AddRepository: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return AddRepositoryResponse{}, nil
			},
		},
		handler.PathPrefix("/orgs/{orgId}/helmrepos").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

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

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

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
