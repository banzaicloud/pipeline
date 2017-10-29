package auth0

import (
	"errors"
	"net/http"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// SecretProvider will provide everything
// needed retrieve the secret.
type SecretProvider interface {
	GetSecret(req *http.Request) (interface{}, error)
}

// SecretProviderFunc simple wrappers to provide
// secret with functions.
type SecretProviderFunc func(req *http.Request) (interface{}, error)

// GetSecret implements the SecretProvider interface.
func (f SecretProviderFunc) GetSecret(req *http.Request) (interface{}, error) {
	return f(req)
}

// NewKeyProvider provide a simple passphrase key provider.
func NewKeyProvider(key interface{}) SecretProvider {
	return SecretProviderFunc(func(req *http.Request) (interface{}, error) {
		return key, nil
	})
}

var (
	ErrNoJWTHeaders = errors.New("No headers in the token")
)

// Configuration contains
// all the information about the
// Auth0 service.
type Configuration struct {
	secretProvider SecretProvider
	expectedClaims jwt.Expected
	signIn         jose.SignatureAlgorithm
}

// NewConfiguration creates a configuration for server
func NewConfiguration(provider SecretProvider, audience []string, issuer string, method jose.SignatureAlgorithm) Configuration {
	return Configuration{
		secretProvider: provider,
		expectedClaims: jwt.Expected{Issuer: issuer, Audience: audience},
		signIn:         method,
	}
}

// JWTValidator helps middleware
// to validate token
type JWTValidator struct {
	config    Configuration
	extractor RequestTokenExtractor
}

// NewValidator creates a new
// validator with the provided configuration.
func NewValidator(config Configuration) *JWTValidator {
	return &JWTValidator{config, RequestTokenExtractorFunc(FromHeader)}
}

// ValidateRequest validates the token within
// the http request.
func (v *JWTValidator) ValidateRequest(r *http.Request) (*jwt.JSONWebToken, error) {

	token, err := v.extractor.Extract(r)

	if err != nil {
		return nil, err
	}

	if len(token.Headers) < 1 {
		return nil, ErrNoJWTHeaders
	}

	header := token.Headers[0]
	if header.Algorithm != string(v.config.signIn) {
		return nil, ErrInvalidAlgorithm
	}

	claims := jwt.Claims{}
	key, err := v.config.secretProvider.GetSecret(r)
	if err != nil {
		return nil, err
	}

	err = token.Claims(key, &claims)

	if err != nil {
		return nil, err
	}

	expected := v.config.expectedClaims.WithTime(time.Now())
	err = claims.Validate(expected)
	return token, err
}

// Claims unmarshall the claims of the provided token
func (v *JWTValidator) Claims(req *http.Request, token *jwt.JSONWebToken, values ...interface{}) error {
	key, err := v.config.secretProvider.GetSecret(req)
	if err != nil {
		return err
	}
	return token.Claims(key, values...)
}
