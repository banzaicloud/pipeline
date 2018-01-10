package auth0

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"gopkg.in/square/go-jose.v2"

	"gopkg.in/square/go-jose.v2/jwt"
)

func TestFromRequestExtraction(t *testing.T) {
	referenceToken := getTestToken(defaultAudience, defaultIssuer, time.Now(), jose.HS256, defaultSecret)
	headerTokenRequest, _ := http.NewRequest("", "http://localhost", nil)

	headerValue := fmt.Sprintf("Bearer %s", referenceToken)
	headerTokenRequest.Header.Add("Authorization", headerValue)

	token, err := FromHeader(headerTokenRequest)

	if err != nil {
		t.Error(err)
		return
	}

	claims := jwt.Claims{}
	err = token.Claims([]byte("secret"), &claims)
	if err != nil {
		t.Errorf("Claims should be decoded correctly with default token: %q \n", err)
		t.FailNow()
	}

	if claims.Issuer != defaultIssuer || !reflect.DeepEqual(claims.Audience, jwt.Audience(defaultAudience)) {
		t.Error("Invalid issuer, audience or subject:", claims.Issuer, claims.Audience)
	}
}

func TestInvalidExtract(t *testing.T) {
	headerTokenRequest, _ := http.NewRequest("", "http://localhost", nil)
	_, err := FromHeader(headerTokenRequest)

	if err == nil {
		t.Error("A request without valid Authorization header should return an error.")
	}
}
