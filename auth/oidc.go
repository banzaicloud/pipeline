// Copyright © 2019 Banzai Cloud
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
	gocontext "context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	"github.com/qor/qor/utils"
	"golang.org/x/oauth2"
)

// OIDCProvider provide login with OIDC auth method
type OIDCProvider struct {
	*OIDCConfig
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
}

type AuthorizeHandler func(*auth.Context) (*claims.Claims, error)

// OIDCConfig holds the oidc configuration parameters
type OIDCConfig struct {
	PublicClientID     string
	ClientID           string
	ClientSecret       string
	IssuerURL          string
	InsecureSkipVerify bool
	RedirectURL        string
	Scopes             []string
	AuthorizeHandler   AuthorizeHandler
}

type IDTokenClaims struct {
	Subject         string            `json:"sub"`
	Name            string            `json:"name"`
	Email           string            `json:"email"`
	Verified        bool              `json:"email_verified"`
	Groups          []string          `json:"groups"`
	FederatedClaims map[string]string `json:"federated_claims"`
}

func newOIDCProvider(config *OIDCConfig) *OIDCProvider {
	if config == nil {
		config = &OIDCConfig{}
	}

	provider := &OIDCProvider{OIDCConfig: config}

	if config.ClientID == "" {
		panic(errors.New("OIDC's ClientID can't be blank"))
	}

	if config.ClientSecret == "" {
		panic(errors.New("OIDC's ClientSecret can't be blank"))
	}

	if config.IssuerURL == "" {
		panic(errors.New("OIDC's IssuerURL can't be blank"))
	}

	if config.Scopes == nil {
		config.Scopes = []string{oidc.ScopeOpenID, "profile", "email", "groups", "federated:id", oidc.ScopeOfflineAccess}
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
	oidcProvider, err := oidc.NewProvider(ctx, provider.IssuerURL)
	if err != nil {
		panic(fmt.Errorf("Failed to query provider %q: %s", provider.IssuerURL, err.Error()))
	}

	provider.provider = oidcProvider
	provider.verifier = oidcProvider.Verifier(&oidc.Config{ClientID: config.ClientID})

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
				ok           bool
			)

			verifier := provider.verifier
			ctx := oidc.ClientContext(req.Context(), &httpClient)
			oauth2Config := provider.OAuthConfig(context)

			switch req.Method {
			case "GET":
				// Authorization redirect callback from OAuth2 auth flow.
				if errMsg := req.FormValue("error"); errMsg != "" {
					err = errors.New(errMsg + ": " + req.FormValue("error_description"))
					return nil, err
				}

				code := req.FormValue("code")
				if code == "" {
					err = fmt.Errorf("no code in request: %q", req.Form)
					return nil, err
				}
				state := req.FormValue("state")

				var claims *claims.Claims

				claims, err = context.Auth.SessionStorer.ValidateClaims(state)
				if err != nil {
					err = fmt.Errorf("failed to validate state claims: %s", err.Error())
					return nil, err
				}

				if err := claims.Valid(); err != nil {
					err = fmt.Errorf("failed to validate state claims: %s", err.Error())
					return nil, err
				}

				if claims.Subject != "state" {
					err = fmt.Errorf("state parameter doesn't match: %s", claims.Subject)
					return nil, err
				}

				token, err = oauth2Config.Exchange(ctx, code)
				if err != nil {
					err = fmt.Errorf("failed to get token: %s", err.Error())
					return nil, err
				}

				rawIDToken, ok = token.Extra("id_token").(string)
				if !ok {
					err = fmt.Errorf("no id_token in token response")
					return nil, err
				}

			case "POST":
				// The Banzai CLI can send an id_token that it has requested from Dex
				// we may consume that as well in a POST request.
				rawIDToken = req.FormValue("id_token")
				if rawIDToken != "" {
					// The public CLI client's verifier is needed in this case
					verifier = oidcProvider.Verifier(&oidc.Config{ClientID: config.PublicClientID})

				} else {
					// Form request from frontend to refresh a token.
					refresh := req.FormValue("refresh_token")
					if refresh == "" {
						err = fmt.Errorf("no refresh_token in request: %q", req.Form)
						return nil, err
					}

					t := &oauth2.Token{
						RefreshToken: refresh,
						Expiry:       time.Now().Add(-time.Hour),
					}

					token, err = oauth2Config.TokenSource(ctx, t).Token()
					if err != nil {
						err = fmt.Errorf("failed to get token: %s", err.Error())
						return nil, err
					}

					rawIDToken, ok = token.Extra("id_token").(string)
					if !ok {
						err = fmt.Errorf("no id_token in token response")
						return nil, err
					}
				}
			default:
				err = fmt.Errorf("method not implemented: %s", req.Method)
				return nil, err
			}

			idToken, err := verifier.Verify(req.Context(), rawIDToken)
			if err != nil {
				err = fmt.Errorf("Failed to verify ID token: %s", err.Error())
				return nil, err
			}

			var claims IDTokenClaims
			err = idToken.Claims(&claims)
			if err != nil {
				err = fmt.Errorf("failed to parse claims: %s", err.Error())
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
				claims := authInfo.ToClaims()
				return claims, SaveOAuthRefreshToken(claims.UserID, token.RefreshToken)
			}

			// Create a new account otherwise
			context.Request = req.WithContext(gocontext.WithValue(req.Context(), SignUp, true))

			{
				schema.Provider = authInfo.Provider
				schema.UID = claims.Subject
				schema.Name = claims.Name
				schema.Email = claims.Email
				schema.RawInfo = &claims
			}
			if _, userID, err := context.Auth.UserStorer.Save(&schema, context); err == nil {
				if userID != "" {
					authInfo.UserID = userID
				}
			} else {
				return nil, err
			}

			if err = tx.Where(authInfo).FirstOrCreate(authIdentity).Error; err == nil {
				claims := authInfo.ToClaims()
				return claims, SaveOAuthRefreshToken(claims.UserID, token.RefreshToken)
			}

			return nil, err
		}
	}

	return provider
}

// GetName return provider name
func (OIDCProvider) GetName() string {
	return "dex"
}

// ConfigAuth config auth
func (provider OIDCProvider) ConfigAuth(*auth.Auth) {
}

// OAuthConfig return oauth config based on configuration
func (provider OIDCProvider) OAuthConfig(context *auth.Context) *oauth2.Config {
	var (
		config = provider.OIDCConfig
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
func (provider OIDCProvider) Login(context *auth.Context) {
	claims := claims.Claims{}
	claims.Subject = "state"
	signedToken := context.Auth.SessionStorer.SignedToken(&claims)

	url := provider.OAuthConfig(context).AuthCodeURL(signedToken)
	http.Redirect(context.Writer, context.Request, url, http.StatusFound)
}

// RedeemRefreshToken plays an OAuth redeem refresh token flow
// https://www.oauth.com/oauth2-servers/access-tokens/refreshing-access-tokens/
func (provider OIDCProvider) RedeemRefreshToken(context *auth.Context, refreshToken string) (*IDTokenClaims, *oauth2.Token, error) {
	token, err := provider.OAuthConfig(context).TokenSource(gocontext.Background(), &oauth2.Token{RefreshToken: refreshToken}).Token()
	if err != nil {
		return nil, nil, err
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, nil, fmt.Errorf("no id_token in token response")
	}

	var claims IDTokenClaims
	idToken, err := provider.verifier.Verify(gocontext.Background(), rawIDToken)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to verify ID token: %s", err.Error())
	}

	err = idToken.Claims(&claims)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse claims: %s", err.Error())
	}

	return &claims, token, nil
}

// Logout implemented logout with dex provider
func (OIDCProvider) Logout(context *auth.Context) {
}

// Register implemented register with dex provider
func (provider OIDCProvider) Register(context *auth.Context) {
	provider.Login(context)
}

// Deregister implemented deregister with dex provider
func (provider OIDCProvider) Deregister(context *auth.Context) {
	context.Auth.DeregisterHandler(context)
}

// Callback implement Callback with dex provider
func (provider OIDCProvider) Callback(context *auth.Context) {
	context.Auth.LoginHandler(context, provider.AuthorizeHandler)
}

// ServeHTTP implement ServeHTTP with dex provider
func (OIDCProvider) ServeHTTP(*auth.Context) {
}

// OIDCOrganizationSyncer synchronizes organizations of a user from an OIDC ID token.
type OIDCOrganizationSyncer struct {
	organizationSyncer OrganizationSyncer
}

// NewOIDCOrganizationSyncer returns a new OIDCOrganizationSyncer.
func NewOIDCOrganizationSyncer(organizationSyncer OrganizationSyncer) OIDCOrganizationSyncer {
	return OIDCOrganizationSyncer{
		organizationSyncer: organizationSyncer,
	}
}

// SyncOrganizations synchronizes organization membership for a user based on the OIDC ID token.
func (s OIDCOrganizationSyncer) SyncOrganizations(ctx gocontext.Context, user User, idTokenClaims *IDTokenClaims) error {
	organizations := make(map[string][]string)

	for _, group := range idTokenClaims.Groups {
		// get the part before :, that will be the organization name
		s := strings.SplitN(group, ":", 2)
		if len(s) < 1 {
			return errors.New("invalid group")
		}

		if _, ok := organizations[s[0]]; !ok {
			organizations[s[0]] = make([]string, 0)
		}

		if len(s) > 1 && s[1] != "" {
			organizations[s[0]] = append(organizations[s[0]], s[1])
		}
	}

	var upstreamMemberships []UpstreamOrganizationMembership
	for org, groups := range organizations {
		membership := UpstreamOrganizationMembership{
			Organization: UpstreamOrganization{
				Name:     org,
				Provider: idTokenClaims.FederatedClaims["connector_id"],
			},
			Role: RoleMember,
		}

		if idTokenClaims.FederatedClaims["connector_id"] == ProviderGithub || idTokenClaims.FederatedClaims["connector_id"] == ProviderGitlab {
			membership.Role = RoleAdmin
		} else {
			// TODO: add role group binding
			for _, group := range groups {
				if roleLevelMap[membership.Role] < roleLevelMap[group] {
					membership.Role = group
				}
			}
		}

		upstreamMemberships = append(upstreamMemberships, membership)
	}

	// When a user registers a default organization is created in which he/she is admin
	upstreamMemberships = append(
		[]UpstreamOrganizationMembership{
			{
				Organization: UpstreamOrganization{
					Name:     user.Login,
					Provider: idTokenClaims.FederatedClaims["connector_id"],
				},
				Role: RoleAdmin,
			},
		},
		upstreamMemberships...
	)

	return s.organizationSyncer.SyncOrganizations(ctx, user, upstreamMemberships)
}
