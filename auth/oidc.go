package auth

import (
	gocontext "context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	oidc "github.com/coreos/go-oidc"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	"github.com/qor/qor/utils"
	"golang.org/x/oauth2"
)

// DexProvider provide login with dex method
type DexProvider struct {
	*DexConfig
	provider *oidc.Provider
}

// DexConfig is the dex Config
type DexConfig struct {
	PublicClientID     string
	ClientID           string
	ClientSecret       string
	IssuerURL          string
	InsecureSkipVerify bool
	RedirectURL        string
	Scopes             []string
	AuthorizeHandler   func(*auth.Context) (*claims.Claims, error)
}

func newDexProvider(config *DexConfig) *DexProvider {
	if config == nil {
		config = &DexConfig{}
	}

	provider := &DexProvider{DexConfig: config}

	if config.ClientID == "" {
		panic(errors.New("Dex's ClientID can't be blank"))
	}

	if config.ClientSecret == "" {
		panic(errors.New("Dex's ClientSecret can't be blank"))
	}

	if config.IssuerURL == "" {
		panic(errors.New("Dex's IssuerURL can't be blank"))
	}

	if config.Scopes == nil {
		config.Scopes = []string{oidc.ScopeOpenID, "profile", "email", "groups", "federated:id"}
	}

	httpClient := http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.InsecureSkipVerify,
			},
		},
	}
	ctx := oidc.ClientContext(gocontext.Background(), &httpClient)
	dexProvider, err := oidc.NewProvider(ctx, provider.IssuerURL)
	if err != nil {
		panic(fmt.Errorf("Failed to query provider %q: %s", provider.IssuerURL, err.Error()))
	}

	provider.provider = dexProvider

	if config.AuthorizeHandler == nil {

		config.AuthorizeHandler = func(context *auth.Context) (*claims.Claims, error) {
			var (
				schema       auth.Schema
				authInfo     auth_identity.Basic
				err          error
				rawIDToken   string
				token        *oauth2.Token
				authIdentity = reflect.New(utils.ModelType(context.Auth.Config.AuthIdentityModel)).Interface()
				req          = context.Request
				tx           = context.Auth.GetDB(req)
				w            = context.Writer
				ok           bool
			)

			verifier := dexProvider.Verifier(&oidc.Config{ClientID: config.ClientID})

			ctx := oidc.ClientContext(req.Context(), &httpClient)
			oauth2Config := provider.OAuthConfig(context)

			switch req.Method {
			case "GET":
				// Authorization redirect callback from OAuth2 auth flow.
				if errMsg := req.FormValue("error"); errMsg != "" {
					err = errors.New(errMsg + ": " + req.FormValue("error_description"))
					http.Error(w, err.Error(), http.StatusBadRequest)
					return nil, err
				}

				code := req.FormValue("code")
				if code == "" {
					err = fmt.Errorf("no code in request: %q", req.Form)
					http.Error(w, err.Error(), http.StatusBadRequest)
					return nil, err
				}
				state := req.FormValue("state")

				var claims *claims.Claims

				claims, err = context.Auth.SessionStorer.ValidateClaims(state)
				if err != nil {
					err = fmt.Errorf("failed to validate state claims: %s", err.Error())
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return nil, err
				}

				if err := claims.Valid(); err != nil {
					err = fmt.Errorf("failed to validate state claims: %s", err.Error())
					http.Error(w, err.Error(), http.StatusBadRequest)
					return nil, err
				}

				if claims.Subject != "state" {
					err = fmt.Errorf("state parameter doesn't match: %s", claims.Subject)
					http.Error(w, err.Error(), http.StatusBadRequest)
					return nil, err
				}

				token, err = oauth2Config.Exchange(ctx, code)
				if err != nil {
					err = fmt.Errorf("failed to get token: %s", err.Error())
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return nil, err
				}

				rawIDToken, ok = token.Extra("id_token").(string)
				if !ok {
					err = fmt.Errorf("no id_token in token response")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return nil, err
				}

			case "POST":
				// The Banzai CLI can send an id_token that it has requested from Dex
				// we may consume that as well in a POST request.
				rawIDToken = req.FormValue("id_token")
				if rawIDToken != "" {
					// The public CLI client's verifier is needed in this case
					verifier = dexProvider.Verifier(&oidc.Config{ClientID: config.PublicClientID})

				} else {
					// Form request from frontend to refresh a token.
					refresh := req.FormValue("refresh_token")
					if refresh == "" {
						err = fmt.Errorf("no refresh_token in request: %q", req.Form)
						http.Error(w, err.Error(), http.StatusBadRequest)
						return nil, err
					}

					t := &oauth2.Token{
						RefreshToken: refresh,
						Expiry:       time.Now().Add(-time.Hour),
					}

					token, err = oauth2Config.TokenSource(ctx, t).Token()
					if err != nil {
						err = fmt.Errorf("failed to get token: %s", err.Error())
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return nil, err
					}

					rawIDToken, ok = token.Extra("id_token").(string)
					if !ok {
						err = fmt.Errorf("no id_token in token response")
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return nil, err
					}
				}
			default:
				err = fmt.Errorf("method not implemented: %s", req.Method)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return nil, err
			}

			idToken, err := verifier.Verify(req.Context(), rawIDToken)
			if err != nil {
				err = fmt.Errorf("Failed to verify ID token: %s", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return nil, err
			}

			var claims struct {
				Subject         string            `json:"sub"`
				Name            string            `json:"name"`
				Email           string            `json:"email"`
				Verified        bool              `json:"email_verified"`
				Groups          []string          `json:"groups"`
				FederatedClaims map[string]string `json:"federated_claims"`
			}

			err = idToken.Claims(&claims)
			if err != nil {
				err = fmt.Errorf("failed to parse claims: %s", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return nil, err
			}

			// Check if authInfo exists with the backend connector already
			authInfo.Provider = claims.FederatedClaims["connector_id"]
			authInfo.UID = claims.FederatedClaims["user_id"]

			if !tx.Model(authIdentity).Where(authInfo).Scan(&authInfo).RecordNotFound() {
				return authInfo.ToClaims(), nil
			}

			// Check if authInfo exists with Dex
			authInfo.Provider = "dex:" + claims.FederatedClaims["connector_id"]
			authInfo.UID = claims.Subject

			if !tx.Model(authIdentity).Where(authInfo).Scan(&authInfo).RecordNotFound() {
				return authInfo.ToClaims(), nil
			}

			// Create a new account otherwise
			context.Request = req.WithContext(gocontext.WithValue(req.Context(), SignUp, true))

			{
				schema.Provider = authInfo.Provider
				schema.UID = claims.Subject
				schema.Name = claims.Name
				schema.Email = claims.Email
				schema.RawInfo = claims
			}
			if _, userID, err := context.Auth.UserStorer.Save(&schema, context); err == nil {
				if userID != "" {
					authInfo.UserID = userID
				}
			} else {
				return nil, err
			}

			if err = tx.Where(authInfo).FirstOrCreate(authIdentity).Error; err == nil {
				return authInfo.ToClaims(), nil
			}

			return nil, err
		}
	}

	return provider
}

// GetName return provider name
func (DexProvider) GetName() string {
	return "dex"
}

// ConfigAuth config auth
func (provider DexProvider) ConfigAuth(*auth.Auth) {
}

// OAuthConfig return oauth config based on configuration
func (provider DexProvider) OAuthConfig(context *auth.Context) *oauth2.Config {
	var (
		config = provider.DexConfig
		req    = context.Request
		scheme = req.URL.Scheme
	)

	if scheme == "" {
		if req.TLS == nil {
			scheme = "http://"
		} else {
			scheme = "https://"
		}
	}

	return &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     provider.provider.Endpoint(),
		RedirectURL:  scheme + req.Host + context.Auth.AuthURL("dex/callback"),
		Scopes:       config.Scopes,
	}
}

// Login implemented login with dex provider
func (provider DexProvider) Login(context *auth.Context) {
	claims := claims.Claims{}
	claims.Subject = "state"
	signedToken := context.Auth.SessionStorer.SignedToken(&claims)

	url := provider.OAuthConfig(context).AuthCodeURL(signedToken)
	http.Redirect(context.Writer, context.Request, url, http.StatusFound)
}

// Logout implemented logout with dex provider
func (DexProvider) Logout(context *auth.Context) {
}

// Register implemented register with dex provider
func (provider DexProvider) Register(context *auth.Context) {
	provider.Login(context)
}

// Deregister implemented deregister with dex provider
func (provider DexProvider) Deregister(context *auth.Context) {
	panic("Not implemented")
}

// Callback implement Callback with dex provider
func (provider DexProvider) Callback(context *auth.Context) {
	context.Auth.LoginHandler(context, provider.AuthorizeHandler)
}

// ServeHTTP implement ServeHTTP with dex provider
func (DexProvider) ServeHTTP(*auth.Context) {
}
