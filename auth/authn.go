package auth

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/auth0-community/go-auth0"
	"github.com/banzaicloud/pipeline/cloud"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"net/http"
	"strings"

	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
)

const jwksUri = "https://banzaicloud.auth0.com/.well-known/jwks.json"
const auth0ApiIssuer = "https://banzaicloud.auth0.com/"

var auth0ApiAudiences = []string{"https://pipeline.banzaicloud.com"}
var validator *auth0.JWTValidator

//ApiGroup is grouping name for the token
var ApiGroup = "ApiGroup"

//Init
func Init() {
	pubKey := viper.GetString("dev.auth0pub")
	banzaiUtils.LogInfo(banzaiConstants.TagAuth, "PubKey", pubKey)
	data, err := ioutil.ReadFile(pubKey)
	if err != nil {
		panic("Impossible to read key form disk")
	}

	secret, err := loadPublicKey(data)
	if err != nil {
		panic("Invalid provided key")
	}
	secretProvider := auth0.NewKeyProvider(secret)
	configuration := auth0.NewConfiguration(secretProvider, auth0ApiAudiences, auth0ApiIssuer, jose.RS256)
	validator = auth0.NewValidator(configuration)
}

// LoadPublicKey loads a public key from PEM/DER-encoded data.
func loadPublicKey(data []byte) (interface{}, error) {
	input := data

	block, _ := pem.Decode(data)
	if block != nil {
		input = block.Bytes
	}

	// Try to load SubjectPublicKeyInfo
	pub, err0 := x509.ParsePKIXPublicKey(input)
	if err0 == nil {
		return pub, nil
	}

	cert, err1 := x509.ParseCertificate(input)
	if err1 == nil {
		return cert.PublicKey, nil
	}

	return nil, fmt.Errorf("square/go-jose: parse error, got '%s' and '%s'", err0, err1)
}

//Auth0Groups handler for Gin
func Auth0Groups(wantedGroups ...string) gin.HandlerFunc {

	return gin.HandlerFunc(func(c *gin.Context) {

		accessToken, err := validator.ValidateRequest(c.Request)
		if err != nil {
			cloud.SetResponseBodyJson(c, http.StatusUnauthorized, gin.H{
				cloud.JsonKeyError: "invalid token",
			})
			c.Abort()
			banzaiUtils.LogInfo(banzaiConstants.TagAuth, "Invalid token:", err)
			return
		}

		claims := map[string]interface{}{}
		err = validator.Claims(c.Request, accessToken, &claims)
		if err != nil {
			cloud.SetResponseBodyJson(c, http.StatusUnauthorized, gin.H{
				cloud.JsonKeyError: "invalid claims",
			})
			c.Abort()
			banzaiUtils.LogInfo(banzaiConstants.TagAuth, "Invalid claims:", err)
			return
		}

		banzaiUtils.LogDebug(banzaiConstants.TagAuth, "Claims: ", claims)
		hasScope := strings.Contains(claims["scope"].(string), "api:invoke")

		// TODO: metadata and group check for later hardening
		/**
		metadata, okMetadata := claims["scope"].(map[string]interface{})
		authorization, okAuthorization := metadata["authorization"].(map[string]interface{})
		groups, hasGroups := authorization["groups"].([]interface{})
		**/

		if !hasScope {
			cloud.SetResponseBodyJson(c, http.StatusUnauthorized, gin.H{
				cloud.JsonKeyError: "needs more privileges",
			})
			c.Abort()
			banzaiUtils.LogInfo(banzaiConstants.TagAuth, "Needs more privileges")
			return
		}
		c.Next()
	})
}
