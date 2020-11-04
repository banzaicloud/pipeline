// Copyright © 2018 Banzai Cloud
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

package hollowtrees

import (
	"encoding/base32"
	"fmt"
	"time"

	"github.com/banzaicloud/gin-utilz/auth"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type TokenGenerator interface {
	Generate(userID, orgID uint, expiresAt *time.Time) (string, string, error)
}

type tokenGenerator struct {
	Issuer     string
	Audience   string
	SigningKey string
}

func NewTokenGenerator(issuer, audience, signingKey string) TokenGenerator {
	return &tokenGenerator{
		Issuer:     issuer,
		Audience:   audience,
		SigningKey: signingKey,
	}
}

func (g *tokenGenerator) Generate(userID, orgID uint, expiresAt *time.Time) (string, string, error) {
	tokenID := uuid.Must(uuid.NewV4()).String()

	// Create the Claims
	claims := &auth.ScopedClaims{
		Claims: jwt.Claims{
			Issuer:   g.Issuer,
			Audience: jwt.Audience{g.Audience},
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Subject:  fmt.Sprintf("clusters/%d/%d", orgID, userID),
			ID:       tokenID,
		},
		Scope: "api:invoke",
	}

	if expiresAt != nil {
		claims.Expiry = jwt.NewNumericDate(*expiresAt)
	}

	if g.SigningKey == "" {
		return "", "", errors.New("missing signingKeyBase32")
	}

	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS256,
		Key:       []byte(base32.StdEncoding.EncodeToString([]byte(g.SigningKey))),
	}, nil)
	if err != nil {
		return "", "", errors.Wrap(err, "failed create user token signer")
	}

	signedToken, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		return "", "", errors.Wrap(err, "failed to sign user token")
	}

	return tokenID, signedToken, nil
}
