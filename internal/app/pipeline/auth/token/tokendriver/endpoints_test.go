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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
)

func TestMakeEndpoints_CreateToken(t *testing.T) {
	ctx := context.Background()

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
	service.On("CreateToken", ctx, newTokenReq).Return(expectedToken, nil)

	e := MakeEndpoints(service).CreateToken

	resp, err := e(context.Background(), newTokenReq)
	require.NoError(t, err)

	assert.Equal(t, expectedToken, resp)

	service.AssertExpectations(t)
}

func TestMakeEndpoints_ListTokens(t *testing.T) {
	ctx := context.Background()

	expectedTokens := []token.Token{
		{
			ID:        "id",
			Name:      "name",
			ExpiresAt: nil,
			CreatedAt: time.Date(2019, time.September, 30, 14, 37, 00, 00, time.UTC),
		},
	}

	service := new(token.MockService)
	service.On("ListTokens", ctx).Return(expectedTokens, nil)

	e := MakeEndpoints(service).ListTokens

	resp, err := e(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, expectedTokens, resp)

	service.AssertExpectations(t)
}

func TestMakeEndpoints_GetToken(t *testing.T) {
	ctx := context.Background()
	tokenID := "id"

	expectedToken := token.Token{
		ID:        tokenID,
		Name:      "name",
		ExpiresAt: nil,
		CreatedAt: time.Date(2019, time.September, 30, 14, 37, 00, 00, time.UTC),
	}

	service := new(token.MockService)
	service.On("GetToken", ctx, tokenID).Return(expectedToken, nil)

	e := MakeEndpoints(service).GetToken

	resp, err := e(context.Background(), getTokenRequest{tokenID})
	require.NoError(t, err)

	assert.Equal(t, expectedToken, resp)

	service.AssertExpectations(t)
}

func TestMakeEndpoints_DeleteToken(t *testing.T) {
	ctx := context.Background()
	tokenID := "id"

	service := new(token.MockService)
	service.On("DeleteToken", ctx, tokenID).Return(nil)

	e := MakeEndpoints(service).DeleteToken

	_, err := e(context.Background(), deleteTokenRequest{tokenID})
	require.NoError(t, err)

	service.AssertExpectations(t)
}
