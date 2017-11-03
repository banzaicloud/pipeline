package auth0

import (

	"net/http"
	"gopkg.in/square/go-jose.v2/jwt"
	"strings"
	"errors"

)

var (
	ErrTokenNotFound = errors.New("Token not found")
)
// RequestTokenExtractor can extract a JWT
// from a request.
type RequestTokenExtractor interface {
	Extract(r *http.Request) (*jwt.JSONWebToken, error)
}

// RequestTokenExtractorFunc function conforming
// to the RequestTokenExtractor interface.
type RequestTokenExtractorFunc func(r *http.Request) (*jwt.JSONWebToken, error)

// Extract calls f(r)
func (f RequestTokenExtractorFunc) Extract(r *http.Request) (*jwt.JSONWebToken, error) {
	return f(r)
}

// FromHeader looks for the request in the
// authentication header or call ParseMultipartForm
// if not present.
func FromHeader(r *http.Request) (*jwt.JSONWebToken, error) {

	raw, err := fromHeader(r)
	if err != nil {
		return nil, err
	}
	return jwt.ParseSigned(string(raw))
}

func fromHeader(r *http.Request) ([]byte, error) {
	if authorizationHeader := r.Header.Get("Authorization"); len(authorizationHeader) > 7 && strings.EqualFold(authorizationHeader[0:7], "BEARER ") {
		return []byte(authorizationHeader[7:]), nil
	}
	return nil, ErrTokenNotFound
}
