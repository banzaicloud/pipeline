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

package auth

import (
	"testing"
	"time"

	"github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type idGeneratorStub struct {
	id string
}

func (s idGeneratorStub) Generate() string {
	return s.id
}

type clockStub struct {
	now time.Time
}

func (s clockStub) Now() time.Time {
	return s.now
}

func TestJWTTokenGenerator_GenerateToken(t *testing.T) {
	now := time.Date(2019, time.September, 20, 14, 44, 00, 00, time.UTC)

	generator := NewJWTTokenGenerator(
		"issuer",
		"audience",
		"signingKey",
		TokenIDGenerator(idGeneratorStub{"id"}),
		TokenGeneratorClock(clockStub{now}),
	)

	tokenID, signedToken, err := generator.GenerateToken("user", NoExpiration, "token", "my_text")
	require.NoError(t, err)

	const expectedSignedToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiJpZCIsImF1ZCI6ImF1ZGllbmNlIiwiaXNzIjoiaXNzdWVyIiwic3ViIjoidXNlciIsImlhdCI6MTU2ODk5MDY0MCwic2NvcGUiOiJhcGk6aW52b2tlIiwidHlwZSI6InRva2VuIiwidGV4dCI6Im15X3RleHQifQ.DZMiDYr8hIuegx2FvKeg1HCsg2LyxMNT4Kc0hVcLI3E"

	assert.Equal(t, "id", tokenID)
	assert.Equal(t, expectedSignedToken, signedToken)
}

func TestTokenManager_GenerateToken(t *testing.T) {
	const sub = "subject"
	const tokenType = "apitoken"
	const tokenText = "token"

	generator := new(MockTokenGenerator)
	generator.On("GenerateToken", sub, NoExpiration, tokenType, tokenText).Return("id", "token", nil)
	generator.On("GenerateToken", sub, NoExpiration, tokenType, tokenText).Return("id2", "token2", nil)

	store := auth.NewInMemoryTokenStore()

	manager := NewTokenManager(generator, store)

	tokenID, signedToken, err := manager.GenerateToken(sub, NoExpiration, tokenType, tokenText, "tokenName", true)
	require.NoError(t, err)

	assert.Equal(t, "id", tokenID)
	assert.Equal(t, "token", signedToken)

	generator.AssertExpectations(t)

	tokens, err := store.List(sub)
	require.NoError(t, err)

	assert.Equal(t, tokens[0].ID, "id")
	assert.Equal(t, tokens[0].Name, "tokenName")
	assert.Equal(t, tokens[0].Value, "token")
}

func TestTokenManager_GenerateToken_NoStoreSecret(t *testing.T) {
	const sub = "subject"
	const tokenType = "apitoken"
	const tokenText = "token"

	generator := new(MockTokenGenerator)
	generator.On("GenerateToken", sub, NoExpiration, tokenType, tokenText).Return("id", "token", nil)
	generator.On("GenerateToken", sub, NoExpiration, tokenType, tokenText).Return("id2", "token2", nil)

	store := auth.NewInMemoryTokenStore()

	manager := NewTokenManager(generator, store)

	tokenID, signedToken, err := manager.GenerateToken(sub, NoExpiration, tokenType, tokenText, "tokenName", false)
	require.NoError(t, err)

	assert.Equal(t, "id", tokenID)
	assert.Equal(t, "token", signedToken)

	generator.AssertExpectations(t)

	tokens, err := store.List(sub)
	require.NoError(t, err)

	assert.Equal(t, tokens[0].ID, "id")
	assert.Equal(t, tokens[0].Name, "tokenName")
	assert.Equal(t, tokens[0].Value, "")
}
