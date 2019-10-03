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
	"time"

	"emperror.dev/errors"
)

// Token represents an access token.
type Token struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt,omitempty"`
}

// Service provides access to personal access tokens.
//go:generate sh -c "test -x \"${MGA}\" && ${MGA} gen kit endpoint --outdir tokendriver --with-oc Service || true"
//go:generate sh -c "test -x \"${MOCKERY}\" && ${MOCKERY} -name Service -inpkg || true"
type Service interface {
	// CreateToken creates a new access token. It returns the generated token value.
	CreateToken(ctx context.Context, tokenRequest NewTokenRequest) (NewToken, error)

	// ListTokens lists access tokens for a user.
	ListTokens(ctx context.Context) ([]Token, error)

	// GetToken returns a single access tokens for a user.
	GetToken(ctx context.Context, id string) (Token, error)

	// DeleteToken deletes a single access token for a user.
	DeleteToken(ctx context.Context, id string) error
}

// NewTokenRequest contains necessary information for generating a new token.
type NewTokenRequest struct {
	Name        string     `json:"name,omitempty"`
	VirtualUser string     `json:"virtualUser,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

// NewToken contains a generated token.
type NewToken struct {
	ID    string `json:"id,omitempty"`
	Token string `json:"token,omitempty"`
}

// NewService returns a new Service.
func NewService(
	userExtractor UserExtractor,
	store Store,
	generator Generator,
) Service {
	return service{
		userExtractor: userExtractor,
		store:         store,
		generator:     generator,
	}
}

type service struct {
	userExtractor UserExtractor
	store         Store
	generator     Generator
}

// UserExtractor extracts user information from the context.
type UserExtractor interface {
	// GetUserID returns the ID of the currently authenticated user.
	// If a user cannot be found in the context, it returns false as the second return value.
	GetUserID(ctx context.Context) (uint, bool)

	// GetUserLogin returns the login name of the currently authenticated user.
	// If a user cannot be found in the context, it returns false as the second return value.
	GetUserLogin(ctx context.Context) (string, bool)
}

// Store persists access tokens in a secret store.
type Store interface {
	// Store stores a token in the persistent secret store.
	Store(ctx context.Context, userID uint, tokenID string, name string, expiresAt *time.Time) error

	// List lists the tokens in the store.
	List(ctx context.Context, userID uint) ([]Token, error)

	// Lookup finds a user token.
	Lookup(ctx context.Context, userID uint, tokenID string) (Token, error)

	// Revoke revokes an access token.
	Revoke(ctx context.Context, userID uint, tokenID string) error
}

// NotFoundError is returned if a token cannot be found.
type NotFoundError struct {
	ID string
}

// Error implements the error interface.
func (NotFoundError) Error() string {
	return "token not found"
}

// Details returns error details.
func (e NotFoundError) Details() []interface{} {
	return []interface{}{"tokenId", e.ID}
}

// IsBusinessError tells the transport layer to return this error to the client.
func (e NotFoundError) IsBusinessError() bool {
	return true
}

// Generator generates a token.
type Generator interface {
	// GenerateToken generates a token.
	GenerateToken(sub string, expiresAt int64, tokenType string, value string) (string, string, error)
}

const (
	// CICDUserTokenType is the CICD token type used for API sessions
	CICDUserTokenType = "user"

	// CICDHookTokenType is the CICD token type used for API sessions
	CICDHookTokenType = "hook"
)

func (s service) CreateToken(ctx context.Context, tokenRequest NewTokenRequest) (NewToken, error) {
	userID, ok := s.userExtractor.GetUserID(ctx)
	if !ok {
		return NewToken{}, errors.New("user not found in the context")
	}

	userLogin, ok := s.userExtractor.GetUserLogin(ctx)
	if !ok {
		return NewToken{}, errors.New("user not found in the context")
	}

	if tokenRequest.Name == "" {
		tokenRequest.Name = "generated"
	}

	sub := fmt.Sprint(userID)
	tokenType := CICDUserTokenType

	if tokenRequest.VirtualUser != "" {
		sub = tokenRequest.VirtualUser
		userLogin = tokenRequest.VirtualUser
		tokenType = CICDHookTokenType
	}

	expiresAt := int64(0)
	if tokenRequest.ExpiresAt != nil {
		expiresAt = tokenRequest.ExpiresAt.Unix()
	}

	tokenID, signedToken, err := s.generator.GenerateToken(sub, expiresAt, tokenType, userLogin)
	if err != nil {
		return NewToken{}, err
	}

	err = s.store.Store(ctx, userID, tokenID, tokenRequest.Name, tokenRequest.ExpiresAt)
	if err != nil {
		return NewToken{}, err
	}

	return NewToken{
		ID:    tokenID,
		Token: signedToken,
	}, nil
}

func (s service) ListTokens(ctx context.Context) ([]Token, error) {
	userID, ok := s.userExtractor.GetUserID(ctx)
	if !ok {
		return nil, errors.New("user not found in the context")
	}

	return s.store.List(ctx, userID)
}

func (s service) GetToken(ctx context.Context, id string) (Token, error) {
	userID, ok := s.userExtractor.GetUserID(ctx)
	if !ok {
		return Token{}, errors.New("user not found in the context")
	}

	return s.store.Lookup(ctx, userID, id)
}

func (s service) DeleteToken(ctx context.Context, id string) error {
	userID, ok := s.userExtractor.GetUserID(ctx)
	if !ok {
		return errors.New("user not found in the context")
	}

	return s.store.Revoke(ctx, userID, id)
}
