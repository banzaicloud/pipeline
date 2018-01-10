package twitter

import (
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/mrjones/oauth"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	"github.com/qor/qor/utils"
	"github.com/qor/session"
)

var UserInfoURL = "https://api.twitter.com/1.1/account/verify_credentials.json?include_email=true"

// Provider provide login with twitter
type Provider struct {
	Auth *auth.Auth
	*Config
}

// Config twitter Config
type Config struct {
	ClientID         string
	ClientSecret     string
	AuthorizeURL     string
	TokenURL         string
	RedirectURL      string
	AuthorizeHandler func(context *auth.Context) (*claims.Claims, error)
}

func New(config *Config) *Provider {
	if config == nil {
		config = &Config{}
	}

	provider := &Provider{Config: config}

	if config.ClientID == "" {
		panic(errors.New("Twitter's ClientID can't be blank"))
	}

	if config.ClientSecret == "" {
		panic(errors.New("Twitter's ClientSecret can't be blank"))
	}

	if config.AuthorizeHandler == nil {
		config.AuthorizeHandler = func(context *auth.Context) (*claims.Claims, error) {
			var (
				authInfo     auth_identity.Basic
				schema       auth.Schema
				requestToken = &oauth.RequestToken{}
				consumer     = provider.NewConsumer(context)
				oauthToken   = context.Request.URL.Query().Get("oauth_verifier")
				authIdentity = reflect.New(utils.ModelType(context.Auth.Config.AuthIdentityModel)).Interface()
				tx           = context.Auth.GetDB(context.Request)
			)

			Claims, err := provider.Auth.Get(context.Request)
			if err != nil {
				return nil, err
			}

			json.Unmarshal([]byte(Claims.Issuer), requestToken)

			if context.Request.URL.Query().Get("oauth_token") != requestToken.Token {
				return nil, errors.New("invalid token")
			}

			atoken, err := consumer.AuthorizeToken(requestToken, oauthToken)

			if err != nil {
				return nil, err
			}

			{
				client, err := consumer.MakeHttpClient(atoken)
				resp, err := client.Get(UserInfoURL)
				if err != nil {
					return nil, err
				}

				defer resp.Body.Close()
				body, _ := ioutil.ReadAll(resp.Body)
				userInfo := UserInfo{}
				if err := json.Unmarshal(body, &userInfo); err != nil {
					return nil, err
				}
				schema.Provider = provider.GetName()
				schema.UID = userInfo.ID
				schema.Email = userInfo.Email
				schema.Image = userInfo.Picture
				schema.Name = userInfo.Name
				schema.Location = userInfo.Location
				schema.URL = userInfo.Profile
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

			return nil, nil
		}
	}

	return provider
}

// GetName return provider name
func (Provider) GetName() string {
	return "twitter"
}

// ConfigAuth config auth
func (provider *Provider) ConfigAuth(auth *auth.Auth) {
	provider.Auth = auth
	provider.Auth.Render.RegisterViewPath("github.com/qor/auth/providers/twitter/views")
}

// NewConsumer new twitter consumer
func (provider Provider) NewConsumer(context *auth.Context) *oauth.Consumer {
	return oauth.NewConsumer(provider.ClientID, provider.ClientSecret, oauth.ServiceProvider{
		RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
		AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
		AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
	})
}

// Login implemented login with twitter provider
func (provider Provider) Login(context *auth.Context) {
	var (
		scheme   = context.Request.URL.Scheme
		consumer = provider.NewConsumer(context)
	)

	if scheme == "" {
		scheme = "http://"
	}

	requestToken, u, err := consumer.GetRequestTokenAndUrl(scheme + context.Request.Host + context.Auth.AuthURL("twitter/callback"))

	if err == nil {
		// save requestToken into session
		Claims := &claims.Claims{}
		if c, err := provider.Auth.Get(context.Request); err == nil {
			Claims = c
		}
		tokenStr, _ := json.Marshal(requestToken)
		Claims.Issuer = string(tokenStr)
		provider.Auth.Update(context.Writer, context.Request, Claims)

		http.Redirect(context.Writer, context.Request, u, http.StatusFound)
		return
	}

	context.SessionStorer.Flash(context.Writer, context.Request, session.Message{Message: template.HTML(err.Error()), Type: "error"})
	context.Auth.Config.Render.Execute("auth/login", context, context.Request, context.Writer)
}

// Logout implemented logout with twitter provider
func (Provider) Logout(context *auth.Context) {
}

// Register implemented register with twitter provider
func (provider Provider) Register(context *auth.Context) {
	provider.Login(context)
}

// Callback implement Callback with twitter provider
func (provider Provider) Callback(context *auth.Context) {
	context.Auth.LoginHandler(context, provider.AuthorizeHandler)
}

// ServeHTTP implement ServeHTTP with twitter provider
func (Provider) ServeHTTP(*auth.Context) {
}

// UserInfo twitter user info structure
type UserInfo struct {
	ID       string `json:"id_str"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Location string `json:"location"`
	Locale   string `json:"lang"`
	Picture  string `json:"profile_image_url"`
	Profile  string `json:"url"`
	Verified bool   `json:"verified"`
}
