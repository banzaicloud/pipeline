package facebook

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
	"golang.org/x/oauth2/facebook"
)

var UserInfoURL = "https://graph.facebook.com/me?access_token="

// FacebookProvider provide login with facebook method
type FacebookProvider struct {
	*Config
}

// Config facebook Config
type Config struct {
	ClientID         string
	ClientSecret     string
	AuthorizeURL     string
	RedirectURL      string
	TokenURL         string
	Scopes           []string
	AuthorizeHandler func(context *auth.Context) (*claims.Claims, error)
}

func New(config *Config) *FacebookProvider {
	if config == nil {
		config = &Config{}
	}

	provider := &FacebookProvider{Config: config}

	if config.ClientID == "" {
		panic(errors.New("Facebook's ClientID can't be blank"))
	}

	if config.ClientSecret == "" {
		panic(errors.New("Facebook's ClientSecret can't be blank"))
	}

	if config.AuthorizeURL == "" {
		config.AuthorizeURL = facebook.Endpoint.AuthURL
	}

	if config.TokenURL == "" {
		config.TokenURL = facebook.Endpoint.TokenURL
	}

	if len(config.Scopes) == 0 {
		config.Scopes = []string{"public_profile", "email"}
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
					resp, err := http.Get("https://graph.facebook.com/me?access_token=" + tkn.AccessToken)
					if err != nil {
						return nil, err
					}

					defer resp.Body.Close()
					body, _ := ioutil.ReadAll(resp.Body)
					userInfo := UserInfo{}
					json.Unmarshal(body, &userInfo)
					schema.Provider = provider.GetName()
					schema.UID = userInfo.ID
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
func (FacebookProvider) GetName() string {
	return "facebook"
}

// ConfigAuth config auth
func (provider FacebookProvider) ConfigAuth(auth *auth.Auth) {
	auth.Render.RegisterViewPath("github.com/qor/auth/providers/facebook/views")
}

// OAuthConfig return oauth config based on configuration
func (provider FacebookProvider) OAuthConfig(context *auth.Context) *oauth2.Config {
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
		RedirectURL: scheme + context.Request.Host + context.Auth.AuthURL("facebook/callback"),
		Scopes:      config.Scopes,
	}
}

// Login implemented login with facebook provider
func (provider FacebookProvider) Login(context *auth.Context) {
	claims := claims.Claims{}
	claims.Subject = "state"
	signedToken := context.Auth.SessionStorer.SignedToken(&claims)

	url := provider.OAuthConfig(context).AuthCodeURL(signedToken)
	http.Redirect(context.Writer, context.Request, url, http.StatusFound)
}

// Logout implemented logout with facebook provider
func (FacebookProvider) Logout(context *auth.Context) {
}

// Register implemented register with facebook provider
func (provider FacebookProvider) Register(context *auth.Context) {
	provider.Login(context)
}

// Callback implement Callback with facebook provider
func (provider FacebookProvider) Callback(context *auth.Context) {
	context.Auth.LoginHandler(context, provider.AuthorizeHandler)
}

// ServeHTTP implement ServeHTTP with facebook provider
func (FacebookProvider) ServeHTTP(*auth.Context) {
}

// UserInfo facebook user info structure
type UserInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	GivenName  string `json:"first_name"`
	FamilyName string `json:"last_name"`
	Picture    string `json:"picture"`
	Profile    string `json:"link"`
	Email      string `json:"email"`
	Gender     string `json:"gender"`
	Locale     string `json:"locale"`
	Verified   bool   `json:"verified"`
}
