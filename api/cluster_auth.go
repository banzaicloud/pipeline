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

package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"emperror.dev/emperror"
	"github.com/coreos/go-oidc"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	k8sClient "k8s.io/client-go/tools/clientcmd"
	k8sClientApi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/banzaicloud/pipeline/api/common"
	"github.com/banzaicloud/pipeline/internal/cluster/auth"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

type ClusterAuthAPI struct {
	// Does the provider use "offline_access" scope to request a refresh token
	// or does it use "access_type=offline" (e.g. Google)?
	offlineAsScope bool

	redirectURI string

	tokenSigningKey []byte

	client *http.Client

	provider *oidc.Provider

	clusterGetter      common.ClusterGetter
	clusterAuthService auth.ClusterAuthService
}

func NewClusterAuthAPI(
	clusterGetter common.ClusterGetter,
	clusterAuthService auth.ClusterAuthService,
	tokenSigningKey string,
	issuerURL string,
	insecureSkipVerify bool,
	redirectURI string,
) (*ClusterAuthAPI, error) {

	httpClient := http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
		},
	}

	a := ClusterAuthAPI{
		tokenSigningKey:    []byte(tokenSigningKey),
		client:             &httpClient,
		clusterGetter:      clusterGetter,
		clusterAuthService: clusterAuthService,
		redirectURI:        redirectURI,
	}

	_, err := url.Parse(redirectURI)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to parse redirect-uri: %q", redirectURI)
	}

	ctx := oidc.ClientContext(context.Background(), a.client)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, emperror.Wrapf(err, "failed to query provider: %q", issuerURL)
	}

	var s struct {
		// What scopes does a provider support?
		//
		// See: https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata
		ScopesSupported []string `json:"scopes_supported"`
	}
	if err := provider.Claims(&s); err != nil {
		return nil, emperror.Wrap(err, "failed to parse provider scopes_supported")
	}

	if len(s.ScopesSupported) == 0 {
		// scopes_supported is a "RECOMMENDED" discovery claim, not a required
		// one. If missing, assume that the provider follows the spec and has
		// an "offline_access" scope.
		a.offlineAsScope = true
	} else {
		// See if scopes_supported has the "offline_access" scope.
		a.offlineAsScope = func() bool {
			for _, scope := range s.ScopesSupported {
				if scope == oidc.ScopeOfflineAccess {
					return true
				}
			}
			return false
		}()
	}

	a.provider = provider

	return &a, nil
}

func (api *ClusterAuthAPI) dexCallback(c *gin.Context) {

	var (
		err          error
		token        *oauth2.Token
		clusterID    uint
		clientID     string
		clientSecret string
	)

	r := c.Request
	ctx := oidc.ClientContext(r.Context(), api.client)

	switch r.Method {
	case "GET":
		// Authorization redirect callback from OAuth2 auth flow.
		if errMsg := r.FormValue("error"); errMsg != "" {
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("%s: %s", errMsg, r.FormValue("error_description")))
			return
		}
		code := r.FormValue("code")
		if code == "" {
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("no code in request: %q", r.Form))
			return
		}
		stateRaw := r.FormValue("state")
		if stateRaw == "" {
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("no state in request: %q", r.Form))
			return
		}

		// stateRaw parseJWT -> state
		stateClaims := stateClaims{}
		_, err = jwt.ParseWithClaims(stateRaw, &stateClaims, func(token *jwt.Token) (interface{}, error) {
			return api.tokenSigningKey, nil
		})

		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("failed to parse state token: %q", err.Error()))
			return
		}

		if err := stateClaims.Valid(); err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("state token is invalid: %q", err.Error()))
			return
		}

		var secret auth.ClusterClientSecret
		secret, err = api.clusterAuthService.GetClusterClientSecret(c.Request.Context(), stateClaims.ClusterID)
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("error getting cluster client secret: %q", err.Error()))
			return
		}

		if secret.ClientID != stateClaims.ClientID {
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("unexpected state, cluster clientID mismatch: %q", stateClaims.ClientID))
			return
		}

		clientID = secret.ClientID
		clientSecret = secret.ClientSecret
		clusterID = stateClaims.ClusterID

		oauth2Config := api.oauth2Config(secret, nil)

		token, err = oauth2Config.Exchange(ctx, code)
	case "POST":
		// Form request from frontend to refresh a token.
		refresh := r.FormValue("refresh_token")
		if refresh == "" {
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("no refresh_token in request: %q", r.Form))
			return
		}
		clientID = r.FormValue("client_id")
		if refresh == "" {
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("no client_id in request: %q", r.Form))
			return
		}
		clientSecret = r.FormValue("client_secret")
		if refresh == "" {
			_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("no client_secret in request: %q", r.Form))
			return
		}
		t := &oauth2.Token{
			RefreshToken: refresh,
			Expiry:       time.Now().Add(-time.Hour),
		}

		oauth2Config := api.oauth2Config(auth.ClusterClientSecret{ClientID: clientID, ClientSecret: clientSecret}, nil)

		token, err = oauth2Config.TokenSource(ctx, t).Token()
	default:
		_ = c.AbortWithError(http.StatusBadRequest, fmt.Errorf("method not implemented: %s", r.Method))
		return
	}

	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to get token: %q", err.Error()))
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		_ = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("no id_token in token response"))
		return
	}

	verifier := api.provider.Verifier(&oidc.Config{ClientID: clientID})

	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to verify ID token: %q", err.Error()))
		return
	}

	var claims claim
	err = idToken.Claims(&claims)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to parse claims: %q", err.Error()))
		return
	}

	configBuffer, err := api.generateKubeConfig(r.Context(), rawIDToken, token.RefreshToken, claims, clientID, clientSecret, clusterID)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to generate kubeconfig: %q", err.Error()))
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\"kubeconfig.yaml\"")
	c.Data(http.StatusOK, "application/x-yaml", configBuffer)
}

type stateClaims struct {
	ClusterID uint   `json:"clusterID"`
	ClientID  string `json:"clientID"`
	jwt.StandardClaims
}

func (api *ClusterAuthAPI) loginHandler(c *gin.Context) {
	var scopes []string

	cluster, ok := api.clusterGetter.GetClusterFromRequest(c)
	if !ok {
		return
	}

	secret, err := api.clusterAuthService.GetClusterClientSecret(c.Request.Context(), cluster.GetID())
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error getting cluster client secret",
			Error:   err.Error(),
		})
		return
	}

	// Create the stateClaims
	claims := stateClaims{
		cluster.GetID(),
		secret.ClientID,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
		},
	}

	stateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	state, err := stateToken.SignedString(api.tokenSigningKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error signing state token",
			Error:   err.Error(),
		})
		return
	}

	var authCodeURL string
	scopes = append(scopes, "groups", "openid", "profile", "email")
	if api.offlineAsScope {
		scopes = append(scopes, "offline_access")
		authCodeURL = api.oauth2Config(secret, scopes).AuthCodeURL(state)
	} else {
		authCodeURL = api.oauth2Config(secret, scopes).AuthCodeURL(state, oauth2.AccessTypeOffline)
	}

	c.Redirect(http.StatusSeeOther, authCodeURL)
}

type claim struct {
	AtHash        string `json:"at_hash"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	jwt.StandardClaims
}

func (api *ClusterAuthAPI) oauth2Config(clientSecret auth.ClusterClientSecret, scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientSecret.ClientID,
		ClientSecret: clientSecret.ClientSecret,
		Endpoint:     api.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  api.redirectURI,
	}
}

func (api *ClusterAuthAPI) generateKubeConfig(ctx context.Context, IDToken string, refreshToken string, claims claim, clientID, clientSecret string, clusterID uint) ([]byte, error) {

	config, err := api.clusterAuthService.GetClusterConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	authInfo := k8sClientApi.NewAuthInfo()

	authInfo.AuthProvider = &k8sClientApi.AuthProviderConfig{
		Name: "oidc",
		Config: map[string]string{
			"client-id":      clientID,
			"client-secret":  clientSecret,
			"id-token":       IDToken,
			"refresh-token":  refreshToken,
			"idp-issuer-url": claims.Issuer,
		},
	}

	config.AuthInfos = map[string]*k8sClientApi.AuthInfo{claims.Email: authInfo}

	currentContext := config.Contexts[config.CurrentContext]
	currentContext.AuthInfo = claims.Email

	newCurrentContext := fmt.Sprint(claims.Email, "@", currentContext.Cluster)
	config.Contexts[newCurrentContext] = currentContext

	delete(config.Contexts, config.CurrentContext)

	config.CurrentContext = newCurrentContext

	return k8sClient.Write(*config)
}

func (api *ClusterAuthAPI) RegisterRoutes(clusterRouter gin.IRouter, authRouter gin.IRouter) {
	clusterRouter.GET("/oidcconfig", api.loginHandler)
	authRouter.GET("/auth/dex/cluster/callback", api.dexCallback)
}
