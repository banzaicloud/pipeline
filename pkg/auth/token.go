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
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
)

// TokenType represents one of the possible token Types.
type TokenType string

// NoExpiration can be passed to the generator to indicate no expiration time.
const NoExpiration int64 = 0

// JWTTokenGenerator generates an API token.
type JWTTokenGenerator struct {
	issuer     string
	audience   string
	signingKey string

	signingMethod jwt.SigningMethod

	idgen IDGenerator
	clock Clock
}

// IDGenerator generates an opaque ID.
type IDGenerator interface {
	Generate() string
}

type uuidGenerator struct{}

func (uuidGenerator) Generate() string { return uuid.Must(uuid.NewV4()).String() }

// Clock provides an interface to Time.
type Clock interface {
	// Now tells the current time.
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// JWTTokenGeneratorOption option configures optional parameters of a JWTTokenGenerator.
type JWTTokenGeneratorOption interface {
	apply(g *JWTTokenGenerator)
}

type jwtTokenGeneratorOptionFunc func(g *JWTTokenGenerator)

func (fn jwtTokenGeneratorOptionFunc) apply(g *JWTTokenGenerator) {
	fn(g)
}

// TokenSigningMethod sets the signing method in a JWTTokenGenerator.
// It falls back to jwt.SigningMethodHS256.
func TokenSigningMethod(signingMethod jwt.SigningMethod) JWTTokenGeneratorOption {
	return jwtTokenGeneratorOptionFunc(func(g *JWTTokenGenerator) {
		g.signingMethod = signingMethod
	})
}

// TokenIDGenerator sets the ID Generator in a JWTTokenGenerator.
// It falls back to UUID.
func TokenIDGenerator(idgen IDGenerator) JWTTokenGeneratorOption {
	return jwtTokenGeneratorOptionFunc(func(g *JWTTokenGenerator) {
		g.idgen = idgen
	})
}

// TokenGeneratorClock sets the clock in a JWTTokenGenerator.
// It falls back to the system clock.
func TokenGeneratorClock(clock Clock) JWTTokenGeneratorOption {
	return jwtTokenGeneratorOptionFunc(func(g *JWTTokenGenerator) {
		g.clock = clock
	})
}

// NewJWTTokenGenerator returns a new JWTTokenGenerator.
func NewJWTTokenGenerator(issuer string, audience string, signingKey string, opts ...JWTTokenGeneratorOption) JWTTokenGenerator {
	generator := JWTTokenGenerator{
		issuer:     issuer,
		audience:   audience,
		signingKey: signingKey,

		signingMethod: jwt.SigningMethodHS256,

		idgen: uuidGenerator{},
		clock: systemClock{},
	}

	for _, opt := range opts {
		opt.apply(&generator)
	}

	return generator
}

// GenerateToken generates a JWT token.
func (g JWTTokenGenerator) GenerateToken(sub string, expiresAt int64, tokenType string, tokenText string) (string, string, error) {
	tokenID := g.idgen.Generate()

	claims := struct {
		jwt.StandardClaims

		Scope string `json:"scope,omitempty"`

		// Virtual user fields
		Type string `json:"type,omitempty"`
		Text string `json:"text,omitempty"`
	}{
		StandardClaims: jwt.StandardClaims{
			Issuer:    g.issuer,
			Audience:  g.audience,
			IssuedAt:  g.clock.Now().Unix(),
			ExpiresAt: expiresAt,
			Subject:   sub,
			Id:        tokenID,
		},
		Scope: "api:invoke",
		Type:  tokenType,
		Text:  tokenText,
	}

	jwtToken := jwt.NewWithClaims(g.signingMethod, claims)

	signedToken, err := jwtToken.SignedString([]byte(g.signingKey))
	if err != nil {
		return "", "", errors.WrapIf(err, "failed to sign token")
	}

	return tokenID, signedToken, nil
}

// TokenGenerator generates a token.
type TokenGenerator interface {
	// GenerateToken generates a token.
	GenerateToken(sub string, expiresAt int64, tokenType string, tokenText string) (string, string, error)
}

// TokenManager manages tokens.
type TokenManager struct {
	generator TokenGenerator
	store     auth.TokenStore
}

// NewTokenManager returns a new TokenManager.
func NewTokenManager(generator TokenGenerator, store auth.TokenStore) TokenManager {
	return TokenManager{
		generator: generator,
		store:     store,
	}
}

// GenerateToken generates a token and stores it in the token store.
func (m TokenManager) GenerateToken(
	sub string,
	expiresAt *time.Time,
	tokenType TokenType,
	tokenText string,
	tokenName string,
	storeSecret bool,
) (string, string, error) {
	tokenExpiresAt := NoExpiration
	if expiresAt != nil {
		tokenExpiresAt = expiresAt.Unix()
	}

	tokenID, signedToken, err := m.generator.GenerateToken(sub, tokenExpiresAt, string(tokenType), tokenText)
	if err != nil {
		return "", "", err
	}

	token := auth.NewToken(tokenID, tokenName)
	token.ExpiresAt = expiresAt

	if storeSecret {
		token.Value = signedToken
	}

	err = m.store.Store(sub, token)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to store token")
	}

	return tokenID, signedToken, nil
}
