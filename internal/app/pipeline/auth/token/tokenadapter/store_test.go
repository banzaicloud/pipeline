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

package tokenadapter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
)

func TestBankVaultsStore_Store(t *testing.T) {
	bstore := auth.NewInMemoryTokenStore()
	store := NewBankVaultsStore(bstore)

	userID := "1"
	tokenID := "token"
	tokenName := "name"
	expiresAt := time.Date(2019, time.September, 30, 15, 15, 00, 00, time.UTC)

	err := store.Store(context.Background(), userID, tokenID, tokenName, &expiresAt)
	require.NoError(t, err)

	bt, err := bstore.Lookup(fmt.Sprint(userID), tokenID)
	require.NoError(t, err)

	expectedToken := &auth.Token{
		ID:        tokenID,
		Name:      tokenName,
		ExpiresAt: &expiresAt,
		CreatedAt: nil,
		Value:     "",
	}

	assert.Equal(t, expectedToken, bt)
}

func TestBankVaultsStore_List(t *testing.T) {
	bstore := auth.NewInMemoryTokenStore()
	store := NewBankVaultsStore(bstore)

	userID := "1"
	tokenID := "token"
	tokenName := "name"
	expiresAt := time.Date(2019, time.September, 30, 15, 15, 00, 00, time.UTC)

	err := bstore.Store(fmt.Sprint(userID), &auth.Token{
		ID:        tokenID,
		Name:      tokenName,
		ExpiresAt: &expiresAt,
		CreatedAt: nil,
		Value:     "",
	})
	require.NoError(t, err)

	tokens, err := store.List(context.Background(), userID)
	require.NoError(t, err)

	expectedTokens := []token.Token{
		{
			ID:        tokenID,
			Name:      tokenName,
			ExpiresAt: &expiresAt,
		},
	}

	assert.Equal(t, expectedTokens, tokens)
}

func TestBankVaultsStore_Lookup(t *testing.T) {
	bstore := auth.NewInMemoryTokenStore()
	store := NewBankVaultsStore(bstore)

	userID := "1"
	tokenID := "token"
	tokenName := "name"
	expiresAt := time.Date(2019, time.September, 30, 15, 15, 00, 00, time.UTC)

	err := bstore.Store(fmt.Sprint(userID), &auth.Token{
		ID:        tokenID,
		Name:      tokenName,
		ExpiresAt: &expiresAt,
		CreatedAt: nil,
		Value:     "",
	})
	require.NoError(t, err)

	tt, err := store.Lookup(context.Background(), userID, tokenID)
	require.NoError(t, err)

	expectedToken := token.Token{
		ID:        tokenID,
		Name:      tokenName,
		ExpiresAt: &expiresAt,
	}

	assert.Equal(t, expectedToken, tt)
}

func TestBankVaultsStore_Revoke(t *testing.T) {
	bstore := auth.NewInMemoryTokenStore()
	store := NewBankVaultsStore(bstore)

	userID := "1"
	tokenID := "token"
	tokenName := "name"
	expiresAt := time.Date(2019, time.September, 30, 15, 15, 00, 00, time.UTC)

	err := bstore.Store(fmt.Sprint(userID), &auth.Token{
		ID:        tokenID,
		Name:      tokenName,
		ExpiresAt: &expiresAt,
		CreatedAt: nil,
		Value:     "",
	})
	require.NoError(t, err)

	err = store.Revoke(context.Background(), userID, tokenID)
	require.NoError(t, err)

	tt, err := bstore.Lookup(fmt.Sprint(userID), tokenID)
	require.NoError(t, err)

	assert.Nil(t, tt)
}
