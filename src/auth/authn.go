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

	"emperror.dev/emperror"
	"emperror.dev/errors"
	bauth "github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	ginauth "github.com/banzaicloud/gin-utilz/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	"github.com/qor/session"
	"github.com/qor/session/gorilla"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/global"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
)

// PipelineSessionCookie holds the name of the Cookie Pipeline sets in the browser
const PipelineSessionCookie = "_banzai_session"

// UserTokenType is the token type used for API sessions
const UserTokenType pkgAuth.TokenType = "user"

// VirtualUserTokenType is the token type used for API sessions by external services
// Used by PKE at the moment
// Legacy token type (used by CICD build hook originally)
const VirtualUserTokenType pkgAuth.TokenType = "hook"

// SessionCookieMaxAge holds long an authenticated session should be valid in seconds
const SessionCookieMaxAge = 30 * 24 * 60 * 60

// SessionCookieHTTPOnly describes if the cookies should be accessible from HTTP requests only (no JS)
const SessionCookieHTTPOnly = true

// SessionCookieName is the name of the token that is stored in the session cookie
const SessionCookieName = "Pipeline session token"

const BanzaiCLIClient = "banzai-cli"

// Init authorization
// nolint: gochecknoglobals
var (
	Auth *auth.Auth

	// CookieDomain is the domain field for cookies
	CookieDomain string

	// Handler is the Gin authentication middleware
	Handler gin.HandlerFunc

	// InternalHandler is the Gin authentication middleware for internal clients
	InternalHandler gin.HandlerFunc

	// SessionManager is responsible for handling browser session Cookies
	SessionManager session.ManagerInterface

	oidcProvider *OIDCProvider
)

// nolint: gochecknoglobals
var log logrus.FieldLogger

func init() {
	log = global.LogrusLogger()

	global.SubscribeLogrusLogger(func(l *logrus.Logger) {
		log = l
	})
}

type cookieExtractor struct {
	sessionStorer *BanzaiSessionStorer
}

func (c cookieExtractor) ExtractToken(r *http.Request) (string, error) {
	return c.sessionStorer.SessionManager.Get(r, c.sessionStorer.SessionName), nil
}

type redirector struct {
	loginUrl  string
	signupUrl string
}

func (r redirector) Redirect(w http.ResponseWriter, req *http.Request, action string) {
	var url string
	if req.Context().Value(SignUp) != nil {
		url = r.signupUrl
	} else {
		url = r.loginUrl
	}
	http.Redirect(w, req, url, http.StatusSeeOther)
}

// Init initializes the auth
func Init(db *gorm.DB, config Config, tokenStore bauth.TokenStore, tokenManager TokenManager, orgSyncer OIDCOrganizationSyncer, serviceAccountService ServiceAccountService) {
	CookieDomain = config.Cookie.Domain

	signingKey := config.Token.SigningKey
	signingKeyBytes := []byte(signingKey)

	cookieAuthenticationKey := signingKeyBytes
	cookieEncryptionKey := signingKeyBytes[:32]

	cookieStore := sessions.NewCookieStore(cookieAuthenticationKey, cookieEncryptionKey)
	cookieStore.Options.MaxAge = SessionCookieMaxAge
	cookieStore.Options.HttpOnly = SessionCookieHTTPOnly
	cookieStore.Options.Secure = config.Cookie.Secure
	if config.Cookie.SetDomain && CookieDomain != "" {
		cookieStore.Options.Domain = CookieDomain
	}

	SessionManager = gorilla.New(PipelineSessionCookie, cookieStore)

	sessionStorer := &BanzaiSessionStorer{
		SessionStorer: auth.SessionStorer{
			SessionName:    "_auth_session",
			SessionManager: SessionManager,
			SigningMethod:  jwt.SigningMethodHS256,
			SignedString:   base32.StdEncoding.EncodeToString(signingKeyBytes),
		},
		tokenManager: tokenManager,
	}

	// Initialize Auth with configuration
	Auth = auth.New(&auth.Config{
		DB: db,
		Redirector: redirector{
			loginUrl:  config.RedirectURL.Login,
			signupUrl: config.RedirectURL.Signup,
		},
		AuthIdentityModel: AuthIdentity{},
		UserModel:         User{},
		ViewPaths:         []string{"views"},
		SessionStorer:     sessionStorer,
		UserStorer: BanzaiUserStorer{
			db:        db,
			orgSyncer: orgSyncer,
		},
		LoginHandler:      banzaiLoginHandler,
		LogoutHandler:     banzaiLogoutHandler,
		RegisterHandler:   banzaiRegisterHandler,
		DeregisterHandler: NewBanzaiDeregisterHandler(tokenStore),
	})

	oidcProvider = newOIDCProvider(&OIDCProviderConfig{
		PublicClientID:     config.CLI.ClientID,
		ClientID:           config.OIDC.ClientID,
		ClientSecret:       config.OIDC.ClientSecret,
		IssuerURL:          config.OIDC.Issuer,
		InsecureSkipVerify: config.OIDC.Insecure,
	}, NewRefreshTokenStore(tokenStore))
	Auth.RegisterProvider(oidcProvider)

	Handler = ginauth.JWTAuthHandler(
		signingKey,
		func(claims *ginauth.ScopedClaims) interface{} {
			userID, _ := strconv.ParseUint(claims.Subject, 10, 32)

			return &User{
				ID:      uint(userID),
				Login:   claims.Text, // This is needed for virtual user tokens
				Virtual: claims.Type == ginauth.TokenType(VirtualUserTokenType),
			}
		},
		func(ctx context.Context, value interface{}) context.Context {
			return context.WithValue(ctx, auth.CurrentUser, value)
		},
		func(ctx context.Context) interface{} {
			return ctx.Value(auth.CurrentUser)
		},
		ginauth.TokenStoreOption(tokenStore),
		ginauth.ExtractorOption(cookieExtractor{sessionStorer}),
		ginauth.ErrorHandlerOption(emperror.MakeContextAware(errorHandler)),
	)

	InternalHandler = newInternalHandler(serviceAccountService)
}

func newInternalHandler(serviceAccountService ServiceAccountService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := serviceAccountService.ExtractServiceAccount(c.Request)
		if user != nil {
			newContext := context.WithValue(c.Request.Context(), auth.CurrentUser, user)
			c.Request = c.Request.WithContext(newContext)
		}
	}
}

func SyncOrgsForUser(
	organizationSyncer OIDCOrganizationSyncer,
	refreshTokenStore RefreshTokenStore,
	user *User,
	request *http.Request,
) error {
	refreshToken, err := refreshTokenStore.GetRefreshToken(user.IDString())
	if err != nil {
		return errors.WrapIf(err, "failed to fetch refresh token from Vault")
	}

	if refreshToken == "" {
		return errors.WrapIf(err, "no refresh token, please login again")
	}

	authContext := auth.Context{Auth: Auth, Request: request}
	idTokenClaims, token, err := oidcProvider.RedeemRefreshToken(&authContext, refreshToken)
	if err != nil {
		return errors.WrapIf(err, "failed to redeem user refresh token")
	}

	err = refreshTokenStore.SaveRefreshToken(user.IDString(), token.RefreshToken)
	if err != nil {
		return errors.WrapIf(err, "failed to save user refresh token")
	}

	return organizationSyncer.SyncOrganizations(request.Context(), *user, idTokenClaims)
}

func StartTokenStoreGC(tokenStore bauth.TokenStore) {
	ticker := time.NewTicker(time.Hour * 12)
	go func() {
		for tick := range ticker.C {
			_ = tick
			err := tokenStore.GC()
			if err != nil {
				errorHandler.Handle(errors.Wrap(err, "failed to garbage collect TokenStore"))
			} else {
				log.Info("TokenStore garbage collected")
			}
		}
	}()
}

// Install the whole OAuth and JWT Token based authn/authz mechanism to the specified Gin Engine.
func Install(engine *gin.Engine) {
	// We have to make the raw net/http handlers a bit Gin-ish
	authHandler := gin.WrapH(Auth.NewServeMux())
	engine.Use(gin.WrapH(SessionManager.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))))

	authGroup := engine.Group("/auth/")
	{
		authGroup.GET("/login", authHandler)
		authGroup.GET("/logout", authHandler)
		authGroup.GET("/register", authHandler)
		authGroup.POST("/deregister", authHandler)
		authGroup.GET("/dex/login", authHandler)
		authGroup.GET("/dex/logout", authHandler)
		authGroup.GET("/dex/register", authHandler)
		authGroup.GET("/dex/callback", authHandler)
		authGroup.POST("/dex/callback", authHandler)
	}
}

// BanzaiSessionStorer stores the banzai session
type BanzaiSessionStorer struct {
	auth.SessionStorer

	tokenManager TokenManager
}

// Update updates the BanzaiSessionStorer
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

	_, cookieToken, err := sessionStorer.tokenManager.GenerateToken(
		claims.UserID,
		&expiresAt,
		UserTokenType,
		currentUser.Login,
		SessionCookieName,
		false,
	)
	if err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed to create user session cookie"))
		return err
	}

	// Set the token as a pipeline session cookie
	err = sessionStorer.SessionManager.Add(w, req, sessionStorer.SessionName, cookieToken)
	if err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed to add user's session cookie to store"))
		return err
	}

	// Add the token in a header to the CLI
	if req.Header.Get("Client") == BanzaiCLIClient {
		w.Header().Add("Authorization", cookieToken)
	}

	return nil
}

func respondAfterLogin(claims *claims.Claims, context *auth.Context) {
	err := context.Auth.Login(context.Writer, context.Request, claims)
	if err != nil {
		httpJSONError(context.Writer, err, http.StatusUnauthorized)
		return
	}

	context.Auth.Redirector.Redirect(context.Writer, context.Request, "login")
}

func banzaiLoginHandler(context *auth.Context, authorize func(*auth.Context) (*claims.Claims, error)) {
	claims, err := authorize(context)
	if err == nil && claims != nil {
		respondAfterLogin(claims, context)
		return
	}

	httpJSONError(context.Writer, err, http.StatusUnauthorized)
}

// BanzaiLogoutHandler does the qor/auth DefaultLogoutHandler default logout behavior
func banzaiLogoutHandler(context *auth.Context) {
	DelCookie(context.Writer, context.Request, PipelineSessionCookie)
}

func banzaiRegisterHandler(context *auth.Context, register func(*auth.Context) (*claims.Claims, error)) {
	claims, err := register(context)
	if err == nil && claims != nil {
		respondAfterLogin(claims, context)
		return
	}

	httpJSONError(context.Writer, err, http.StatusUnauthorized)
}

type banzaiDeregisterHandler struct {
	tokenStore bauth.TokenStore
}

// NewBanzaiDeregisterHandler returns a handler that deletes the user and all his/her tokens from the database
func NewBanzaiDeregisterHandler(tokenStore bauth.TokenStore) func(*auth.Context) {
	handler := &banzaiDeregisterHandler{
		tokenStore: tokenStore,
	}

	return handler.handler
}

// BanzaiDeregisterHandler deletes the user and all his/her tokens from the database
func (h *banzaiDeregisterHandler) handler(context *auth.Context) {
	user := GetCurrentUser(context.Request)

	db := context.GetDB(context.Request)

	// Remove organization memberships
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

	// Delete Tokens
	tokens, err := h.tokenStore.List(user.IDString())
	if err != nil {
		errorHandler.Handle(errors.Wrap(err, "failed list user's tokens during user deletetion"))
		http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, token := range tokens {
		err = h.tokenStore.Revoke(user.IDString(), token.ID)
		if err != nil {
			errorHandler.Handle(errors.Wrap(err, "failed remove user's tokens during user deletetion"))
			http.Error(context.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	banzaiLogoutHandler(context)
}

// GetOrgNameFromVirtualUser returns the organization name for which the virtual user has access
func GetOrgNameFromVirtualUser(virtualUser string) string {
	return strings.Split(virtualUser, "/")[0]
}

const internalUserLogin = "internal"
const internalUserEmail = "internal@pipeline.banzaicloud.com"
const internalUserID = 99999
const internalUserName = "Internal user"

func InternalUserHandler(ctx *gin.Context) {
	user := &User{
		ID:    internalUserID,
		Name:  internalUserName,
		Email: internalUserEmail,
		Login: internalUserLogin,
	}
	newContext := context.WithValue(ctx.Request.Context(), auth.CurrentUser, user)
	ctx.Request = ctx.Request.WithContext(newContext)
}
