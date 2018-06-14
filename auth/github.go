package auth

import (
	"fmt"
	"reflect"

	"github.com/google/go-github/github"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	githubauth "github.com/qor/auth/providers/github"
	"github.com/qor/qor/utils"
	"golang.org/x/oauth2"
)

//GithubExtraInfo struct for github credentials
type GithubExtraInfo struct {
	Login string
	Token string
}

//NewGithubAuthorizeHandler handler for Github auth
func NewGithubAuthorizeHandler(provider *githubauth.GithubProvider) func(context *auth.Context) (*claims.Claims, error) {
	return func(context *auth.Context) (*claims.Claims, error) {
		var (
			schema       auth.Schema
			authInfo     auth_identity.Basic
			authIdentity = reflect.New(utils.ModelType(context.Auth.Config.AuthIdentityModel)).Interface()
			req          = context.Request
			tx           = context.Auth.GetDB(req)
			oauthCfg     = provider.OAuthConfig(context)
			token        *oauth2.Token
		)

		// A user can pass in pre-defined GitHub personal access token, let's check that first
		// This should be used only for testing a non-web flow in CI for example
		token = &oauth2.Token{AccessToken: req.URL.Query().Get("access_token")}

		if token.AccessToken == "" {
			state := req.URL.Query().Get("state")
			claims, err := context.Auth.SessionStorer.ValidateClaims(state)

			if err != nil {
				log.Info(context.Request.RemoteAddr, err.Error())
				return nil, err
			}

			if claims.Valid() != nil || claims.Subject != "state" {
				log.Info(context.Request.RemoteAddr, auth.ErrUnauthorized.Error())
				return nil, auth.ErrUnauthorized
			}

			token, err = oauthCfg.Exchange(oauth2.NoContext, req.URL.Query().Get("code"))

			if err != nil {
				log.Info(context.Request.RemoteAddr, err.Error())
				return nil, err
			}
		}

		client := github.NewClient(oauthCfg.Client(oauth2.NoContext, token))
		user, _, err := client.Users.Get(oauth2.NoContext, "")
		if err != nil {
			log.Info(context.Request.RemoteAddr, err.Error())
			return nil, err
		}

		authInfo.Provider = provider.GetName()
		authInfo.UID = fmt.Sprint(*user.ID)

		if !tx.Model(authIdentity).Where(authInfo).Scan(&authInfo).RecordNotFound() {
			return authInfo.ToClaims(), nil
		}

		{
			schema.Provider = provider.GetName()
			schema.UID = fmt.Sprint(*user.ID)
			schema.Name = user.GetName()
			schema.Email = user.GetEmail()
			schema.Image = user.GetAvatarURL()
			schema.RawInfo = &GithubExtraInfo{Login: user.GetLogin(), Token: token.AccessToken}
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

		log.Info(context.Request.RemoteAddr, err.Error())
		return nil, err
	}
}

// GetGithubUser returns github user by token
func GetGithubUser(accessToken string) (*github.User, error) {
	client := github.NewClient(oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})))
	user, _, err := client.Users.Get(oauth2.NoContext, "")
	return user, err
}
