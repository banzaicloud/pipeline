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

// +kit:endpoint:errorStrategy=service
// +testify:mock

// Service provides access to personal access tokens.
type Service interface {
	// CreateToken creates a new access token. It returns the generated token value.
	CreateToken(ctx context.Context, tokenRequest NewTokenRequest) (newToken NewToken, err error)

	// ListTokens lists access tokens for a user.
	ListTokens(ctx context.Context) (tokens []Token, err error)

	// GetToken returns a single access tokens for a user.
	GetToken(ctx context.Context, id string) (token Token, err error)

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

// +testify:mock:testOnly=true

// UserExtractor extracts user information from the context.
type UserExtractor interface {
	// GetUserID returns the ID of the currently authenticated user.
	// If a user cannot be found in the context, it returns false as the second return value.
	GetUserID(ctx context.Context) (uint, bool)

	// GetUserLogin returns the login name of the currently authenticated user.
	// If a user cannot be found in the context, it returns false as the second return value.
	GetUserLogin(ctx context.Context) (string, bool)
}

// +testify:mock:testOnly=true

// Store persists access tokens in a secret store.
type Store interface {
	// Store stores a token in the persistent secret store.
	Store(ctx context.Context, userID string, tokenID string, name string, expiresAt *time.Time) error

	// List lists the tokens in the store.
	List(ctx context.Context, userID string) ([]Token, error)

	// Lookup finds a user token.
	Lookup(ctx context.Context, userID string, tokenID string) (Token, error)

	// Revoke revokes an access token.
	Revoke(ctx context.Context, userID string, tokenID string) error
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

// NotFound tells a client that this error is related to a resource being not found.
// Can be used to translate the error to eg. status code.
func (NotFoundError) NotFound() bool {
	return true
}

// ServiceError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
func (NotFoundError) ServiceError() bool {
	return true
}

// +testify:mock:testOnly=true

// Generator generates a token.
type Generator interface {
	// GenerateToken generates a token.
	GenerateToken(sub string, expiresAt time.Time, tokenType string, value string) (string, string, error)
}

const (
	// UserTokenType is the token type used for API sessions
	UserTokenType = "user"

	// VirtualUserTokenType is the token type used for API sessions by external services
	// Used by PKE at the moment
	// Legacy token type (used by CICD build hook originally)
	VirtualUserTokenType = "hook"
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
	tokenType := UserTokenType

	if tokenRequest.VirtualUser != "" {
		sub = tokenRequest.VirtualUser
		userLogin = tokenRequest.VirtualUser
		tokenType = VirtualUserTokenType
	}

	var expiresAt time.Time
	if tokenRequest.ExpiresAt != nil {
		expiresAt = *tokenRequest.ExpiresAt
	}

	tokenID, signedToken, err := s.generator.GenerateToken(sub, expiresAt, tokenType, userLogin)
	if err != nil {
		return NewToken{}, err
	}

	err = s.store.Store(ctx, sub, tokenID, tokenRequest.Name, tokenRequest.ExpiresAt)
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

	return s.store.List(ctx, fmt.Sprint(userID))
}

func (s service) GetToken(ctx context.Context, id string) (Token, error) {
	userID, ok := s.userExtractor.GetUserID(ctx)
	if !ok {
		return Token{}, errors.New("user not found in the context")
	}

	return s.store.Lookup(ctx, fmt.Sprint(userID), id)
}

func (s service) DeleteToken(ctx context.Context, id string) error {
	userID, ok := s.userExtractor.GetUserID(ctx)
	if !ok {
		return errors.New("user not found in the context")
	}

	return s.store.Revoke(ctx, fmt.Sprint(userID), id)
}
