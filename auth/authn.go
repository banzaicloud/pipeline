package auth

import (
	"context"
	"encoding/base32"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
	jwtRequest "github.com/dgrijalva/jwt-go/request"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
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
	"github.com/banzaicloud/pipeline/utils"
	"github.com/sirupsen/logrus"
)

// DroneTokenType represents one of the possible Drone token Types
type DroneTokenType string

// DroneSessionCookie holds the name of the Cookie Drone sets in the browser
const DroneSessionCookie = "user_sess"

// DroneSessionCookieType is the Drone token type used for browser sessions
const DroneSessionCookieType = "sess"

// DroneUserTokenType is the Drone token type used for API sessions
const DroneUserTokenType DroneTokenType = "user"

// DroneHookTokenType is the Drone token type used for API sessions
const DroneHookTokenType DroneTokenType = "hook"

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
	Type DroneTokenType `json:"type,omitempty"`
	Text string         `json:"text,omitempty"`
}

//DroneClaims struct to store the drone claim related things
type DroneClaims struct {
	*claims.Claims
	Type DroneTokenType `json:"type,omitempty"`
	Text string         `json:"text,omitempty"`
}

func isTokenWhitelisted(claims *ScopedClaims) (bool, error) {
	userID := claims.Subject
	tokenID := claims.Id
	token, err := tokenStore.Lookup(userID, tokenID)
	return token != nil, err
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
		},
		LogoutHandler: BanzaiLogoutHandler,
		UserStorer:    BanzaiUserStorer{signingKeyBase32: signingKeyBase32, droneDB: initDroneDB()},
	})

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

	tokenStore = NewVaultTokenStore("pipeline")
}

// Install the whole OAuth and JWT Token based auth/authz mechanism to the specified Gin Engine.
func Install(engine *gin.Engine) {
	authHandler := gin.WrapH(Auth.NewServeMux())

	// We have to make the raw net/http handlers a bit Gin-ish
	engine.Use(gin.WrapH(manager.SessionManager.Middleware(utils.NopHandler{})))
	engine.Use(gin.WrapH(RedirectBack.Middleware(utils.NopHandler{})))

	authGroup := engine.Group("/auth/")
	{
		authGroup.GET("/login", authHandler)
		authGroup.GET("/logout", authHandler)
		authGroup.GET("/register", authHandler)
		authGroup.GET("/github/login", authHandler)
		authGroup.GET("/github/logout", authHandler)
		authGroup.GET("/github/register", authHandler)
		authGroup.GET("/github/callback", authHandler)
		authGroup.POST("/tokens", GenerateToken)
		authGroup.GET("/tokens", GetTokens)
		authGroup.GET("/tokens/:id", GetTokens)
		authGroup.DELETE("/tokens/:id", DeleteToken)
	}
}

//GenerateToken generates token from context
func GenerateToken(c *gin.Context) {
	var currentUser *User

	if accessToken, ok := c.GetQuery("access_token"); ok {
		githubUser, err := GetGithubUser(accessToken)
		if err != nil {
			err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
			log.Info(c.ClientIP(), " ", err.Error())
			return
		}
		user := User{}
		err = Auth.GetDB(c.Request).
			Joins("left join auth_identities on users.id = auth_identities.user_id").
			Where("auth_identities.uid = ?", githubUser.GetID()).
			Find(&user).Error
		if err != nil {
			err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
			log.Info(c.ClientIP(), " ", err.Error())
			return
		}
		currentUser = &user
	} else {
		Handler(c)
		if c.IsAborted() {
			return
		}
		currentUser = GetCurrentUser(c.Request)
		if currentUser == nil {
			err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
			log.Info(c.ClientIP(), " ", err.Error())
			return
		}
	}

	tokenRequest := struct {
		Name        string `json:"name,omitempty"`
		VirtualUser string `json:"virtual_user,omitempty"`
	}{Name: "generated"}

	if c.Request.Method == http.MethodPost && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&tokenRequest); err != nil {
			err := c.AbortWithError(http.StatusBadRequest, err)
			log.Info(c.ClientIP(), " ", err.Error())
			return
		}
	}

	isForVirtualUser := tokenRequest.VirtualUser != ""

	userID := currentUser.IDString()
	userLogin := currentUser.Login
	tokenType := DroneUserTokenType
	if isForVirtualUser {
		userID = tokenRequest.VirtualUser
		userLogin = tokenRequest.VirtualUser
		tokenType = DroneHookTokenType
	}

	tokenID, signedToken, err := createAndStoreAPIToken(userID, userLogin, tokenType, tokenRequest.Name)

	if err != nil {
		err = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("%s", err))
		log.Info(c.ClientIP(), " ", err.Error())
		return
	}

	if isForVirtualUser {
		orgName := GetOrgNameFromVirtualUser(tokenRequest.VirtualUser)
		organization := Organization{Name: orgName}
		err = Auth.GetDB(c.Request).
			Model(currentUser).
			Where(&organization).
			Related(&organization, "Organizations").Error
		if err != nil {
			statusCode := GormErrorToStatusCode(err)
			err = c.AbortWithError(statusCode, err)
			log.Info(c.ClientIP(), " ", err.Error())
			return
		}

		AddDefaultRoleForVirtualUser(userID)
		AddOrgRoleForUser(userID, organization.ID)
	}

	c.JSON(http.StatusOK, gin.H{"id": tokenID, "token": signedToken})
}

func createAndStoreAPIToken(userID string, userLogin string, tokenType DroneTokenType, tokenName string) (string, string, error) {
	tokenID := uuid.NewV4().String()

	// Create the Claims
	claims := &ScopedClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    JwtIssuer,
			Audience:  JwtAudience,
			IssuedAt:  jwt.TimeFunc().Unix(),
			ExpiresAt: 0,
			Subject:   userID,
			Id:        tokenID,
		},
		Scope: "api:invoke", // "scope" for Pipeline
		Type:  tokenType,    // "type" for Drone
		Text:  userLogin,    // "text" for Drone
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := jwtToken.SignedString([]byte(signingKeyBase32))
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to sign user token")
	}

	token := NewToken(tokenID, tokenName)
	err = tokenStore.Store(userID, token)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to store user token")
	}

	return tokenID, signedToken, nil
}

// GetTokens returns the calling user's access tokens
func GetTokens(c *gin.Context) {
	currentUser := GetCurrentUser(c.Request)
	if currentUser == nil {
		err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
		log.Info(c.ClientIP(), " ", err.Error())
		return
	}
	tokenID := c.Param("id")

	if tokenID == "" {
		tokens, err := tokenStore.List(currentUser.IDString())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, tokens)
		}
	} else {
		token, err := tokenStore.Lookup(currentUser.IDString(), tokenID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		} else if token != nil {
			c.JSON(http.StatusOK, token)
		} else {
			c.AbortWithStatusJSON(http.StatusNotFound, btype.ErrorResponse{
				Code:    http.StatusNotFound,
				Message: "Token not found",
				Error:   "Token not found",
			})
		}
	}
}

// DeleteToken deletes the calling user's access token specified by token id
func DeleteToken(c *gin.Context) {
	currentUser := GetCurrentUser(c.Request)
	if currentUser == nil {
		err := c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
		log.Info(c.ClientIP(), err.Error())
		return
	}
	tokenID := c.Param("id")

	if tokenID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, fmt.Errorf("Missing token id"))
	} else {
		err := tokenStore.Revoke(currentUser.IDString(), tokenID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		} else {
			c.Status(http.StatusNoContent)
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

	var claims ScopedClaims
	accessToken, err := jwtRequest.ParseFromRequest(c.Request, jwtRequest.OAuth2Extractor, hmacKeyFunc, jwtRequest.WithClaims(&claims))

	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized,
			btype.ErrorResponse{
				Code:    http.StatusUnauthorized,
				Message: "Invalid token",
				Error:   err.Error(),
			})
		log.Info("Invalid token: ", err)
		return
	}

	isTokenWhitelisted, err := isTokenWhitelisted(&claims)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			btype.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Failed to validate user token",
				Error:   err.Error(),
			})
		log.Info("Failed to lookup user token: ", err)
		return
	}

	if !accessToken.Valid || !isTokenWhitelisted {
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
}

func saveUserIntoContext(c *gin.Context, claims *ScopedClaims) {
	userID, _ := strconv.ParseUint(claims.Subject, 10, 32)
	user := &User{
		ID:      uint(userID),
		Login:   claims.Text, // This is needed for Drone virtual user tokens
		Virtual: claims.Type == DroneHookTokenType,
	}
	newContext := context.WithValue(c.Request.Context(), auth.CurrentUser, user)
	c.Request = c.Request.WithContext(newContext)
}

//BanzaiSessionStorer stores the banzai session
type BanzaiSessionStorer struct {
	auth.SessionStorer
}

//Update updates the BanzaiSessionStorer
func (sessionStorer *BanzaiSessionStorer) Update(w http.ResponseWriter, req *http.Request, claims *claims.Claims) error {
	token := sessionStorer.SignedToken(claims)
	err := sessionStorer.SessionManager.Add(w, req, sessionStorer.SessionName, token)
	if err != nil {
		log.Info(req.RemoteAddr, err.Error())
		return err
	}

	// Set the drone cookie as well, but that cookie's value is actually a Pipeline API token
	currentUser := GetCurrentUser(req)
	if currentUser == nil {
		return fmt.Errorf("Can't get current user")
	}

	_, droneToken, err := createAndStoreAPIToken(claims.UserID, currentUser.Login, DroneUserTokenType, "Drone session token")
	if err != nil {
		log.Info(req.RemoteAddr, err.Error())
		return err
	}
	SetCookie(w, req, DroneSessionCookie, droneToken)
	return nil
}

// BanzaiLogoutHandler does the qor/auth DefaultLogoutHandler default logout behaviour + deleting the Drone cookie
func BanzaiLogoutHandler(context *auth.Context) {
	DelCookie(context.Writer, context.Request, DroneSessionCookie)
	auth.DefaultLogoutHandler(context)
}

func GetOrgNameFromVirtualUser(virtualUser string) string {
	return strings.Split(virtualUser, "/")[0]
}
