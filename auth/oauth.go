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
	"emperror.dev/errors"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
)

// RefreshTokenStore stores refresh tokens in the underlying store.
type RefreshTokenStore struct {
	tokenStore auth.TokenStore
}

// NewRefreshTokenStore returns a new RefreshTokenStore.
func NewRefreshTokenStore(tokenStore auth.TokenStore) RefreshTokenStore {
	return RefreshTokenStore{
		tokenStore: tokenStore,
	}
}

// GetRefreshToken returns the refresh token from the token store.
func (s RefreshTokenStore) GetRefreshToken(userID string) (string, error) {
	token, err := TokenStore.Lookup(userID, OAuthRefreshTokenID)
	if err != nil {
		return "", errors.WrapIf(err, "failed to lookup user refresh token")
	}

	if token == nil {
		return "", nil
	}

	return token.Value, nil
}

// SaveRefreshToken saves the refresh token in the token store.
func (s RefreshTokenStore) SaveRefreshToken(userID string, refreshToken string) error {
	// Revoke the old refresh token from Vault if any
	err := s.tokenStore.Revoke(userID, OAuthRefreshTokenID)
	if err != nil {
		return errors.WrapIf(err, "failed to revoke old refresh token")
	}

	token := auth.NewToken(OAuthRefreshTokenID, "OAuth refresh token")
	token.Value = refreshToken

	err = s.tokenStore.Store(userID, token)
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to store refresh token for user", "user", userID)
	}

	return nil
}
