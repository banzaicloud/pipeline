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
	"strings"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
)

// Middleware describes a service middleware.
type Middleware func(token.Service) token.Service

// AuthorizationMiddleware makes sure a user has the required permissions to create a token.
func AuthorizationMiddleware(authorizer Authorizer) Middleware {
	return func(next token.Service) token.Service {
		return authorizationMiddleware{
			next: next,

			authorizer: authorizer,
		}
	}
}

// Authorizer checks if a context has permission to execute an action.
type Authorizer interface {
	// Authorize authorizes a context to execute an action on an object.
	Authorize(ctx context.Context, action string, object interface{}) (bool, error)
}

type authorizationMiddleware struct {
	next token.Service

	authorizer Authorizer
}

type sentinel string

func (e sentinel) Error() string {
	return string(e)
}

func (e sentinel) IsBusinessError() bool {
	return true
}

// CannotCreateVirtualUser is returned when a user does not have the right to create a virtual user token.
const CannotCreateVirtualUser = sentinel("cannot create virtual user")

func (m authorizationMiddleware) CreateToken(ctx context.Context, tokenRequest token.NewTokenRequest) (token.NewToken, error) {
	if tokenRequest.VirtualUser != "" { // authorize creating a virtual user
		orgName := strings.Split(tokenRequest.VirtualUser, "/")[0]

		ok, err := m.authorizer.Authorize(ctx, "virtualUser.create", orgName)
		if err != nil {
			return token.NewToken{}, err
		}

		if !ok {
			return token.NewToken{}, CannotCreateVirtualUser
		}
	}

	return m.next.CreateToken(ctx, tokenRequest)
}

func (m authorizationMiddleware) ListTokens(ctx context.Context) ([]token.Token, error) {
	return m.next.ListTokens(ctx)
}

func (m authorizationMiddleware) GetToken(ctx context.Context, id string) (token.Token, error) {
	return m.next.GetToken(ctx, id)
}

func (m authorizationMiddleware) DeleteToken(ctx context.Context, id string) error {
	return m.next.DeleteToken(ctx, id)
}
