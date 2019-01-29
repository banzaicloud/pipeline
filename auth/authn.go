// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"encoding/base32"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	bauth "github.com/banzaicloud/bank-vaults/pkg/auth"
	"github.com/banzaicloud/pipeline/config"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/utils"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	"github.com/qor/auth/providers/dex"
	"github.com/qor/session"
	"github.com/qor/session/gorilla"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// PipelineSessionCookie holds the name of the Cookie Pipeline sets in the browser
const PipelineSessionCookie = "_banzai_session"

// CICDSessionCookie holds the name of the Cookie CICD sets in the browser
const CICDSessionCookie = "user_sess"

// CICDUserTokenType is the CICD token type used for API sessions
const CICDUserTokenType bauth.TokenType = "user"

// CICDHookTokenType is the CICD token type used for API sessions
const CICDHookTokenType bauth.TokenType = "hook"

// SessionCookieMaxAge holds long an authenticated session should be valid in seconds
const SessionCookieMaxAge = 30 * 24 * 60 * 60

// SessionCookieHTTPOnly describes if the cookies should be accessible from HTTP requests only (no JS)
const SessionCookieHTTPOnly = true

// SessionCookieName is the name of the token that is stored in the session cookie
const SessionCookieName = "Pipeline session token"

// Init authorization
var (
	log *logrus.Logger

	cicdDB *gorm.DB

	Auth *auth.Auth

	signingKey       string
	signingKeyBase32 string
	TokenStore       bauth.TokenStore

	// JwtIssuer ("iss") claim identifies principal that issued the JWT
	JwtIssuer string

	// JwtAudience ("aud") claim identifies the recipients that the JWT is intended for
	JwtAudience string

	// CookieDomain is the domain field for cookies
	CookieDomain string

	// Handler is the Gin authentication middleware
	Handler gin.HandlerFunc

	// SessionManager is responsible for handling browser session Cookies
	SessionManager session.ManagerInterface
)

// Simple init for logging
func init() {
	log = config.Logger()
}

// CICDClaims struct to store the cicd claim related things
type CICDClaims struct {
	*claims.Claims
	Type bauth.TokenType `json:"type,omitempty"`
	Text string          `json:"text,omitempty"`
}

func claimConverter(claims *bauth.ScopedClaims) interface{} {
	userID, _ := strconv.ParseUint(claims.Subject, 10, 32)
	return &User{
		ID:      uint(userID),
		Login:   claims.Text, // This is needed for CICD virtual user tokens
		Virtual: claims.Type == CICDHookTokenType,
	}
}

type cookieExtractor struct {
	sessionStorer *BanzaiSessionStorer
}

func (c cookieExtractor) ExtractToken(r *http.Request) (string, error) {
	return c.sessionStorer.SessionManager.Get(r, c.sessionStorer.SessionName), nil
}

type accessManager interface {
	GrantDefaultAccessToUser(userID string)
	GrantDefaultAccessToVirtualUser(userID string)
	AddOrganizationPolicies(orgID uint)
	GrantOganizationAccessToUser(userID string, orgID uint)
	RevokeOrganizationAccessFromUser(userID string, orgID uint)
	RevokeAllAccessFromUser(userID string)
}

type redirector struct {
}

func (redirector) Redirect(w http.ResponseWriter, req *http.Request, action string) {
	var url string
	if req.Context().Value(SignUp) != nil {
		url = viper.GetString("pipeline.signupRedirectPath")
	} else {
		url = viper.GetString("pipeline.uipath")
	}
	http.Redirect(w, req, url, http.StatusSeeOther)
}

// Init initializes the auth
func Init(db *gorm.DB, accessManager accessManager, githubImporter *GithubImporter) {
	JwtIssuer = viper.GetString("auth.jwtissuer")
	JwtAudience = viper.GetString("auth.jwtaudience")
	CookieDomain = viper.GetString("auth.cookieDomain")

	signingKey = viper.GetString("auth.tokensigningkey")
	if signingKey == "" {
		panic("Token signing key is missing from configuration")
	}
	if len(signingKey) < 32 {
		panic("Token signing key must be at least 32 characters")
	}

	signingKeyBytes := []byte(signingKey)
	signingKeyBase32 = base32.StdEncoding.EncodeToString(signingKeyBytes)

	cookieAuthenticationKey := signingKeyBytes
	cookieEncryptionKey := signingKeyBytes[:32]

	cookieStore := sessions.NewCookieStore(cookieAuthenticationKey, cookieEncryptionKey)
	cookieStore.Options.MaxAge = SessionCookieMaxAge
	cookieStore.Options.HttpOnly = SessionCookieHTTPOnly
	cookieStore.Options.Secure = viper.GetBool("auth.secureCookie")
	if viper.GetBool(config.SetCookieDomain) && CookieDomain != "" {
		cookieStore.Options.Domain = CookieDomain
	}

	SessionManager = gorilla.New(PipelineSessionCookie, cookieStore)

	cicdDB = db

	sessionStorer := &BanzaiSessionStorer{
		SessionStorer: auth.SessionStorer{
			SessionName:    "_auth_session",
			SessionManager: SessionManager,
			SigningMethod:  jwt.SigningMethodHS256,
			SignedString:   signingKeyBase32,
		},
	}

	// Initialize Auth with configuration
	Auth = auth.New(&auth.Config{
		DB:                config.DB(),
		Redirector:        redirector{},
		AuthIdentityModel: AuthIdentity{},
		UserModel:         User{},
		ViewPaths:         []string{"views"},
		SessionStorer:     sessionStorer,
		UserStorer: BanzaiUserStorer{
			signingKeyBase32: signingKeyBase32,
			cicdDB:           cicdDB,
			events:           ebAuthEvents{eb: config.EventBus},
			accessManager:    accessManager,
			githubImporter:   githubImporter,
		},
		LogoutHandler:     BanzaiLogoutHandler,
		DeregisterHandler: NewBanzaiDeregisterHandler(accessManager),
	})

	dexProvider := dex.New(&dex.Config{
		ClientID:     viper.GetString("auth.clientid"),
		ClientSecret: viper.GetString("auth.clientsecret"),
		IssuerURL:    "http://127.0.0.1:5556/dex",
	})
	// dexProvider.AuthorizeHandler = NewDexAuthorizeHandler(dexProvider)
	Auth.RegisterProvider(dexProvider)

	TokenStore = bauth.NewVaultTokenStore("pipeline")

	Handler = bauth.JWTAuth(TokenStore, signingKey, claimConverter, cookieExtractor{sessionStorer})
}

func StartTokenStoreGC() {
	ticker := time.NewTicker(time.Hour * 12)
	go func() {
		for tick := range ticker.C {
			_ = tick
			err := TokenStore.GC()
			if err != nil {
				errorHandler.Handle(errors.Wrap(err, "failed to garbage collect TokenStore"))
			} else {
				log.Info("TokenStore garbage collected")
			}
		}
	}()
}

// Install the whole OAuth and JWT Token based authn/authz mechanism to the specified Gin Engine.
func Install(engine *gin.Engine, generateTokenHandler gin.HandlerFunc) {

	// We have to make the raw net/http handlers a bit Gin-ish
	authHandler := gin.WrapH(Auth.NewServeMux())
	engine.Use(gin.WrapH(SessionManager.Middleware(utils.NopHandler{})))

	authGroup := engine.Group("/auth/")
	{
		authGroup.GET("/login", authHandler)
		authGroup.GET("/logout", authHandler)
		authGroup.GET("/register", authHandler)
		authGroup.POST("/deregister", authHandler)
		authGroup.GET("/github/login", authHandler)
		authGroup.GET("/github/logout", authHandler)
		authGroup.GET("/github/register", authHandler)
		authGroup.GET("/github/callback", authHandler)
		authGroup.GET("/dex/login", authHandler)
		authGroup.GET("/dex/logout", authHandler)
		authGroup.GET("/dex/register", authHandler)
		authGroup.GET("/dex/callback", authHandler)
		authGroup.POST("/tokens", generateTokenHandler)
		authGroup.GET("/tokens", GetTokens)
		authGroup.GET("/tokens/:id", GetTokens)
		authGroup.DELETE("/tokens/:id", DeleteToken)
	}
}

type tokenHandler struct {
	accessManager accessManager
}

func NewTokenHandler(accessManager accessManager) gin.HandlerFunc {
	handler := &tokenHandler{
		accessManager: accessManager,
	}

	return handler.GenerateToken
}

//GenerateToken generates token from context
func (h *tokenHandler) GenerateToken(c *gin.Context) {
	var currentUser *User

	if accessToken, ok := c.GetQuery("access_token"); ok {
		githubUser, err := getGithubUser(accessToken)
		if err != nil {
			errorHandler.Handle(errors.Wrap(err, "failed to query GitHub user"))
			c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session"))
			return
		}
		user := User{}
		err = Auth.GetDB(c.Request).
			Joins("left join auth_identities on users.id = auth_identities.user_id").
			Where("auth_identities.uid = ?", githubUser.GetID()).
			Find(&user).Error
		if err != nil {
			if gorm.IsRecordNotFoundError(err) {
				c.Status(http.StatusUnauthorized)
			} else {
				errorHandler.Handle(errors.Wrap(err, "failed to query registered user"))
				c.Status(http.StatusInternalServerError)
			}
			return
		}
		currentUser = &user
	} else {
		Handler(c)
		if c.IsAborted() {
			return
		}
		currentUser = GetCurrentUser(c.Request)
	}

	tokenRequest := struct {
		Name        string     `json:"name,omitempty"`
		VirtualUser string     `json:"virtualUser,omitempty"`
		ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	}{Name: "generated"}

	if c.Request.Method == http.MethodPost && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&tokenRequest); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
	}

	isForVirtualUser := tokenRequest.VirtualUser != ""

	userID := currentUser.IDString()
	userLogin := currentUser.Login
	tokenType := CICDUserTokenType
	if isForVirtualUser {
		userID = tokenRequest.VirtualUser
		userLogin = tokenRequest.VirtualUser
		tokenType = CICDHookTokenType
	}

	tokenID, signedToken, err := createAndStoreAPIToken(userID, userLogin, tokenType, tokenRequest.Name, tokenRequest.ExpiresAt)

	if err != nil {
		err = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("%s", err))
		errorHandler.Handle(errors.Wrap(err, "failed to create and store API token"))
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
			errorHandler.Handle(errors.Wrap(err, "failed to query organization name for virtual user"))
			return
		}

		h.accessManager.GrantDefaultAccessToVirtualUser(userID)
		h.accessManager.GrantOganizationAccessToUser(userID, organization.ID)
	}

	c.JSON(http.StatusOK, gin.H{"id": tokenID, "token": signedToken})
}

func createAPIToken(userID string, userLogin string, tokenType bauth.TokenType, expiresAt *time.Time) (string, string, error) {
	tokenID := uuid.NewV4().String()

	var expiresAtUnix int64
	if expiresAt != nil {
		expiresAtUnix = expiresAt.Unix()
	}

	// Create the Claims
	claims := &bauth.ScopedClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    JwtIssuer,
			Audience:  JwtAudience,
			IssuedAt:  jwt.TimeFunc().Unix(),
			ExpiresAt: expiresAtUnix,
			Subject:   userID,
			Id:        tokenID,
		},
		Scope: "api:invoke", // "scope" for Pipeline
		Type:  tokenType,    // "type" for CICD
		Text:  userLogin,    // "text" for CICD
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := jwtToken.SignedString([]byte(signingKeyBase32))
	if err != nil {
		return "", "", errors.Wrap(err, "failed to sign user token")
	}

	return tokenID, signedToken, nil
}

func createAndStoreAPIToken(userID string, userLogin string, tokenType bauth.TokenType, tokenName string, expiresAt *time.Time) (string, string, error) {
	tokenID, signedToken, err := createAPIToken(userID, userLogin, tokenType, expiresAt)
	if err != nil {
		return "", "", err
	}

	token := bauth.NewToken(tokenID, tokenName)
	token.ExpiresAt = expiresAt
	err = TokenStore.Store(userID, token)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to store user token")
	}

	return tokenID, signedToken, nil
}

// GetTokens returns the calling user's access tokens
func GetTokens(c *gin.Context) {
	currentUser := GetCurrentUser(c.Request)
	tokenID := c.Param("id")

	if tokenID == "" {
		tokens, err := TokenStore.List(currentUser.IDString())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		} else {
			for _, token := range tokens {
				token.Value = ""
			}
			c.JSON(http.StatusOK, tokens)
		}
	} else {
		token, err := TokenStore.Lookup(currentUser.IDString(), tokenID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		} else if token != nil {
			token.Value = ""
			c.JSON(http.StatusOK, token)
		} else {
			c.AbortWithStatusJSON(http.StatusNotFound, pkgCommon.ErrorResponse{
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
	tokenID := c.Param("id")

	if tokenID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, fmt.Errorf("Missing token id"))
	} else {
		err := TokenStore.Revoke(currentUser.IDString(), tokenID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		} else {
			c.Status(http.StatusNoContent)
		}
	}
}

//BanzaiSessionStorer stores the banzai session
type BanzaiSessionStorer struct {
	auth.SessionStorer
}

//Update updates the BanzaiSessionStorer
func (sessionStorer *BanzaiSessionStorer) Update(w http.ResponseWriter, req *http.Request, claims *claims.Claims) error {

	// Get the current user object, in this early stage this is how to get it
	context := &auth.Context{Auth: Auth, Claims: claims, Request: req}
	user, err := Auth.UserStorer.Get(claims, context)
	if err != nil {
		return err
	}
	currentUser := user.(*User)
	if currentUser == nil {
		return fmt.Errorf("Can't get current user")
	}

	// These tokens are GCd after they expire
	expiresAt := time.Now().Add(SessionCookieMaxAge * time.Second)

	_, cookieToken, err := createAndStoreAPIToken(claims.UserID, currentUser.Login, CICDUserTokenType, SessionCookieName, &expiresAt)
	if err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed to create user session cookie"))
		return err
	}

	// Set the pipeline cookie
	err = sessionStorer.SessionManager.Add(w, req, sessionStorer.SessionName, cookieToken)
	if err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed to add user's session cookie to store"))
		return err
	}

	// Set the CICD cookie as well, but that cookie's value is actually a Pipeline API token
	SetCookie(w, req, CICDSessionCookie, cookieToken)

	return nil
}

// BanzaiLogoutHandler does the qor/auth DefaultLogoutHandler default logout behavior + deleting the CICD cookie
func BanzaiLogoutHandler(context *auth.Context) {
	DelCookie(context.Writer, context.Request, CICDSessionCookie)
	DelCookie(context.Writer, context.Request, PipelineSessionCookie)
}

type banzaiDeregisterHandler struct {
	accessManager accessManager
}

// NewBanzaiDeregisterHandler returns a handler that deletes the user and all his/her tokens from the database
func NewBanzaiDeregisterHandler(accessManager accessManager) func(*auth.Context) {
	handler := &banzaiDeregisterHandler{
		accessManager: accessManager,
	}

	return handler.handler
}

// BanzaiDeregisterHandler deletes the user and all his/her tokens from the database
func (h *banzaiDeregisterHandler) handler(context *auth.Context) {
	user := GetCurrentUser(context.Request)

	db := context.GetDB(context.Request)

	userAdminOrganizations := []UserOrganization{}

	// Query the organizations where the only admin is the current user.
	sql :=
		`SELECT * FROM user_organizations WHERE role = ? AND organization_id IN
		(SELECT DISTINCT organization_id FROM user_organizations WHERE user_id = ? AND role = ?)
		GROUP BY user_id, organization_id
		HAVING COUNT(*) = 1`

	if err := db.Raw(sql, "admin", user.ID, "admin").Scan(&userAdminOrganizations).Error; err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed select user only owned organizations"))
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// If there are any organizations with only this user as admin, throw an error
	if len(userAdminOrganizations) != 0 {
		orgs := []string{}
		for _, org := range userAdminOrganizations {
			orgs = append(orgs, fmt.Sprint(org.OrganizationID))
		}
		http.Error(context.Writer, "You must remove yourself or transfer ownership or delete these organizations before you can delete your user: "+strings.Join(orgs, ", "), http.StatusBadRequest)
		return
	}

	if err := db.Model(user).Association("Organizations").Clear().Error; err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed delete user's organization associations"))
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := db.Delete(user).Error; err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed delete user from DB"))
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	authIdentity := &AuthIdentity{Basic: auth_identity.Basic{UserID: user.IDString()}}
	if err := db.Delete(authIdentity).Error; err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed delete user's auth_identity from DB"))
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	cicdUser := CICDUser{Login: user.Login}
	// We need to pass cicdUser as well as the where clause, because Delete() filters by primary
	// key by default: http://doc.gorm.io/crud.html#delete but here we need to delete by the Login
	if err := cicdDB.Delete(cicdUser, cicdUser).Error; err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed delete user from CICD"))
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete Tokens
	tokens, err := TokenStore.List(user.IDString())
	if err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed list user's tokens during user deletetion"))
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, token := range tokens {
		err = TokenStore.Revoke(user.IDString(), token.ID)
		if err != nil {
			errorHandler.Handle(errors.Wrap(err, "failed remove user's tokens during user deletetion"))
			http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Delete Casbin roles
	h.accessManager.RevokeAllAccessFromUser(user.IDString())

	BanzaiLogoutHandler(context)
}

// GetOrgNameFromVirtualUser returns the organization name for which the virtual user has access
func GetOrgNameFromVirtualUser(virtualUser string) string {
	return strings.Split(virtualUser, "/")[0]
}

const internalUserLoginName = "internal"

func InternalUserHandler(ctx *gin.Context) {
	user, err := GetUserByLoginName(internalUserLoginName)
	if err != nil {
		err = errors.Wrap(err, "failed to retrieve internal user")
		errorHandler.Handle(err)
		http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
		ctx.Abort()
		return
	}
	newContext := context.WithValue(ctx.Request.Context(), bauth.CurrentUser, user)
	ctx.Request = ctx.Request.WithContext(newContext)
}
