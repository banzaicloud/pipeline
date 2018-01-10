package auth0

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func validConfiguration(configuration Configuration, tokenRaw string) error {
	validator := NewValidator(configuration)
	headerTokenRequest, _ := http.NewRequest("", "http://localhost", nil)
	headerValue := fmt.Sprintf("Bearer %s", tokenRaw)
	headerTokenRequest.Header.Add("Authorization", headerValue)

	_, err := validator.ValidateRequest(headerTokenRequest)
	return err
}

func TestValidatorFull(t *testing.T) {

	token := getTestToken(defaultAudience, defaultIssuer, time.Now().Add(24*time.Hour), jose.HS256, defaultSecret)
	configuration := NewConfiguration(defaultSecretProvider, defaultAudience, defaultIssuer, jose.HS256)
	err := validConfiguration(configuration, token)

	if err != nil {
		t.Error(err)
	}

	invalidToken := token + `wefwefwef`
	err = validConfiguration(configuration, invalidToken)

	if err == nil {
		t.Error("In case of an invalid token, the validation should failed")
	}
}
func TestValidatorEmpty(t *testing.T) {

	configuration := NewConfiguration(defaultSecretProvider, []string{}, "", jose.HS256)
	validToken := getTestToken([]string{}, "", time.Now().Add(24*time.Hour), jose.HS256, defaultSecret)

	err := validConfiguration(configuration, validToken)

	if err != nil {
		t.Error(err)
	}
}

func TestValidatorPartial(t *testing.T) {

	configuration := NewConfiguration(defaultSecretProvider, []string{"required"}, "", jose.HS256)
	validToken := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ`
	err := validConfiguration(configuration, validToken)

	if err == nil {
		t.Error("In case of a wrong password, the validation should failed")
	}
	otherValidToken := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJkcWRxd2Rxd2Rxd2RxIiwibmFtZSI6ImRxd2Rxd2Rxd2Rxd2Rxd2QiLCJhZG1pbiI6ZmFsc2V9.-MZNG6n5KtLIG4Tsa6oi25zZK5oadmrebS-1r1Ln82c`
	err = validConfiguration(configuration, otherValidToken)

	if err == nil {
		t.Error("In case of a wrong password, the validation should failed")
	}
}

func invalidProvider(req *http.Request) (interface{}, error) {
	return nil, errors.New("simple error")
}
func TestInvalidProvider(t *testing.T) {

	provider := SecretProviderFunc(invalidProvider)
	configuration := NewConfiguration(provider, []string{"required"}, "", jose.HS256)

	token := getTestToken([]string{"required"}, "", time.Now().Add(24*time.Hour), jose.HS256, defaultSecret)
	err := validConfiguration(configuration, token)

	if err == nil {
		t.Error("Should failed if the provider was not able to provide a valid secret")
	}
}

func TestClaims(t *testing.T) {

	configuration := NewConfiguration(defaultSecretProvider, defaultAudience, defaultIssuer, jose.HS256)
	validator := NewValidator(configuration)
	token := getTestToken(defaultAudience, defaultIssuer, time.Now().Add(24*time.Hour), jose.HS256, defaultSecret)

	headerTokenRequest, _ := http.NewRequest("", "http://localhost", nil)
	headerValue := fmt.Sprintf("Bearer %s", token)

	// Valid token
	headerTokenRequest.Header.Add("Authorization", headerValue)
	_, err := validator.ValidateRequest(headerTokenRequest)

	if err != nil {
		t.Errorf("The token should be considered valid: %q \n", err)
		t.FailNow()
	}

	claims := map[string]interface{}{}
	tok, _ := jwt.ParseSigned(string(token))

	err = validator.Claims(headerTokenRequest, tok, &claims)

	if err != nil {
		t.Errorf("Claims should be valid in case of valid configuration: %q \n", err)
	}
}

func TestTokenTimeValidity(t *testing.T) {
	expiredToken := getTestToken(defaultAudience, defaultIssuer, time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), jose.HS256, defaultSecret)
	configuration := NewConfiguration(defaultSecretProvider, defaultAudience, defaultIssuer, jose.HS256)
	err := validConfiguration(configuration, expiredToken)
	if err == nil {
		t.Errorf("Message should be considered as outdated")
	}
}
