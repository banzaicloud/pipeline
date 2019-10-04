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
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/auth"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
)

// BankVaultsStore stores user tokens in a Bank-Vaults store.
type BankVaultsStore struct {
	store auth.TokenStore
}

// NewBankVaultsStore returns a new BankVaultsStore.
func NewBankVaultsStore(store auth.TokenStore) BankVaultsStore {
	return BankVaultsStore{
		store: store,
	}
}

// Store stores a token in the persistent secret store.
func (s BankVaultsStore) Store(ctx context.Context, userID string, tokenID string, name string, expiresAt *time.Time) error {
	t := auth.NewToken(tokenID, name)
	t.ExpiresAt = expiresAt

	err := s.store.Store(userID, t)
	if err != nil {
		return errors.WrapIfWithDetails(
			err, "failed to save user access token",
			"userId", userID,
			"tokenId", tokenID,
			"tokenName", name,
		)
	}

	return nil
}

// List lists the tokens in the store.
func (s BankVaultsStore) List(ctx context.Context, userID string) ([]token.Token, error) {
	ts, err := s.store.List(userID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to list user tokens", "userId", userID)
	}

	tokens := make([]token.Token, 0, len(ts))
	for _, t := range ts {
		tokens = append(tokens, s.mapToken(t))
	}

	return tokens, nil
}

// Lookup finds a user token.
func (s BankVaultsStore) Lookup(_ context.Context, userID string, tokenID string) (token.Token, error) {
	t, err := s.store.Lookup(userID, tokenID)
	if err != nil {
		return token.Token{}, errors.WrapIfWithDetails(
			err, "failed to lookup user token",
			"userId", userID,
			"tokenId", tokenID,
		)
	}

	if t == nil {
		return token.Token{}, errors.WithStack(token.NotFoundError{ID: tokenID})
	}

	return s.mapToken(t), nil
}

func (s BankVaultsStore) mapToken(t *auth.Token) token.Token {
	tt := token.Token{
		ID:        t.ID,
		Name:      t.Name,
		ExpiresAt: t.ExpiresAt,
	}

	if t.CreatedAt != nil {
		tt.CreatedAt = *t.CreatedAt
	}

	return tt
}

// Revoke revokes an access token.
func (s BankVaultsStore) Revoke(_ context.Context, userID string, tokenID string) error {
	err := s.store.Revoke(userID, tokenID)
	if err != nil {
		return errors.WrapIfWithDetails(
			err, "failed to revoke user token",
			"userId", userID,
			"tokenId", tokenID,
		)
	}

	return nil
}
