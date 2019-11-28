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
	"fmt"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
)

// SCMTokenStore stores SCM tokens in the underlying store.
type SCMTokenStore struct {
	tokenStore  auth.TokenStore
	cicdEnabled bool
}

// NewSCMTokenStore returns a new SCMTokenStore.
func NewSCMTokenStore(tokenStore auth.TokenStore, cicdEnabled bool) SCMTokenStore {
	return SCMTokenStore{
		tokenStore:  tokenStore,
		cicdEnabled: cicdEnabled,
	}
}

// GetSCMToken returns the stored SCM token and the provider name for a user.
func (s SCMTokenStore) GetSCMToken(userID uint) (string, string, error) {
	scmToken, err := s.GetSCMTokenByProvider(userID, GithubTokenID)
	if err == nil && scmToken != "" {
		return scmToken, GithubTokenID, nil
	}

	scmToken, err = s.GetSCMTokenByProvider(userID, GitlabTokenID)
	if err == nil && scmToken != "" {
		return scmToken, GitlabTokenID, nil
	}

	return "", "", emperror.Wrap(err, "failed to fetch user's scm token")
}

// GetSCMToken returns the stored SCM token and the provider name for a user.
func (s SCMTokenStore) GetSCMTokenByProvider(userID uint, provider string) (string, error) {
	token, err := s.tokenStore.Lookup(fmt.Sprint(userID), provider)
	if err != nil {
		return "", emperror.Wrap(err, "failed to lookup user token")
	}

	if token == nil {
		return "", nil
	}

	return token.Value, nil
}

// SaveSCMToken saves an SCM token for a user.
func (s SCMTokenStore) SaveSCMToken(user *User, scmToken string, provider string) error {
	// Revoke the old Github token from Vault if any
	err := s.tokenStore.Revoke(user.IDString(), provider)
	if err != nil {
		return errors.WrapIf(err, "failed to revoke old access token")
	}

	token := auth.NewToken(provider, "scm access token")
	token.Value = scmToken

	err = s.tokenStore.Store(user.IDString(), token)
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to store access token for user", "user", user.Login)
	}

	if s.cicdEnabled && (provider == GithubTokenID || provider == GitlabTokenID) {
		// TODO CICD should use Vault as well, and this should be removed by then
		err = updateUserInCICDDB(user, scmToken)
		if err != nil {
			return errors.WrapIfWithDetails(err, "failed to update access token for user in CICD", "user", user.Login)
		}

		synchronizeCICDRepos(user.Login)
	}

	return nil
}

// RemoveSCMToken removes an SCM token for a user.
func (s SCMTokenStore) RemoveSCMToken(user *User, provider string) error {
	// Revoke the old Github token from Vault if any
	err := s.tokenStore.Revoke(user.IDString(), provider)
	if err != nil {
		return errors.WrapIf(err, "failed to revoke old access token")
	}

	if s.cicdEnabled && (provider == GithubTokenID || provider == GitlabTokenID) {
		// TODO CICD should use Vault as well, and this should be removed by then
		err = updateUserInCICDDB(user, "")
		if err != nil {
			return errors.WrapIfWithDetails(err, "failed to update access token for user in CICD", "user", user.Login)
		}
	}

	return nil
}
