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

package integratedservicesdriver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

func TestRegisterHTTPHandlers_List(t *testing.T) {
	expectedIntegratedServices := map[string]pipeline.IntegratedServiceDetails{
		"example": {
			Status: "ACTIVE",
			Spec: map[string]interface{}{
				"hello": "world",
			},
			Output: map[string]interface{}{
				"hello": "world",
			},
		},
	}

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			List: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return ListResponse{Services: []integratedservices.IntegratedService{
					{
						Name:   "example",
						Status: "ACTIVE",
						Spec: map[string]interface{}{
							"hello": "world",
						},
						Output: map[string]interface{}{
							"hello": "world",
						},
					},
				}}, nil
			},
		},
		handler.PathPrefix("/clusters/{clusterId}/services").Subrouter(),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/clusters/1/services")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var integratedServices map[string]pipeline.IntegratedServiceDetails

	err = json.NewDecoder(resp.Body).Decode(&integratedServices)
	require.NoError(t, err)

	assert.Equal(t, expectedIntegratedServices, integratedServices)
}

func TestRegisterHTTPHandlers_Details(t *testing.T) {
	expectedDetails := pipeline.IntegratedServiceDetails{
		Spec: map[string]interface{}{
			"hello": "world",
		},
		Output: map[string]interface{}{
			"hello": "world",
		},
		Status: "ACTIVE",
	}

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			Details: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return DetailsResponse{Service: integratedservices.IntegratedService{
					Name: "example",
					Spec: map[string]interface{}{
						"hello": "world",
					},
					Output: map[string]interface{}{
						"hello": "world",
					},
					Status: "ACTIVE",
				}}, nil
			},
		},
		handler.PathPrefix("/clusters/{clusterId}/services").Subrouter(),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/clusters/1/services/hello-world")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var integratedServiceDetails pipeline.IntegratedServiceDetails

	err = json.NewDecoder(resp.Body).Decode(&integratedServiceDetails)
	require.NoError(t, err)

	assert.Equal(t, expectedDetails, integratedServiceDetails)
}

func TestRegisterHTTPHandlers_Activate(t *testing.T) {
	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			Activate: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return ActivateResponse{}, nil
			},
		},
		handler.PathPrefix("/clusters/{clusterId}/services").Subrouter(),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	apiReq := pipeline.ActivateIntegratedServiceRequest{
		Spec: map[string]interface{}{
			"hello": "world",
		},
	}

	body, err := json.Marshal(apiReq)
	require.NoError(t, err)

	resp, err := ts.Client().Post(ts.URL+"/clusters/1/services/hello-world", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func TestRegisterHTTPHandlers_Deactivate(t *testing.T) {
	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			Deactivate: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return DeactivateResponse{}, nil
			},
		},
		handler.PathPrefix("/clusters/{clusterId}/services").Subrouter(),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/clusters/1/services/hello-world", nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestRegisterHTTPHandlers_Update(t *testing.T) {
	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			Update: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return UpdateResponse{}, nil
			},
		},
		handler.PathPrefix("/clusters/{clusterId}/services").Subrouter(),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	apiReq := pipeline.UpdateIntegratedServiceRequest{
		Spec: map[string]interface{}{
			"hello": "world",
		},
	}

	body, err := json.Marshal(apiReq)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/clusters/1/services/hello-world", bytes.NewReader(body))
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}
