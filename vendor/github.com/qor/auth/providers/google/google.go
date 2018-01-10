package google

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	"github.com/qor/qor/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var UserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"

// GoogleProvider provide login with google method
type GoogleProvider struct {
	*Config
}

// Config google Config
type Config struct {
	ClientID         string
	ClientSecret     string
	AuthorizeURL     string
	TokenURL         string
	RedirectURL      string
	Scopes           []string
	AuthorizeHandler func(context *auth.Context) (*claims.Claims, error)
}

func New(config *Config) *GoogleProvider {
	if config == nil {
		config = &Config{}
	}

	provider := &GoogleProvider{Config: config}

	if config.ClientID == "" {
		panic(errors.New("Google's ClientID can't be blank"))
	}

	if config.ClientSecret == "" {
		panic(errors.New("Google's ClientSecret can't be blank"))
	}

	if config.AuthorizeURL == "" {
		config.AuthorizeURL = google.Endpoint.AuthURL
	}

	if config.TokenURL == "" {
		config.TokenURL = google.Endpoint.TokenURL
	}

	if len(config.Scopes) == 0 {
		config.Scopes = []string{"https://www.googleapis.com/auth/userinfo.email"}
	}

	if config.AuthorizeHandler == nil {
		config.AuthorizeHandler = func(context *auth.Context) (*claims.Claims, error) {
			var (
				req          = context.Request
				schema       auth.Schema
				authInfo     auth_identity.Basic
				tx           = context.Auth.GetDB(req)
				authIdentity = reflect.New(utils.ModelType(context.Auth.Config.AuthIdentityModel)).Interface()
			)

			state := req.URL.Query().Get("state")
			claims, err := context.Auth.SessionStorer.ValidateClaims(state)

			if err != nil || claims.Valid() != nil || claims.Subject != "state" {
				return nil, auth.ErrUnauthorized
			}

			if err == nil {
				oauthCfg := provider.OAuthConfig(context)
				tkn, err := oauthCfg.Exchange(oauth2.NoContext, req.URL.Query().Get("code"))

				if err != nil {
					return nil, err
				}

				{
					client := oauthCfg.Client(oauth2.NoContext, tkn)
					resp, err := client.Get(UserInfoURL)
					if err != nil {
						return nil, err
					}

					defer resp.Body.Close()
					body, _ := ioutil.ReadAll(resp.Body)
					userInfo := UserInfo{}
					json.Unmarshal(body, &userInfo)
					schema.Provider = provider.GetName()
					schema.UID = userInfo.Email
					schema.Email = userInfo.Email
					schema.FirstName = userInfo.GivenName
					schema.LastName = userInfo.FamilyName
					schema.Image = userInfo.Picture
					schema.Name = userInfo.Name
					schema.RawInfo = userInfo
				}

				authInfo.Provider = provider.GetName()
				authInfo.UID = schema.UID

				if !tx.Model(authIdentity).Where(authInfo).Scan(&authInfo).RecordNotFound() {
					return authInfo.ToClaims(), nil
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
			}

			return nil, err
		}
	}
	return provider
}

// GetName return provider name
func (GoogleProvider) GetName() string {
	return "google"
}

// ConfigAuth config auth
func (provider GoogleProvider) ConfigAuth(*auth.Auth) {
}

// OAuthConfig return oauth config based on configuration
func (provider GoogleProvider) OAuthConfig(context *auth.Context) *oauth2.Config {
	var (
		config = provider.Config
		scheme = context.Request.URL.Scheme
	)

	if scheme == "" {
		scheme = "http://"
	}

	return &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.AuthorizeURL,
			TokenURL: config.TokenURL,
		},
		RedirectURL: scheme + context.Request.Host + context.Auth.AuthURL("google/callback"),
		Scopes:      config.Scopes,
	}
}

// Login implemented login with google provider
func (provider GoogleProvider) Login(context *auth.Context) {
	claims := claims.Claims{}
	claims.Subject = "state"
	signedToken := context.Auth.SessionStorer.SignedToken(&claims)

	url := provider.OAuthConfig(context).AuthCodeURL(signedToken)
	http.Redirect(context.Writer, context.Request, url, http.StatusFound)
}

// Logout implemented logout with google provider
func (GoogleProvider) Logout(context *auth.Context) {
}

// Register implemented register with google provider
func (provider GoogleProvider) Register(context *auth.Context) {
	provider.Login(context)
}

// Callback implement Callback with google provider
func (provider GoogleProvider) Callback(context *auth.Context) {
	context.Auth.LoginHandler(context, provider.AuthorizeHandler)
}

// ServeHTTP implement ServeHTTP with google provider
func (GoogleProvider) ServeHTTP(*auth.Context) {
}

// UserInfo google user info structure
type UserInfo struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Profile       string `json:"profile"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Gender        string `json:"gender"`
}
