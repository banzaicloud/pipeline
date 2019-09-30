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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
)

func TestRegisterHTTPHandlers_CreateToken(t *testing.T) {
	newTokenReq := token.NewTokenRequest{
		Name:        "token",
		VirtualUser: "",
		ExpiresAt:   nil,
	}

	expectedToken := token.NewToken{
		ID:    "id",
		Token: "token",
	}

	service := new(token.MockService)
	service.On("CreateToken", mock.Anything, newTokenReq).Return(expectedToken, nil)

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		MakeEndpoints(service),
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tsClient := ts.Client()

	body, err := json.Marshal(newTokenReq)
	require.NoError(t, err)

	resp, err := tsClient.Post(ts.URL+"/tokens", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	decoder := json.NewDecoder(resp.Body)

	var tokenResp token.NewToken

	err = decoder.Decode(&tokenResp)
	require.NoError(t, err)

	assert.Equal(t, expectedToken, tokenResp)

	service.AssertExpectations(t)
}

func TestRegisterHTTPHandlers_CreateToken_VirtualUserDenied(t *testing.T) {
	newTokenReq := token.NewTokenRequest{
		Name:        "token",
		VirtualUser: "",
		ExpiresAt:   nil,
	}

	service := new(token.MockService)
	service.On("CreateToken", mock.Anything, newTokenReq).Return(token.NewToken{}, CannotCreateVirtualUser)

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		MakeEndpoints(service),
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tsClient := ts.Client()

	body, err := json.Marshal(newTokenReq)
	require.NoError(t, err)

	resp, err := tsClient.Post(ts.URL+"/tokens", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	service.AssertExpectations(t)
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

	service := new(token.MockService)
	service.On("ListTokens", mock.Anything).Return(expectedTokens, nil)

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		MakeEndpoints(service),
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tsClient := ts.Client()

	resp, err := tsClient.Get(ts.URL + "/tokens")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	decoder := json.NewDecoder(resp.Body)

	var tokenResp []token.Token

	err = decoder.Decode(&tokenResp)
	require.NoError(t, err)

	assert.Equal(t, expectedTokens, tokenResp)

	service.AssertExpectations(t)
}

func TestRegisterHTTPHandlers_GetToken(t *testing.T) {
	tokenID := "id"

	expectedToken := token.Token{
		ID:        tokenID,
		Name:      "name",
		ExpiresAt: nil,
		CreatedAt: time.Date(2019, time.September, 30, 14, 37, 00, 00, time.UTC),
	}

	service := new(token.MockService)
	service.On("GetToken", mock.Anything, tokenID).Return(expectedToken, nil)

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		MakeEndpoints(service),
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tsClient := ts.Client()

	resp, err := tsClient.Get(ts.URL + "/tokens/" + tokenID)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	decoder := json.NewDecoder(resp.Body)

	var tokenResp token.Token

	err = decoder.Decode(&tokenResp)
	require.NoError(t, err)

	assert.Equal(t, expectedToken, tokenResp)

	service.AssertExpectations(t)
}

func TestRegisterHTTPHandlers_GetToken_NotFound(t *testing.T) {
	tokenID := "id"

	service := new(token.MockService)
	service.On("GetToken", mock.Anything, tokenID).Return(token.Token{}, token.NotFoundError{ID: tokenID})

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		MakeEndpoints(service),
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tsClient := ts.Client()

	resp, err := tsClient.Get(ts.URL + "/tokens/" + tokenID)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	service.AssertExpectations(t)
}

func TestRegisterHTTPHandlers_DeleteToken(t *testing.T) {
	tokenID := "id"

	service := new(token.MockService)
	service.On("DeleteToken", mock.Anything, tokenID).Return(nil)

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		MakeEndpoints(service),
		handler.PathPrefix("/tokens").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tsClient := ts.Client()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/tokens/"+tokenID, nil)
	require.NoError(t, err)

	resp, err := tsClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	service.AssertExpectations(t)
}
