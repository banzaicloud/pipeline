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

package token

import (
	"context"
	"fmt"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate sh -c "test -x \"${MOCKERY}\" && ${MOCKERY} -name UserExtractor -inpkg -testonly || true"
//go:generate sh -c "test -x \"${MOCKERY}\" && ${MOCKERY} -name Store -inpkg -testonly || true"
//go:generate sh -c "test -x \"${MOCKERY}\" && ${MOCKERY} -name Generator -inpkg -testonly || true"

func TestService_CreateToken(t *testing.T) {
	ctx := context.Background()
	userID := uint(1)
	userIDString := fmt.Sprint(userID)
	userLogin := "john.doe"
	tokenID := "id"
	tokenValue := "token"

	tokenRequest := NewTokenRequest{
		Name:      "tokenName",
		ExpiresAt: nil,
	}

	userExtractor := new(MockUserExtractor)
	userExtractor.On("GetUserID", ctx).Return(userID, true)
	userExtractor.On("GetUserLogin", ctx).Return(userLogin, true)

	expectedToken := NewToken{
		ID:    tokenID,
		Token: tokenValue,
	}

	store := new(MockStore)
	store.On("Store", ctx, userIDString, tokenID, tokenRequest.Name, tokenRequest.ExpiresAt).Return(nil)

	generator := new(MockGenerator)
	generator.On("GenerateToken", userIDString, int64(0), CICDUserTokenType, userLogin).Return(tokenID, tokenValue, nil)

	service := NewService(userExtractor, store, generator)

	newToken, err := service.CreateToken(ctx, tokenRequest)
	require.NoError(t, err)

	assert.Equal(t, expectedToken, newToken)

	userExtractor.AssertExpectations(t)
	store.AssertExpectations(t)
	generator.AssertExpectations(t)
}

func TestService_CreateToken_DefaultName(t *testing.T) {
	ctx := context.Background()
	userID := uint(1)
	userIDString := fmt.Sprint(userID)
	userLogin := "john.doe"
	tokenID := "id"
	tokenValue := "token"

	tokenRequest := NewTokenRequest{
		Name:      "",
		ExpiresAt: nil,
	}

	userExtractor := new(MockUserExtractor)
	userExtractor.On("GetUserID", ctx).Return(userID, true)
	userExtractor.On("GetUserLogin", ctx).Return(userLogin, true)

	expectedToken := NewToken{
		ID:    tokenID,
		Token: tokenValue,
	}

	store := new(MockStore)
	store.On("Store", ctx, userIDString, tokenID, "generated", tokenRequest.ExpiresAt).Return(nil)

	generator := new(MockGenerator)
	generator.On("GenerateToken", userIDString, int64(0), CICDUserTokenType, userLogin).Return(tokenID, tokenValue, nil)

	service := NewService(userExtractor, store, generator)

	newToken, err := service.CreateToken(ctx, tokenRequest)
	require.NoError(t, err)

	assert.Equal(t, expectedToken, newToken)

	userExtractor.AssertExpectations(t)
	store.AssertExpectations(t)
	generator.AssertExpectations(t)
}

func TestService_VirtualUser(t *testing.T) {
	ctx := context.Background()
	userID := "virtualUser"
	userLogin := "john.doe"
	tokenID := "id"
	tokenValue := "token"

	tokenRequest := NewTokenRequest{
		Name:        "tokenName",
		VirtualUser: "virtualUser",
		ExpiresAt:   nil,
	}

	userExtractor := new(MockUserExtractor)
	userExtractor.On("GetUserID", ctx).Return(uint(1), true)
	userExtractor.On("GetUserLogin", ctx).Return(userLogin, true)

	expectedToken := NewToken{
		ID:    tokenID,
		Token: tokenValue,
	}

	store := new(MockStore)
	store.On("Store", ctx, userID, tokenID, tokenRequest.Name, tokenRequest.ExpiresAt).Return(nil)

	generator := new(MockGenerator)
	generator.On("GenerateToken", "virtualUser", int64(0), CICDHookTokenType, "virtualUser").Return(tokenID, tokenValue, nil)

	service := NewService(userExtractor, store, generator)

	newToken, err := service.CreateToken(ctx, tokenRequest)
	require.NoError(t, err)

	assert.Equal(t, expectedToken, newToken)

	userExtractor.AssertExpectations(t)
	store.AssertExpectations(t)
	generator.AssertExpectations(t)
}

func TestService_ListTokens(t *testing.T) {
	ctx := context.Background()
	userID := uint(1)
	userIDString := fmt.Sprint(userID)

	userExtractor := new(MockUserExtractor)
	userExtractor.On("GetUserID", ctx).Return(userID, true)

	expectedTokens := []Token{
		{
			ID:        "tokenid",
			Name:      "generated",
			ExpiresAt: nil,
			CreatedAt: time.Date(2019, time.September, 30, 14, 37, 00, 00, time.UTC),
		},
	}

	store := new(MockStore)
	store.On("List", ctx, userIDString).Return(expectedTokens, nil)

	generator := new(MockGenerator)

	service := NewService(userExtractor, store, generator)

	tokens, err := service.ListTokens(ctx)
	require.NoError(t, err)

	assert.Equal(t, expectedTokens, tokens)

	userExtractor.AssertExpectations(t)
	store.AssertExpectations(t)
	generator.AssertExpectations(t)
}

func TestService_GetToken(t *testing.T) {
	ctx := context.Background()
	userID := uint(1)
	userIDString := fmt.Sprint(userID)
	tokenID := "tokenid"

	userExtractor := new(MockUserExtractor)
	userExtractor.On("GetUserID", ctx).Return(userID, true)

	expectedToken := Token{
		ID:        tokenID,
		Name:      "generated",
		ExpiresAt: nil,
		CreatedAt: time.Date(2019, time.September, 30, 14, 37, 00, 00, time.UTC),
	}

	store := new(MockStore)
	store.On("Lookup", ctx, userIDString, tokenID).Return(expectedToken, nil)

	generator := new(MockGenerator)

	service := NewService(userExtractor, store, generator)

	token, err := service.GetToken(ctx, tokenID)
	require.NoError(t, err)

	assert.Equal(t, expectedToken, token)

	userExtractor.AssertExpectations(t)
	store.AssertExpectations(t)
	generator.AssertExpectations(t)
}

func TestService_GetToken_NotFound(t *testing.T) {
	ctx := context.Background()
	userID := uint(1)
	userIDString := fmt.Sprint(userID)
	tokenID := "notfound"

	userExtractor := new(MockUserExtractor)
	userExtractor.On("GetUserID", ctx).Return(userID, true)

	notFoundError := NotFoundError{ID: tokenID}

	store := new(MockStore)
	store.On("Lookup", ctx, userIDString, tokenID).Return(Token{}, notFoundError)

	generator := new(MockGenerator)

	service := NewService(userExtractor, store, generator)

	_, err := service.GetToken(ctx, tokenID)
	require.Error(t, err)

	assert.True(t, errors.Is(err, notFoundError))

	userExtractor.AssertExpectations(t)
	store.AssertExpectations(t)
	generator.AssertExpectations(t)
}

func TestService_DeleteToken(t *testing.T) {
	ctx := context.Background()
	userID := uint(1)
	userIDString := fmt.Sprint(userID)
	tokenID := "tokenid"

	userExtractor := new(MockUserExtractor)
	userExtractor.On("GetUserID", ctx).Return(userID, true)

	store := new(MockStore)
	store.On("Revoke", ctx, userIDString, tokenID).Return(nil)

	generator := new(MockGenerator)

	service := NewService(userExtractor, store, generator)

	err := service.DeleteToken(ctx, tokenID)
	require.NoError(t, err)

	userExtractor.AssertExpectations(t)
	store.AssertExpectations(t)
	generator.AssertExpectations(t)
}
