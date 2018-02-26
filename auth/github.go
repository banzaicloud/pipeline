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
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type GithubExtraInfo struct {
	Login string
	Token string
}

func NewGithubAuthorizeHandler(provider *githubauth.GithubProvider) func(context *auth.Context) (*claims.Claims, error) {
	log = logger.WithFields(logrus.Fields{"tag": "Auth"})
	return func(context *auth.Context) (*claims.Claims, error) {
		var (
			schema       auth.Schema
			authInfo     auth_identity.Basic
			authIdentity = reflect.New(utils.ModelType(context.Auth.Config.AuthIdentityModel)).Interface()
			req          = context.Request
			tx           = context.Auth.GetDB(req)
		)

		state := req.URL.Query().Get("state")
		claims, err := context.Auth.SessionStorer.ValidateClaims(state)

		if err != nil || claims.Valid() != nil || claims.Subject != "state" {
			log.Info(context.Request.RemoteAddr, auth.ErrUnauthorized.Error())
			return nil, auth.ErrUnauthorized
		}

		if err == nil {
			oauthCfg := provider.OAuthConfig(context)
			tkn, err := oauthCfg.Exchange(oauth2.NoContext, req.URL.Query().Get("code"))

			if err != nil {
				log.Info(context.Request.RemoteAddr, err.Error())
				return nil, err
			}

			client := github.NewClient(oauthCfg.Client(oauth2.NoContext, tkn))
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
				schema.RawInfo = &GithubExtraInfo{Login: user.GetLogin(), Token: tkn.AccessToken}
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

		log.Info(context.Request.RemoteAddr, err.Error())
		return nil, err
	}
}
