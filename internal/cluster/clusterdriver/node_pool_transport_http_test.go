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

package clusterdriver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterHTTPHandlers_DeleteNodePool(t *testing.T) {
	tests := []struct {
		name               string
		endpointFunc       func(ctx context.Context, request interface{}) (response interface{}, err error)
		expectedStatusCode int
	}{
		{
			name: "already_deleted",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return true, nil
			},
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name: "async_delete",
			endpointFunc: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return false, nil
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
			RegisterNodePoolHTTPHandlers(
				NodePoolEndpoints{
					DeleteNodePool: test.endpointFunc,
				},
				handler.PathPrefix("/clusters/{clusterId}/nodepools").Subrouter(),
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
