package auth

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

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
	Auth     *AuthHandler
	Claims   *Claims
	Provider Provider
	Request  *http.Request
	Writer   http.ResponseWriter
}

// Flashes get flash messages
func (context Context) Flashes() ([]Message, error) {
	return context.Auth.SessionStorer.Flashes(context.Writer, context.Request)
}

// FormValue get form value with name
func (context Context) FormValue(name string) string {
	return context.Request.Form.Get(name)
}

// AuthHandler auth struct
type AuthHandler struct {
	*AuthHandlerConfig
	// Embed SessionStorer to match Authority's AuthInterface
	SessionStorer *BanzaiSessionStorer
	provider      Provider
}

// AuthHandlerConfig auth config
type AuthHandlerConfig struct {
	// Default Database, which will be used in Auth when do CRUD, you can change a request's DB isntance by setting request Context's value, refer https://github.com/qor/auth/blob/master/utils.go#L32
	DB *gorm.DB
	// AuthIdentityModel a model used to save auth info, like email/password, OAuth token, linked user's ID, https://github.com/qor/auth/blob/master/auth_identity/auth_identity.go is the default implemention
	AuthIdentityModel interface{}
	// UserModel should be point of user struct's instance, it could be nil, then Auth will assume there is no user linked to auth info, and will return current auth info when get current user
	UserModel interface{}
	// Mount Auth into router with URLPrefix's value as prefix, default value is `/auth`.
	URLPrefix string

	// UserStorer is an interface that defined how to get/save user, Auth provides a default one based on AuthIdentityModel, UserModel's definition
	UserStorer BanzaiUserStorer
	// SessionStorer is an interface that defined how to encode/validate/save/destroy session data and flash messages between requests, Auth provides a default method do the job, to use the default value, don't forgot to mount SessionManager's middleware into your router to save session data correctly. refer [session](https://github.com/qor/session) for more details
	SessionStorer *BanzaiSessionStorer
	// Redirector redirect user to a new page after registered, logged, confirmed...
	Redirector RedirectorInterface

	// LoginHandler defined behaviour when request `{Auth Prefix}/login`, default behaviour defined in http://godoc.org/github.com/qor/auth#pkg-variables
	LoginHandler func(*Context, func(*Context) (*Claims, error))
	// RegisterHandler defined behaviour when request `{Auth Prefix}/register`, default behaviour defined in http://godoc.org/github.com/qor/auth#pkg-variables
	RegisterHandler func(*Context, func(*Context) (*Claims, error))
	// LogoutHandler defined behaviour when request `{Auth Prefix}/logout`, default behaviour defined in http://godoc.org/github.com/qor/auth#pkg-variables
	LogoutHandler func(*Context)
	// DeregisterHandler defined behaviour when request `{Auth Prefix}/deregister`, default behaviour defined in http://godoc.org/github.com/qor/auth#pkg-variables
	DeregisterHandler func(*Context)

	provider Provider
}

// New initialize Auth
func New(config *AuthHandlerConfig) *AuthHandler {
	if config == nil {
		config = &AuthHandlerConfig{}
	}

	if config.URLPrefix == "" {
		config.URLPrefix = "/auth/"
	} else {
		config.URLPrefix = fmt.Sprintf("/%v/", strings.Trim(config.URLPrefix, "/"))
	}

	auth := &AuthHandler{AuthHandlerConfig: config, SessionStorer: config.SessionStorer, provider: config.provider}

	return auth
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

type RedirectorInterface interface {
	Redirect(http.ResponseWriter, *http.Request, string)
}

// SessionManagerInterface session manager interface
type SessionManagerInterface interface {
	// Add value to session data, if value is not string, will marshal it into JSON encoding and save it into session data.
	Add(w http.ResponseWriter, req *http.Request, key string, value interface{}) error
	// Get value from session data
	Get(req *http.Request, key string) string
	// Pop value from session data
	Pop(w http.ResponseWriter, req *http.Request, key string) string

	// Flash add flash message to session data
	Flash(w http.ResponseWriter, req *http.Request, message Message) error
	// Flashes returns a slice of flash messages from session data
	Flashes(w http.ResponseWriter, req *http.Request) ([]Message, error)

	// Load get value from session data and unmarshal it into result
	Load(req *http.Request, key string, result interface{}) error
	// PopLoad pop value from session data and unmarshal it into result
	PopLoad(w http.ResponseWriter, req *http.Request, key string, result interface{}) error

	// Middleware returns a new session manager middleware instance.
	Middleware(http.Handler) http.Handler
}

// Message message struct
type Message struct {
	Type string
}

// SessionStorerInterface session storer interface for Auth
type SessionStorerInterface interface {
	// Get get claims from request
	Get(req *http.Request) (*Claims, error)
	// Update update claims with session manager
	Update(w http.ResponseWriter, req *http.Request, claims *Claims) error
	// Delete delete session
	Delete(w http.ResponseWriter, req *http.Request) error

	// Flash add flash message to session data
	Flash(w http.ResponseWriter, req *http.Request, message Message) error
	// Flashes returns a slice of flash messages from session data
	Flashes(w http.ResponseWriter, req *http.Request) []Message

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

// Delete delete claims from session manager
func (sessionStorer *SessionStorer) Delete(w http.ResponseWriter, req *http.Request) error {
	sessionStorer.SessionManager.Pop(w, req, sessionStorer.SessionName)
	return nil
}

// Flash add flash message to session data
func (sessionStorer *SessionStorer) Flash(w http.ResponseWriter, req *http.Request, message Message) error {
	return sessionStorer.SessionManager.Flash(w, req, message)
}

// Flashes returns a slice of flash messages from session data
func (sessionStorer *SessionStorer) Flashes(w http.ResponseWriter, req *http.Request) ([]Message, error) {
	return sessionStorer.SessionManager.Flashes(w, req)
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

// NewServeMux generate http.Handler for auth
func (auth *AuthHandler) NewServeMux() http.Handler {
	return &serveMux{auth: auth}
}

type serveMux struct {
	auth *AuthHandler
}

// ServeHTTP dispatches the handler registered in the matched route
func (serveMux *serveMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var (
		claims  *Claims
		reqPath = strings.TrimPrefix(req.URL.Path, serveMux.auth.URLPrefix)
		paths   = strings.Split(reqPath, "/")
		context = &Context{Auth: serveMux.auth, Claims: claims, Request: req, Writer: w}
		path    string
	)

	provider := serveMux.auth.provider
	context.Provider = provider

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

// AuthURL generate URL for auth
func (auth *AuthHandler) AuthURL(pth string) string {
	return path.Join(auth.URLPrefix, pth)
}
