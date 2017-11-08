package auth

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/auth0-community/go-auth0"
	"github.com/banzaicloud/pipeline/conf"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"net/http"
	"strings"
)

var log = conf.Logger()

const JWKS_URI = "https://banzaicloud.auth0.com/.well-known/jwks.json"
const AUTH0_API_ISSUER = "https://banzaicloud.auth0.com/"

var AUTH0_API_AUDIENCE = []string{"https://pipeline.banzaicloud.com"}
var validator *auth0.JWTValidator

var (
	ApiGroup = "ApiGroup"
)

func init() {
	//Creates a configuration with the Auth0 information
	conf.Init()
	pubKey := viper.GetString("dev.auth0pub")
	log.Info("PubKey", pubKey)
	data, err := ioutil.ReadFile(pubKey)
	if err != nil {
		panic("Impossible to read key form disk")
	}

	secret, err := loadPublicKey(data)
	if err != nil {
		panic("Invalid provided key")
	}
	secretProvider := auth0.NewKeyProvider(secret)
	configuration := auth0.NewConfiguration(secretProvider, AUTH0_API_AUDIENCE, AUTH0_API_ISSUER, jose.RS256)
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

func Auth0Groups(wantedGroups ...string) gin.HandlerFunc {

	return gin.HandlerFunc(func(c *gin.Context) {

		accessToken, err := validator.ValidateRequest(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			log.Info("Invalid token:", err)
			return
		}

		claims := map[string]interface{}{}
		err = validator.Claims(c.Request, accessToken, &claims)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			c.Abort()
			log.Info("Invalid claims:", err)
			return
		}

		log.Println("Claims: ", claims)
		has_scope := strings.Contains(claims["scope"].(string), "api:invoke")

		// TODO: metadata and group check for later hardening
		/**
		metadata, okMetadata := claims["scope"].(map[string]interface{})
		authorization, okAuthorization := metadata["authorization"].(map[string]interface{})
		groups, hasGroups := authorization["groups"].([]interface{})
		**/

		if !has_scope {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "needs more privileges"})
			c.Abort()
			log.Info("Needs more privileges")
			return
		}
		c.Next()
	})
}
