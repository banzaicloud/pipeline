package auth0

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/go-errors/errors"
	"gopkg.in/square/go-jose.v2"
)

var (
	ErrInvalidContentType = errors.New("Should have a JSON content type for JWKS endpoint.")
	ErrNoKeyFound         = errors.New("No Keys has been found")
	ErrInvalidTokenHeader = errors.New("No valid header found")
	ErrInvalidAlgorithm   = errors.New("Only RS256 is supported")
)

type JWKClientOptions struct {
	URI string
}

type JWKS struct {
	Keys []jose.JSONWebKey `json:"keys"`
}

type JWKClient struct {
	keys    map[string]jose.JSONWebKey
	mu      sync.Mutex
	options JWKClientOptions
}

// NewJWKClient creates a new JWKClient instance from the
// provided options.
func NewJWKClient(options JWKClientOptions) *JWKClient {
	return &JWKClient{keys: map[string]jose.JSONWebKey{}, options: options}
}

// GetKey returns the key associated with the provided ID.
func (j *JWKClient) GetKey(ID string) (jose.JSONWebKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	searchedKey, exist := j.keys[ID]
	if !exist {
		if keys, err := j.downloadKeys(); err != nil {
			return jose.JSONWebKey{}, err
		} else {

			for _, key := range keys {
				// Cache key
				j.keys[key.KeyID] = key

				if key.KeyID == ID {
					searchedKey = key
					exist = true
				}
			}
		}
	}

	if exist {
		return searchedKey, nil
	}
	return jose.JSONWebKey{}, ErrNoKeyFound
}

func (j *JWKClient) downloadKeys() ([]jose.JSONWebKey, error) {
	resp, err := http.Get(j.options.URI)

	if err != nil {
		return []jose.JSONWebKey{}, err
	}
	defer resp.Body.Close()

	if contentH := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentH, "application/json") {
		return []jose.JSONWebKey{}, ErrInvalidContentType
	}

	var jwks = JWKS{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return []jose.JSONWebKey{}, err
	}

	if len(jwks.Keys) < 1 {
		return []jose.JSONWebKey{}, ErrNoKeyFound
	}

	return jwks.Keys, nil
}

func (j *JWKClient) GetSecret(req *http.Request) (interface{}, error) {
	t, err := FromHeader(req)

	if err != nil {
		return nil, err
	}

	if len(t.Headers) < 1 {
		return nil, ErrInvalidTokenHeader
	}

	header := t.Headers[0]
	if header.Algorithm != "RS256" {
		return nil, ErrInvalidAlgorithm
	}

	return j.GetKey(header.KeyID)
}
