package auth

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/auth0-community/go-auth0"
	"github.com/banzaicloud/banzai-types/database"
	"github.com/banzaicloud/pipeline/cloud"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/qor/auth"
	"github.com/qor/auth/authority"
	"github.com/qor/auth/providers/github"
	"github.com/qor/redirect_back"
	"github.com/qor/session/manager"
	"github.com/satori/go.uuid"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"

	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
)

const jwksUri = "https://banzaicloud.auth0.com/.well-known/jwks.json"
const auth0ApiIssuer = "https://banzaicloud.auth0.com/"

var auth0ApiAudiences = []string{"https://pipeline.banzaicloud.com"}
var validator *auth0.JWTValidator

//ApiGroup is grouping name for the token
var ApiGroup = "ApiGroup"

// Init authorization
var (
	RedirectBack = redirect_back.New(&redirect_back.Config{
		SessionManager:  manager.SessionManager,
		IgnoredPrefixes: []string{"/auth"},
	})

	Auth *auth.Auth

	Authority *authority.Authority
)

type ScopedClaims struct {
	jwt.StandardClaims
	Scope string `json:"scope"`
}

var signingKey = viper.GetString("dev.tokensigningkey")
var tokenStore TokenStore = NewVaultTokenStore()

func LookupAccessToken(userId, token string) (bool, error) {
	return tokenStore.Lookup(userId, token)
}

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
	// TODO: jose.RS256 once the private key is there
	secret, _ = base64.URLEncoding.DecodeString(signingKey)
	secretProvider = auth0.NewKeyProvider(secret)
	configuration := auth0.NewConfiguration(secretProvider, auth0ApiAudiences, auth0ApiIssuer, jose.HS256)
	validator = auth0.NewValidator(configuration)

	// Initialize Auth with configuration
	Auth = auth.New(&auth.Config{
		DB:         database.DB(),
		Redirector: auth.Redirector{RedirectBack},
		UserModel:  User{},
	})

	Auth.RegisterProvider(github.New(&github.Config{
		ClientID:     viper.GetString("dev.clientid"),
		ClientSecret: viper.GetString("dev.clientsecret"),
	}))

	Authority = authority.New(&authority.Config{
		Auth: Auth,
	})
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

// TODO: it should be possible to generate tokens via a token (not just session cookie)
func GenerateToken(c *gin.Context) {
	currentUser := GetCurrentUser(c.Request)
	if currentUser == nil {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
		return
	}

	tokenID := uuid.NewV4().String()

	// Create the Claims
	claims := &ScopedClaims{
		jwt.StandardClaims{
			Issuer:    auth0ApiIssuer,
			Audience:  auth0ApiAudiences[0],
			IssuedAt:  time.Now().UnixNano(),
			ExpiresAt: time.Now().UnixNano() * 2,
			Subject:   strconv.Itoa(int(currentUser.ID)),
			Id:        tokenID,
		},
		"api:invoke",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signingKeyData, _ := base64.URLEncoding.DecodeString(signingKey)
	signedToken, err := token.SignedString(signingKeyData)

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to sign token: %s", err))
	} else {
		err = tokenStore.Store(strconv.Itoa(int(currentUser.ID)), tokenID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to store token: %s", err))
		}
		c.JSON(http.StatusOK, gin.H{"token": signedToken})
	}
}

//Auth0Groups handler for Gin
func Auth0Handler(wantedGroups ...string) gin.HandlerFunc {

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

		userID := claims["sub"].(string)
		tokenID := claims["jti"].(string)
		isTokenValid, err := LookupAccessToken(userID, tokenID)
		if err != nil || !isTokenValid {
			cloud.SetResponseBodyJson(c, http.StatusUnauthorized, gin.H{
				cloud.JsonKeyError: "invalid token",
			})
			c.Abort()
			banzaiUtils.LogInfo(banzaiConstants.TagAuth, "Invalid token:", err)
			return
		}

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
