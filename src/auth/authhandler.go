// Copyright © 2020 Banzai Cloud
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
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// Provider define Provider interface
type Provider interface {
	Login(*Context)
	Logout(*Context)
	Register(*Context)
	Deregister(*Context)
	Callback(*Context)
	ServeHTTP(*Context)
}

// Context context
type Context struct {
	Auth    *AuthHandler
	Claims  *Claims
	Request *http.Request
	Writer  http.ResponseWriter
}

// AuthHandler auth struct
type AuthHandler struct {
	*AuthHandlerConfig
	SessionStorer SessionStorerInterface
	Provider      Provider
}

// AuthHandlerConfig auth config
type AuthHandlerConfig struct {
	// Default Database, which will be used in Auth when do CRUD, you can change a request's DB isntance by setting request Context's value
	DB *gorm.DB

	// Mount Auth into router with URLPrefix's value as prefix, default value is `/auth`.
	URLPrefix string

	// UserStorer is an interface that defined how to get/save user, Auth provides a default one based on AuthIdentityModel, UserModel's definition
	UserStorer BanzaiUserStorer
	// SessionStorer is an interface that defined how to encode/validate/save/destroy session data between requests, Auth provides a default method do the job, to use the default value, don't forgot to mount SessionManager's middleware into your router to save session data correctly.
	SessionStorer SessionStorerInterface
	// Redirector redirect user to a new page after registered, logged, confirmed...
	Redirector Redirector

	// LoginHandler defined behaviour when request `{Auth Prefix}/login`
	LoginHandler func(*Context, func(*Context) (*Claims, error))
	// RegisterHandler defined behaviour when request `{Auth Prefix}/register`
	RegisterHandler func(*Context, func(*Context) (*Claims, error))
	// LogoutHandler defined behaviour when request `{Auth Prefix}/logout`
	LogoutHandler func(*Context)
	// DeregisterHandler defined behaviour when request `{Auth Prefix}/deregister`
	DeregisterHandler func(*Context)

	Provider Provider
}

// New initialize Auth
func New(config *AuthHandlerConfig) *AuthHandler {
	if config == nil {
		panic("config should be set")
	}

	if config.URLPrefix == "" {
		config.URLPrefix = "/auth/"
	} else {
		config.URLPrefix = fmt.Sprintf("/%v/", strings.Trim(config.URLPrefix, "/"))
	}

	return &AuthHandler{
		AuthHandlerConfig: config,
		SessionStorer:     config.SessionStorer,
		Provider:          config.Provider,
	}
}

// GetCurrentUser get current user from request
func (auth *AuthHandler) GetCurrentUser(req *http.Request) interface{} {
	if currentUser := req.Context().Value(CurrentUser); currentUser != nil {
		return currentUser
	}

	claims, err := auth.SessionStorer.Get(req)
	if err == nil {
		context := &Context{Auth: auth, Claims: claims, Request: req}
		if user, err := auth.UserStorer.Get(claims, context); err == nil {
			return user
		}
	}

	return nil
}

// Login sign user in
func (auth *AuthHandler) Login(w http.ResponseWriter, req *http.Request, claims *Claims) error {
	now := time.Now()
	claims.LastLoginAt = &now

	return auth.SessionStorer.Update(w, req, claims)
}

type Redirector interface {
	Redirect(http.ResponseWriter, *http.Request, string)
}

// SessionManagerInterface session manager interface
type SessionManagerInterface interface {
	// Add value to session data, if value is not string, will marshal it into JSON encoding and save it into session data.
	Add(w http.ResponseWriter, req *http.Request, key, value string) error
	// Get value from session data
	Get(req *http.Request, key string) string
}

// SessionStorerInterface session storer interface for Auth
type SessionStorerInterface interface {
	// Get get claims from request
	Get(req *http.Request) (*Claims, error)
	// Update update claims with session manager
	Update(w http.ResponseWriter, req *http.Request, claims *Claims) error

	// SignedToken generate signed token with Claims
	SignedToken(claims *Claims) (string, error)
	// ValidateClaims validate auth token
	ValidateClaims(tokenString string) (*Claims, error)
}

// SessionStorer default session storer
type SessionStorer struct {
	SessionName    string
	SigningMethod  jose.SignatureAlgorithm
	SignedString   string
	SessionManager SessionManagerInterface
}

// Get get claims from request
func (sessionStorer *SessionStorer) Get(req *http.Request) (*Claims, error) {
	tokenString := req.Header.Get("Authorization")

	// Get Token from Cookie
	if tokenString == "" {
		tokenString = sessionStorer.SessionManager.Get(req, sessionStorer.SessionName)
	}

	return sessionStorer.ValidateClaims(tokenString)
}

// Update update claims with session manager
func (sessionStorer *SessionStorer) Update(w http.ResponseWriter, req *http.Request, claims *Claims) error {
	token, err := sessionStorer.SignedToken(claims)
	if err != nil {
		return err
	}
	return sessionStorer.SessionManager.Add(w, req, sessionStorer.SessionName, token)
}

// SignedToken generate signed token with Claims
func (sessionStorer *SessionStorer) SignedToken(claims *Claims) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: sessionStorer.SigningMethod,
		Key:       []byte(sessionStorer.SignedString),
	}, nil)
	if err != nil {
		return "", err
	}

	return jwt.Signed(signer).Claims(claims).CompactSerialize()
}

// ValidateClaims validate auth token
func (sessionStorer *SessionStorer) ValidateClaims(tokenString string) (*Claims, error) {
	token, err := jwt.ParseSigned(tokenString)
	if err != nil {
		return nil, err
	}

	var claims Claims
	err = token.Claims([]byte(sessionStorer.SignedString), &claims)
	if err != nil {
		return nil, err
	}

	return &claims, claims.Validate(jwt.Expected{Time: time.Now()})
}

// HandlerFunc generate gin.HandlerFunc for auth
func (auth *AuthHandler) HandlerFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			w       = c.Writer
			req     = c.Request
			claims  *Claims
			reqPath = strings.TrimPrefix(req.URL.Path, auth.URLPrefix)
			paths   = strings.Split(reqPath, "/")
			context = &Context{
				Auth:    auth,
				Claims:  claims,
				Request: req,
				Writer:  w,
			}
			path string
		)

		provider := auth.Provider

		if len(paths) >= 2 {
			path = paths[1]
		} else if len(paths) == 1 {
			path = paths[0]
		}

		switch path {
		case "login":
			provider.Login(context)
		case "logout":
			provider.Logout(context)
		case "register":
			provider.Register(context)
		case "callback":
			provider.Callback(context)
		default:
			http.NotFound(w, req)
		}
	}
}

// AuthURL generate URL for auth
func (auth *AuthHandler) AuthURL(pth string) string {
	return path.Join(auth.URLPrefix, pth)
}
