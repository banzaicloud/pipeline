package auth

import (
	"context"
	"encoding/base32"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	jwtRequest "github.com/dgrijalva/jwt-go/request"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/qor/auth"
	"github.com/qor/auth/authority"
	"github.com/qor/auth/claims"
	"github.com/qor/auth/providers/github"
	"github.com/qor/redirect_back"
	"github.com/qor/session/manager"
	"github.com/satori/go.uuid"
	"github.com/spf13/viper"

	btype "github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/sirupsen/logrus"
)

// DroneSessionCookie holds the name of the Cookie Drone sets in the browser
const DroneSessionCookie = "user_sess"

// DroneSessionCookieType is the Drone token type used for browser sessions
const DroneSessionCookieType = "sess"

// DroneUserCookieType is the Drone token type used for API sessions
const DroneUserCookieType = "user"

// For all Drone token types please see: https://github.com/drone/drone/blob/master/shared/token/token.go#L12

// Init authorization
var (
	logger *logrus.Logger
	log    *logrus.Entry

	RedirectBack *redirect_back.RedirectBack

	Auth *auth.Auth

	Authority *authority.Authority

	authEnabled      bool
	signingKeyBase32 string
	tokenStore       TokenStore

	// JwtIssuer ("iss") claim identifies principal that issued the JWT
	JwtIssuer string

	// JwtAudience ("aud") claim identifies the recipients that the JWT is intended for
	JwtAudience string
)

// TODO se who will win

// Simple init for logging
func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"tag": "Auth"})
}

//ScopedClaims struct to store the scoped claim related things
type ScopedClaims struct {
	jwt.StandardClaims
	Scope string `json:"scope,omitempty"`
	// Drone
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

//DroneClaims struct to store the drone claim related things
type DroneClaims struct {
	*claims.Claims
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

func lookupAccessToken(userId, token string) (bool, error) {
	return tokenStore.Lookup(userId, token)
}

func validateAccessToken(claims *ScopedClaims) (bool, error) {
	userID := claims.Subject
	tokenID := claims.Id
	return lookupAccessToken(userID, tokenID)
}

//Init initialize the auth
func Init() {
	viper.SetDefault("auth.jwtissuer", "https://banzaicloud.com/")
	viper.SetDefault("auth.jwtaudience", "https://pipeline.banzaicloud.com")
	JwtIssuer = viper.GetString("auth.jwtissuer")
	JwtAudience = viper.GetString("auth.jwtaudience")

	signingKey := viper.GetString("auth.tokensigningkey")
	if signingKey == "" {
		panic("Token signing key is missing from configuration")
	}
	signingKeyBase32 = base32.StdEncoding.EncodeToString([]byte(signingKey))

	// A RedirectBack instance which constantly redirects to /ui
	RedirectBack = redirect_back.New(&redirect_back.Config{
		SessionManager:  manager.SessionManager,
		IgnoredPrefixes: []string{"/"},
		IgnoreFunc: func(r *http.Request) bool {
			return true
		},
		FallbackPath: viper.GetString("pipeline.uipath"),
	})

	// Initialize Auth with configuration
	Auth = auth.New(&auth.Config{
		DB:         model.GetDB(),
		Redirector: auth.Redirector{RedirectBack},
		UserModel:  User{},
		ViewPaths:  []string{"views"},
		SessionStorer: &BanzaiSessionStorer{
			SessionStorer: auth.SessionStorer{
				SessionName:    "_auth_session",
				SessionManager: manager.SessionManager,
				SigningMethod:  jwt.SigningMethodHS256,
				SignedString:   signingKeyBase32,
			},
			SignedStringBytes: []byte(signingKeyBase32),
		},
	})
	if viper.GetBool("drone.enabled") {
		Auth.UserStorer = BanzaiUserStorer{signingKeyBase32: signingKeyBase32, droneDB: initDroneDB()}
	} else {
		Auth.UserStorer = BanzaiUserStorer{signingKeyBase32: signingKeyBase32, droneDB: nil}
	}

	githubProvider := github.New(&github.Config{
		// ClientID and ClientSecret is validated inside github.New()
		ClientID:     viper.GetString("auth.clientid"),
		ClientSecret: viper.GetString("auth.clientsecret"),

		// The same as Drone's scopes
		Scopes: []string{
			"repo",
			"repo:status",
			"user:email",
			"read:org",
		},
	})
	githubProvider.AuthorizeHandler = NewGithubAuthorizeHandler(githubProvider)
	Auth.RegisterProvider(githubProvider)

	Authority = authority.New(&authority.Config{
		Auth: Auth,
	})

	tokenStore = NewVaultTokenStore()
}

//GenerateToken generates token from context
// TODO: it should be possible to generate tokens via a token (not just session cookie)
func GenerateToken(c *gin.Context) {
	currentUser := GetCurrentUser(c.Request)
	if currentUser == nil {
		err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
		log.Info(c.ClientIP(), err.Error())
		return
	}

	tokenID := uuid.NewV4().String()

	// Create the Claims
	claims := &ScopedClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    JwtIssuer,
			Audience:  JwtAudience,
			IssuedAt:  jwt.TimeFunc().Unix(),
			ExpiresAt: 0,
			Subject:   strconv.Itoa(int(currentUser.ID)),
			Id:        tokenID,
		},
		Scope: "api:invoke",        // "scope" for Pipeline
		Type:  DroneUserCookieType, // "type" for Drone
		Text:  currentUser.Login,   // "text" for Drone
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(signingKeyBase32))

	if err != nil {
		err = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to sign token: %s", err))
		log.Info(c.ClientIP(), err.Error())
	} else {
		err = tokenStore.Store(strconv.Itoa(int(currentUser.ID)), tokenID)
		if err != nil {
			err = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to store token: %s", err))
			log.Info(c.ClientIP(), err.Error())
		} else {
			c.JSON(http.StatusOK, gin.H{"token": signedToken})
		}
	}
}

func hmacKeyFunc(token *jwt.Token) (interface{}, error) {
	// Don't forget to validate the alg is what you expect:
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("Unexpected signing method: %v", token.Method.Alg())
	}
	return []byte(signingKeyBase32), nil
}

//Handler handles authentication
func Handler(c *gin.Context) {
	currentUser := Auth.GetCurrentUser(c.Request)
	if currentUser != nil {
		return
	}

	claims := ScopedClaims{}
	accessToken, err := jwtRequest.ParseFromRequestWithClaims(c.Request, jwtRequest.OAuth2Extractor, &claims, hmacKeyFunc)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized,
			btype.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Invalid token",
				Error:   err.Error(),
			})
		log.Info("Invalid token:", err)
		return
	}

	isTokenValid, err := validateAccessToken(&claims)
	if err != nil || !accessToken.Valid || !isTokenValid {
		resp := btype.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Invalid token",
		}
		if err != nil {
			resp.Error = err.Error()
			log.Info("Invalid token: ", err)
		} else {
			resp.Error = ""
			log.Info("Invalid token")
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, resp)
		return
	}

	hasScope := strings.Contains(claims.Scope, "api:invoke")

	// TODO: metadata and group check for later hardening
	/**
	metadata, okMetadata := claims["scope"].(map[string]interface{})
	authorization, okAuthorization := metadata["authorization"].(map[string]interface{})
	groups, hasGroups := authorization["groups"].([]interface{})
	**/

	if !hasScope {
		c.AbortWithStatusJSON(http.StatusUnauthorized, btype.ErrorResponse{
			Code:    http.StatusUnauthorized,
			Message: "Need more privileges",
			Error:   err.Error(),
		})
		log.Info("Needs more privileges")
		return
	}

	saveUserIntoContext(c, &claims)

	c.Next()
}

func saveUserIntoContext(c *gin.Context, claims *ScopedClaims) {
	userID, _ := strconv.ParseUint(claims.Subject, 10, 32)
	newContext := context.WithValue(c.Request.Context(), auth.CurrentUser, &User{Model: gorm.Model{ID: uint(userID)}})
	c.Request = c.Request.WithContext(newContext)
}

//BanzaiSessionStorer stores the banzai session
type BanzaiSessionStorer struct {
	auth.SessionStorer
	SignedStringBytes []byte
}

//Update updates the BanzaiSessionStorer
func (sessionStorer *BanzaiSessionStorer) Update(w http.ResponseWriter, req *http.Request, claims *claims.Claims) error {
	token := sessionStorer.SignedToken(claims)
	err := sessionStorer.SessionManager.Add(w, req, sessionStorer.SessionName, token)
	if err != nil {
		log.Info(req.RemoteAddr, err.Error())
		return err
	}

	// Set the drone cookie as well
	currentUser := GetCurrentUser(req)
	if currentUser == nil {
		return fmt.Errorf("Can't get current user")
	}
	droneClaims := &DroneClaims{Claims: claims, Type: DroneSessionCookieType, Text: currentUser.Login}
	droneToken, err := sessionStorer.SignedTokenWithDrone(droneClaims)
	if err != nil {
		log.Info(req.RemoteAddr, err.Error())
		return err
	}
	SetCookie(w, req, DroneSessionCookie, droneToken)
	return nil
}

// SignedTokenWithDrone generate signed token with Claims
func (sessionStorer *BanzaiSessionStorer) SignedTokenWithDrone(claims *DroneClaims) (string, error) {
	token := jwt.NewWithClaims(sessionStorer.SigningMethod, claims)
	return token.SignedString(sessionStorer.SignedStringBytes)
}
