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

package clusterdriver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

func TestRegisterHTTPHandlers_DeleteCluster(t *testing.T) {
	tests := []struct {
		name               string
		endpointFunc       func(ctx context.Context, request interface{}) (response interface{}, err error)
		expectedStatusCode int
	}{
		{
			name: "already_deleted",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return DeleteClusterResponse{Deleted: true}, nil
			},
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name: "async_delete",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return DeleteClusterResponse{Deleted: false}, nil
			},
			expectedStatusCode: http.StatusAccepted,
		},
	}

	t.Run("no_field", func(t *testing.T) {
		for _, test := range tests {
			test := test

			t.Run(test.name, func(t *testing.T) {
				const orgID = uint(1)
				const clusterID = uint(1)
				const force = true

				handler := mux.NewRouter()
				RegisterHTTPHandlers(
					Endpoints{
						DeleteCluster: test.endpointFunc,
					},
					handler.PathPrefix("/orgs/{orgId}/clusters/{clusterId}").Subrouter(),
				)

				ts := httptest.NewServer(handler)
				defer ts.Close()

				req, err := http.NewRequest(
					http.MethodDelete,
					fmt.Sprintf("%s/orgs/%d/clusters/%d?force=%t", ts.URL, orgID, clusterID, force),
					nil,
				)
				require.NoError(t, err)

				resp, err := ts.Client().Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
			})
		}
	})

	t.Run("id_field", func(t *testing.T) {
		for _, test := range tests {
			test := test

			t.Run(test.name, func(t *testing.T) {
				const orgID = uint(1)
				const clusterID = uint(1)
				const force = true

				handler := mux.NewRouter()
				RegisterHTTPHandlers(
					Endpoints{
						DeleteCluster: test.endpointFunc,
					},
					handler.PathPrefix("/orgs/{orgId}/clusters/{clusterId}").Subrouter(),
				)

				ts := httptest.NewServer(handler)
				defer ts.Close()

				req, err := http.NewRequest(
					http.MethodDelete,
					fmt.Sprintf("%s/orgs/%d/clusters/%d?force=%t&field=id", ts.URL, orgID, clusterID, force),
					nil,
				)
				require.NoError(t, err)

				resp, err := ts.Client().Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
			})
		}
	})

	t.Run("name_field", func(t *testing.T) {
		for _, test := range tests {
			test := test

			t.Run(test.name, func(t *testing.T) {
				const orgID = uint(1)
				const clusterName = "my-cluster"
				const force = true

				handler := mux.NewRouter()
				RegisterHTTPHandlers(
					Endpoints{
						DeleteCluster: test.endpointFunc,
					},
					handler.PathPrefix("/orgs/{orgId}/clusters/{clusterId}").Subrouter(),
				)

				ts := httptest.NewServer(handler)
				defer ts.Close()

				req, err := http.NewRequest(
					http.MethodDelete,
					fmt.Sprintf("%s/orgs/%d/clusters/%s?force=%t&field=name", ts.URL, orgID, clusterName, force),
					nil,
				)
				require.NoError(t, err)

				resp, err := ts.Client().Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
			})
		}
	})
}

func TestRegisterHTTPHandlers_CreateNodePool(t *testing.T) {
	tests := []struct {
		name               string
		endpointFunc       func(ctx context.Context, request interface{}) (response interface{}, err error)
		expectedStatusCode int
	}{
		{
			name: "invalid",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return CreateNodePoolResponse{Err: cluster.NewValidationError(
					"invalid node pool request",
					[]string{"name cannot be empty"},
				)}, nil
			},
			expectedStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "already_exists",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return CreateNodePoolResponse{Err: cluster.NodePoolAlreadyExistsError{
					ClusterID: 1,
					NodePool:  "pool0",
				}}, nil
			},
			expectedStatusCode: http.StatusConflict,
		},
		{
			name: "success",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return CreateNodePoolResponse{}, nil
			},
			expectedStatusCode: http.StatusAccepted,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			const clusterID = uint(1)

			handler := mux.NewRouter()
			RegisterHTTPHandlers(
				Endpoints{
					CreateNodePool: test.endpointFunc,
				},
				handler.PathPrefix("/clusters/{clusterId}").Subrouter(),
			)

			ts := httptest.NewServer(handler)
			defer ts.Close()

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("%s/clusters/%d/nodepools", ts.URL, clusterID),
				strings.NewReader(`{"name": "pool0"}`),
			)
			require.NoError(t, err)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
		})
	}
}

func TestRegisterHTTPHandlers_UpdateNodePool(t *testing.T) {
	tests := []struct {
		name               string
		endpointFunc       func(ctx context.Context, request interface{}) (response interface{}, err error)
		expectedStatusCode int
	}{
		{
			name: "NotFound",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return UpdateNodePoolResponse{Err: cluster.NodePoolNotFoundError{
					ClusterID: 1,
					NodePool:  "pool0",
				}}, nil
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name: "Invalid",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return UpdateNodePoolResponse{Err: cluster.NewValidationError(
					"invalid node pool update request",
					[]string{"invalid instance type"},
				)}, nil
			},
			expectedStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name: "success",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return UpdateNodePoolResponse{}, nil
			},
			expectedStatusCode: http.StatusAccepted,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			const clusterID = uint(1)

			handler := mux.NewRouter()
			RegisterHTTPHandlers(
				Endpoints{
					UpdateNodePool: test.endpointFunc,
				},
				handler.PathPrefix("/clusters/{clusterId}").Subrouter(),
			)

			ts := httptest.NewServer(handler)
			defer ts.Close()

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("%s/clusters/%d/nodepools/%s/update", ts.URL, clusterID, "pool0"),
				strings.NewReader(`{"instanceType": "some-instance-type"}`),
			)
			require.NoError(t, err)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
		})
	}
}

func TestRegisterHTTPHandlers_DeleteNodePool(t *testing.T) {
	tests := []struct {
		name               string
		endpointFunc       func(ctx context.Context, request interface{}) (response interface{}, err error)
		expectedStatusCode int
	}{
		{
			name: "already_deleted",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return DeleteNodePoolResponse{Deleted: true}, nil
			},
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name: "async_delete",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return DeleteNodePoolResponse{Deleted: false}, nil
			},
			expectedStatusCode: http.StatusAccepted,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			const clusterID = uint(1)
			const nodePoolName = "pool0"

			handler := mux.NewRouter()
			RegisterHTTPHandlers(
				Endpoints{
					DeleteNodePool: test.endpointFunc,
				},
				handler.PathPrefix("/clusters/{clusterId}").Subrouter(),
			)

			ts := httptest.NewServer(handler)
			defer ts.Close()

			req, err := http.NewRequest(
				http.MethodDelete,
				fmt.Sprintf("%s/clusters/%d/nodepools/%s", ts.URL, clusterID, nodePoolName),
				nil,
			)
			require.NoError(t, err)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, test.expectedStatusCode, resp.StatusCode)
		})
	}
}
