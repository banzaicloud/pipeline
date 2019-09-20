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
	"time"

	bauth "github.com/banzaicloud/bank-vaults/pkg/sdk/auth"

	"github.com/banzaicloud/pipeline/pkg/auth"
)

// ClusterToken is the token given to clusters to manage themselves.
const ClusterToken auth.TokenType = "cluster"

// ClusterTokenGenerator looks up or generates and stores a token for a cluster.
type ClusterTokenGenerator struct {
	tokenManager TokenManager
	tokenStore   bauth.TokenStore
}

// TokenManager manages tokens.
type TokenManager interface {
	// GenerateToken generates a token and stores it in the token store.
	GenerateToken(
		sub string,
		expiresAt *time.Time,
		tokenType auth.TokenType,
		tokenText string,
		tokenName string,
		storeSecret bool,
	) (string, string, error)
}

// NewClusterTokenGenerator returns a new ClusterTokenGenerator.
func NewClusterTokenGenerator(tokenManager TokenManager, tokenStore bauth.TokenStore) ClusterTokenGenerator {
	return ClusterTokenGenerator{
		tokenManager: tokenManager,
		tokenStore:   tokenStore,
	}
}

// GenerateClusterToken looks up or generates and stores a token for a cluster.
func (g ClusterTokenGenerator) GenerateClusterToken(orgID uint, clusterID uint) (string, string, error) {
	userID := fmt.Sprintf("clusters/%d/%d", orgID, clusterID)

	if tokens, err := g.tokenStore.List(userID); err == nil {
		for _, token := range tokens {
			if token.Value != "" && token.ExpiresAt == nil {
				return token.ID, token.Value, nil
			}
		}
	}

	return g.tokenManager.GenerateToken(userID, nil, ClusterToken, userID, userID, true)
}
