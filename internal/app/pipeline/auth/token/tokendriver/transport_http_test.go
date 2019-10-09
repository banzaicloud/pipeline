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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/sagikazarmark/kitx/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
)

func TestRegisterHTTPHandlers_CreateToken(t *testing.T) {
	expectedToken := token.NewToken{
		ID:    "id",
		Token: "token",
	}

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			CreateToken: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return expectedToken, nil
			},
		},
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	newTokenReq := token.NewTokenRequest{
		Name:        "token",
		VirtualUser: "",
		ExpiresAt:   nil,
	}

	body, err := json.Marshal(newTokenReq)
	require.NoError(t, err)

	resp, err := ts.Client().Post(ts.URL+"/tokens", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var tokenResp token.NewToken

	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	require.NoError(t, err)

	assert.Equal(t, expectedToken, tokenResp)
}

func TestRegisterHTTPHandlers_CreateToken_VirtualUserDenied(t *testing.T) {
	newTokenReq := token.NewTokenRequest{
		Name:        "token",
		VirtualUser: "",
		ExpiresAt:   nil,
	}

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			CreateToken: endpoint.BusinessErrorMiddleware(func(_ context.Context, _ interface{}) (interface{}, error) {
				return token.NewToken{}, CannotCreateVirtualUser
			}),
		},
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	body, err := json.Marshal(newTokenReq)
	require.NoError(t, err)

	resp, err := ts.Client().Post(ts.URL+"/tokens", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRegisterHTTPHandlers_ListTokens(t *testing.T) {
	expectedTokens := []token.Token{
		{
			ID:        "id",
			Name:      "name",
			ExpiresAt: nil,
			CreatedAt: time.Date(2019, time.September, 30, 14, 37, 00, 00, time.UTC),
		},
	}

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			ListTokens: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return expectedTokens, nil
			},
		},
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/tokens")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var tokenResp []token.Token

	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	require.NoError(t, err)

	assert.Equal(t, expectedTokens, tokenResp)
}

func TestRegisterHTTPHandlers_GetToken(t *testing.T) {
	tokenID := "id"

	expectedToken := token.Token{
		ID:        tokenID,
		Name:      "name",
		ExpiresAt: nil,
		CreatedAt: time.Date(2019, time.September, 30, 14, 37, 00, 00, time.UTC),
	}

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			GetToken: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return expectedToken, nil
			},
		},
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/tokens/" + tokenID)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var tokenResp token.Token

	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	require.NoError(t, err)

	assert.Equal(t, expectedToken, tokenResp)
}

func TestRegisterHTTPHandlers_GetToken_NotFound(t *testing.T) {
	tokenID := "id"

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			GetToken: endpoint.BusinessErrorMiddleware(func(_ context.Context, _ interface{}) (interface{}, error) {
				return token.Token{}, token.NotFoundError{ID: tokenID}
			}),
		},
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/tokens/" + tokenID)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRegisterHTTPHandlers_DeleteToken(t *testing.T) {
	tokenID := "id"

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			DeleteToken: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return nil, nil
			},
		},
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/tokens/"+tokenID, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}
