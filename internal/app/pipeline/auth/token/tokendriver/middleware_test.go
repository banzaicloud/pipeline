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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
)

//go:generate sh -c "test -x \"${MOCKERY}\" && ${MOCKERY} -name Authorizer -inpkg -testonly || true"

func TestAuthorizationMiddleware_CreateToken(t *testing.T) {
	ctx := context.Background()

	tokenRequest := token.NewTokenRequest{
		Name:        "token",
		VirtualUser: "",
		ExpiresAt:   nil,
	}

	expectedNewToken := token.NewToken{
		ID:    "id",
		Token: "token",
	}

	service := new(token.MockService)
	service.On("CreateToken", ctx, tokenRequest).Return(expectedNewToken, nil)

	authorizer := new(MockAuthorizer)

	middleware := AuthorizationMiddleware(authorizer)(service)

	newToken, err := middleware.CreateToken(ctx, tokenRequest)
	require.NoError(t, err)

	assert.Equal(t, expectedNewToken, newToken)

	service.AssertExpectations(t)
	authorizer.AssertExpectations(t)
}

func TestAuthorizationMiddleware_CreateToken_VirtualUser(t *testing.T) {
	ctx := context.Background()

	tokenRequest := token.NewTokenRequest{
		Name:        "token",
		VirtualUser: "example/clusters",
		ExpiresAt:   nil,
	}

	expectedNewToken := token.NewToken{
		ID:    "id",
		Token: "token",
	}

	service := new(token.MockService)
	service.On("CreateToken", ctx, tokenRequest).Return(expectedNewToken, nil)

	authorizer := new(MockAuthorizer)
	authorizer.On("Authorize", ctx, "virtualUser.create", "example").Return(true, nil)

	middleware := AuthorizationMiddleware(authorizer)(service)

	newToken, err := middleware.CreateToken(ctx, tokenRequest)
	require.NoError(t, err)

	assert.Equal(t, expectedNewToken, newToken)

	service.AssertExpectations(t)
	authorizer.AssertExpectations(t)
}

func TestAuthorizationMiddleware_CreateToken_VirtualUserDenied(t *testing.T) {
	ctx := context.Background()

	tokenRequest := token.NewTokenRequest{
		Name:        "token",
		VirtualUser: "example/clusters",
		ExpiresAt:   nil,
	}

	service := new(token.MockService)

	authorizer := new(MockAuthorizer)
	authorizer.On("Authorize", ctx, "virtualUser.create", "example").Return(false, nil)

	middleware := AuthorizationMiddleware(authorizer)(service)

	_, err := middleware.CreateToken(ctx, tokenRequest)
	require.Error(t, err)

	assert.Equal(t, CannotCreateVirtualUser, err)

	service.AssertExpectations(t)
	authorizer.AssertExpectations(t)
}
