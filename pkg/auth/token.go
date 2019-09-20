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
	"github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
)

// TokenType represents one of the possible token Types.
type TokenType string

// NoExpiration can be passed to the generator to indicate no expiration time.
const NoExpiration int64 = 0

// ScopedClaims struct to store the scoped claim related things.
type ScopedClaims struct {
	jwt.StandardClaims

	Scope string `json:"scope,omitempty"`

	// Virtual user fields
	Type TokenType `json:"type,omitempty"`
	Text string    `json:"text,omitempty"`
}

// TokenGenerator generates an API token.
type TokenGenerator struct {
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

// TokenGenerator option configures optional parameters of a TokenGenerator.
type TokenGeneratorOption interface {
	apply(g *TokenGenerator)
}

type tokenGeneratorOptionFunc func(g *TokenGenerator)

func (fn tokenGeneratorOptionFunc) apply(g *TokenGenerator) {
	fn(g)
}

// TokenSigningMethod sets the signing method in a TokenGenerator.
// It falls back to jwt.SigningMethodHS256.
func TokenSigningMethod(signingMethod jwt.SigningMethod) TokenGeneratorOption {
	return tokenGeneratorOptionFunc(func(g *TokenGenerator) {
		g.signingMethod = signingMethod
	})
}

// TokenIDGenerator sets the ID Generator in a TokenGenerator.
// It falls back to UUID.
func TokenIDGenerator(idgen IDGenerator) TokenGeneratorOption {
	return tokenGeneratorOptionFunc(func(g *TokenGenerator) {
		g.idgen = idgen
	})
}

// TokenGeneratorClock sets the clock in a TokenGenerator.
// It falls back to the system clock.
func TokenGeneratorClock(clock Clock) TokenGeneratorOption {
	return tokenGeneratorOptionFunc(func(g *TokenGenerator) {
		g.clock = clock
	})
}

// NewTokenGenerator returns a new TokenGenerator.
func NewTokenGenerator(issuer string, audience string, signingKey string, opts ...TokenGeneratorOption) TokenGenerator {
	generator := TokenGenerator{
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

// GenerateToken looks up, or generates and stores a token for a cluster
func (g TokenGenerator) GenerateToken(sub string, expiresAt int64, tokenType TokenType, tokenText string) (string, string, error) {
	tokenID := g.idgen.Generate()

	claims := ScopedClaims{
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
