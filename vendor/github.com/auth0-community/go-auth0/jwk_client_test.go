package auth0

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gopkg.in/square/go-jose.v2"
)

func TestJWKDownloadKeySuccess(t *testing.T) {
	// Generate RSA
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	// Generate JWKS
	jsonWebKey := jose.JSONWebKey{
		Key:       key,
		Use:       "sig",
		Algorithm: "RS256",
	}

	jwks := JWKS{
		Keys: []jose.JSONWebKey{jsonWebKey},
	}

	value, err := json.Marshal(&jwks)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	// Generate Token
	expiredToken := getTestToken(defaultAudience, defaultIssuer, time.Now().Add(24*time.Hour), jose.RS256, key)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, string(value))
	}))
	opts := JWKClientOptions{URI: ts.URL}
	client := NewJWKClient(opts)

	keys, err := client.downloadKeys()
	if err != nil || len(keys) < 1 {
		t.Errorf("The keys should have been correctly received: %v", err)
		t.FailNow()
	}

	req, _ := http.NewRequest("", "http://localhost", nil)
	headerValue := fmt.Sprintf("Bearer %s", expiredToken)
	req.Header.Add("Authorization", headerValue)

	_, err = client.GetSecret(req)
	if err != nil {
		t.Errorf("Should be considered as valid")
	}

}

func TestJWKDownloadKeyInvalid(t *testing.T) {

	// Invalid content
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Invalid Data")
	}))

	opts := JWKClientOptions{URI: ts.URL}
	client := NewJWKClient(opts)

	_, err := client.downloadKeys()
	if err != ErrInvalidContentType {
		t.Errorf("An ErrInvalidContentType should be returned in case of invalid Content-Type Header.")
	}

	// Invalid Payload
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "Invalid Data")
	}))

	opts = JWKClientOptions{URI: ts.URL}
	client = NewJWKClient(opts)

	_, err = client.downloadKeys()
	if err == nil {
		t.Errorf("An non JSON payload should return an error.")
	}
}
