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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"emperror.dev/emperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/client"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/pkg/ctxutil"
)

func TestMakeHTTPHandlers_List(t *testing.T) {
	featureService := &dummyFeatureService{
		FeatureList: []clusterfeature.Feature{
			{
				Name: "example",
				Spec: map[string]interface{}{
					"hello": "world",
				},
				Output: map[string]interface{}{
					"hello": "world",
				},
				Status: "ACTIVE",
			},
		},
	}

	handler := MakeHTTPHandlers(MakeEndpoints(featureService), emperror.NewNoopHandler()).List

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctxutil.WithClusterID(r.Context(), 1))

		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	tsClient := ts.Client()

	resp, err := tsClient.Get(ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, resp.StatusCode, http.StatusOK)

	decoder := json.NewDecoder(resp.Body)

	var featureMap map[string]client.ClusterFeatureDetails

	err = decoder.Decode(&featureMap)
	require.NoError(t, err)

	assert.Equal(t, map[string]client.ClusterFeatureDetails{
		"example": {
			Status: "ACTIVE",
			Spec: map[string]interface{}{
				"hello": "world",
			},
			Output: map[string]interface{}{
				"hello": "world",
			},
		},
	}, featureMap)
}

func TestMakeHTTPHandlers_Details(t *testing.T) {
	featureService := &dummyFeatureService{
		FeatureDetails: clusterfeature.Feature{
			Name: "example",
			Spec: map[string]interface{}{
				"hello": "world",
			},
			Output: map[string]interface{}{
				"hello": "world",
			},
			Status: "ACTIVE",
		},
	}

	handler := MakeHTTPHandlers(MakeEndpoints(featureService), emperror.NewNoopHandler()).Details

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctxutil.WithParams(
			ctxutil.WithClusterID(r.Context(), 1),
			map[string]string{
				"featureName": "example",
			},
		))

		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	tsClient := ts.Client()

	resp, err := tsClient.Get(ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, resp.StatusCode, http.StatusOK)

	decoder := json.NewDecoder(resp.Body)

	var featureDetails client.ClusterFeatureDetails

	err = decoder.Decode(&featureDetails)
	require.NoError(t, err)

	assert.Equal(t, client.ClusterFeatureDetails{
		Spec: map[string]interface{}{
			"hello": "world",
		},
		Output: map[string]interface{}{
			"hello": "world",
		},
		Status: "ACTIVE",
	}, featureDetails)
}

func TestMakeHTTPHandlers_Activate(t *testing.T) {
	featureService := &dummyFeatureService{}

	handler := MakeHTTPHandlers(MakeEndpoints(featureService), emperror.NewNoopHandler()).Activate

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctxutil.WithParams(
			ctxutil.WithClusterID(r.Context(), 1),
			map[string]string{
				"featureName": "example",
			},
		))

		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	tsClient := ts.Client()

	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)

	apiReq := client.ActivateClusterFeatureRequest{
		Spec: map[string]interface{}{
			"hello": "world",
		},
	}

	err := encoder.Encode(&apiReq)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ts.URL, &buf)
	require.NoError(t, err)

	resp, err := tsClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, resp.StatusCode, http.StatusAccepted)
}

func TestMakeHTTPHandlers_Deactivate(t *testing.T) {
	featureService := &dummyFeatureService{}

	handler := MakeHTTPHandlers(MakeEndpoints(featureService), emperror.NewNoopHandler()).Deactivate

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctxutil.WithParams(
			ctxutil.WithClusterID(r.Context(), 1),
			map[string]string{
				"featureName": "example",
			},
		))

		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	client := ts.Client()

	req, err := http.NewRequest(http.MethodDelete, ts.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, resp.StatusCode, http.StatusNoContent)
}

func TestMakeHTTPHandlers_Update(t *testing.T) {
	featureService := &dummyFeatureService{}

	handler := MakeHTTPHandlers(MakeEndpoints(featureService), emperror.NewNoopHandler()).Update

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctxutil.WithParams(
			ctxutil.WithClusterID(r.Context(), 1),
			map[string]string{
				"featureName": "example",
			},
		))

		handler.ServeHTTP(w, r)
	}))
	defer ts.Close()

	tsClient := ts.Client()

	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)

	apiReq := client.UpdateClusterFeatureRequest{
		Spec: map[string]interface{}{
			"hello": "world",
		},
	}

	err := encoder.Encode(&apiReq)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, ts.URL, &buf)
	require.NoError(t, err)

	resp, err := tsClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, resp.StatusCode, http.StatusAccepted)
}
