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

package clusterfeaturedriver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"emperror.dev/emperror"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/pkg/ctxutil"
)

func TestRegisterHTTPHandlers_List(t *testing.T) {
	expectedFeatures := map[string]pipeline.ClusterFeatureDetails{
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
				return expectedFeatures, nil
			},
		},
		handler.PathPrefix("/features").Subrouter(),
		emperror.NewNoopHandler(),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctxutil.WithClusterID(r.Context(), 1))

		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/features")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var featureMap map[string]pipeline.ClusterFeatureDetails

	err = json.NewDecoder(resp.Body).Decode(&featureMap)
	require.NoError(t, err)

	assert.Equal(t, expectedFeatures, featureMap)
}

func TestRegisterHTTPHandlers_Details(t *testing.T) {
	expectedDetails := pipeline.ClusterFeatureDetails{
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
				return expectedDetails, nil
			},
		},
		handler.PathPrefix("/features").Subrouter(),
		emperror.NewNoopHandler(),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctxutil.WithClusterID(r.Context(), 1))
		r = r.WithContext(ctxutil.WithParams(r.Context(), map[string]string{"featureName": "hello-world"}))

		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/features/hello-world")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var featureDetails pipeline.ClusterFeatureDetails

	err = json.NewDecoder(resp.Body).Decode(&featureDetails)
	require.NoError(t, err)

	assert.Equal(t, expectedDetails, featureDetails)
}

func TestRegisterHTTPHandlers_Activate(t *testing.T) {
	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			Activate: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return nil, nil
			},
		},
		handler.PathPrefix("/features").Subrouter(),
		emperror.NewNoopHandler(),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctxutil.WithClusterID(r.Context(), 1))
		r = r.WithContext(ctxutil.WithParams(r.Context(), map[string]string{"featureName": "hello-world"}))

		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	apiReq := pipeline.ActivateClusterFeatureRequest{
		Spec: map[string]interface{}{
			"hello": "world",
		},
	}

	body, err := json.Marshal(apiReq)
	require.NoError(t, err)

	resp, err := ts.Client().Post(ts.URL+"/features/hello-world", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func TestRegisterHTTPHandlers_Deactivate(t *testing.T) {
	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			Deactivate: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return nil, nil
			},
		},
		handler.PathPrefix("/features").Subrouter(),
		emperror.NewNoopHandler(),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctxutil.WithClusterID(r.Context(), 1))
		r = r.WithContext(ctxutil.WithParams(r.Context(), map[string]string{"featureName": "hello-world"}))

		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/features/hello-world", nil)
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
				return nil, nil
			},
		},
		handler.PathPrefix("/features").Subrouter(),
		emperror.NewNoopHandler(),
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctxutil.WithClusterID(r.Context(), 1))
		r = r.WithContext(ctxutil.WithParams(r.Context(), map[string]string{"featureName": "hello-world"}))

		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	apiReq := pipeline.UpdateClusterFeatureRequest{
		Spec: map[string]interface{}{
			"hello": "world",
		},
	}

	body, err := json.Marshal(apiReq)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/features/hello-world", bytes.NewReader(body))
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}
