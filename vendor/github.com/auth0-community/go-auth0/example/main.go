package main

import (
	"time"

	"github.com/auth0-community/go-auth0"
	cors "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"

	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	AdminGroup = "Admin"
)
var validator *auth0.JWTValidator

func main() {

	// TODO: Get logs also in files -> use gin.DefaultWriter and io.MultiWriter

	// Gin
	r := gin.Default()

	// Cors
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "PATCH", "DELETE", "GET", "POST"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	/* Routes */

	/*  News */
	r.GET("/news", Auth0Groups(AdminGroup), GetNews)
	// News ID

	r.Run(":6060") // listen and server on 0.0.0.0:8080
}

func GetNews(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"Some data": "some data"})
}

// LoadPublicKey loads a public key from PEM/DER-encoded data.
func LoadPublicKey(data []byte) (interface{}, error) {
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

func init() {
	//Creates a configuration with the Auth0 information
	data, err := ioutil.ReadFile("./public.pub")
	if err != nil {
		panic("Impossible to read key form disk")
	}

	secret, err := LoadPublicKey(data)
	if err != nil {
		panic("Invalid provided key")
	}
	secretProvider := auth0.NewKeyProvider(secret)
	configuration := auth0.NewConfiguration(secretProvider, []string{"AUDIENCE"}, "ISSUER", jose.RS256)
	validator = auth0.NewValidator(configuration)
}

func Auth0Groups(wantedGroups ...string) gin.HandlerFunc {

	return gin.HandlerFunc(func(c *gin.Context) {

		tok, err := validator.ValidateRequest(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			log.Println("Invalid token:", err)
			return
		}

		claims := map[string]interface{}{}
		err = validator.Claims(c.Request, tok, &claims)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			c.Abort()
			log.Println("Invalid claims:", err)
			return
		}

		metadata, okMetadata := claims["app_metadata"].(map[string]interface{})
		authorization, okAuthorization := metadata["authorization"].(map[string]interface{})
		groups, hasGroups := authorization["groups"].([]interface{})
		if !okMetadata || !okAuthorization || !hasGroups || !shouldAccess(wantedGroups, groups) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "need more privileges"})
			c.Abort()
			log.Println("Need more provileges")
			return
		}
		c.Next()
	})
}

func shouldAccess(wantedGroups []string, groups []interface{}) bool {

	if len(groups) < 1 {
		return true
	}

	for _, wantedScope := range wantedGroups {

		scopeFound := false

		for _, iScope := range groups {
			scope, ok := iScope.(string)

			if !ok {
				continue
			}
			if scope == wantedScope {
				scopeFound = true
				break
			}
		}
		if !scopeFound {
			return false
		}
	}
	return true
}
