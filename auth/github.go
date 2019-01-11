// Copyright © 2018 Banzai Cloud
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
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-github/github"
	"github.com/goph/emperror"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	githubauth "github.com/qor/auth/providers/github"
	"github.com/qor/qor/utils"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type githubUserMeta struct {
	Login         string
	AvatarURL     string
	Organizations []string
}

//NewGithubAuthorizeHandler handler for Github auth
func NewGithubAuthorizeHandler(provider *githubauth.GithubProvider) func(context *auth.Context) (*claims.Claims, error) {
	return func(ctx *auth.Context) (*claims.Claims, error) {
		var (
			schema       auth.Schema
			authInfo     auth_identity.Basic
			authIdentity = reflect.New(utils.ModelType(ctx.Auth.Config.AuthIdentityModel)).Interface()
			req          = ctx.Request
			db           = ctx.Auth.GetDB(req)
			oauthCfg     = provider.OAuthConfig(ctx)
			token        *oauth2.Token
		)

		// A user can pass in pre-defined GitHub personal access token, let's check that first
		// This should be used only for testing a non-web flow in CI for example
		token = &oauth2.Token{AccessToken: req.URL.Query().Get("access_token")}

		if token.AccessToken == "" {
			state := req.URL.Query().Get("state")
			claims, err := ctx.Auth.SessionStorer.ValidateClaims(state)

			if err != nil {
				log.Errorln("failed to validate user claims", err.Error())
				return nil, err
			}

			if claims.Valid() != nil || claims.Subject != "state" {
				log.Infoln("invalid user claims", auth.ErrUnauthorized.Error())
				return nil, auth.ErrUnauthorized
			}

			token, err = oauthCfg.Exchange(oauth2.NoContext, req.URL.Query().Get("code"))

			if err != nil {
				log.Errorln("oauth exchange failed", err.Error())
				return nil, err
			}
		}

		client := github.NewClient(oauthCfg.Client(oauth2.NoContext, token))
		user, _, err := client.Users.Get(oauth2.NoContext, "")
		if err != nil {
			log.Errorln("failed to query user metadata from GitHub", err.Error())
			return nil, err
		}

		authInfo.Provider = provider.GetName()
		authInfo.UID = fmt.Sprint(user.GetID())

		schema.RawInfo = &githubUserMeta{Login: user.GetLogin()}

		// If the user is already registered, just return
		if tx := db.Model(authIdentity).Where(authInfo).Scan(&authInfo); tx.Error == nil {
			ctx.Claims = authInfo.ToClaims()
			return authInfo.ToClaims(), ctx.Auth.UserStorer.Update(&schema, ctx)
		} else if !tx.RecordNotFound() {
			log.Errorln("failed to check if user is already registered", tx.Error.Error())
			return nil, err
		}

		if viper.GetBool("auth.whitelistEnabled") {

			whitelistedCandidates := []*WhitelistedAuthIdentity{}

			// Check here that a user login name is in the whitelisted_auth_identities table
			userWhitelisted := WhitelistedAuthIdentity{
				Provider: authInfo.Provider,
				UID:      authInfo.UID,
				Login:    user.GetLogin(),
				Type:     UserType,
			}

			whitelistedCandidates = append(whitelistedCandidates, &userWhitelisted)

			// Also check if the user is member of one of the whitelisted organizations
			userOrgs, _, err := client.Organizations.List(oauth2.NoContext, "", &github.ListOptions{})
			if err != nil {
				log.Errorln("failed to query user's organizations from GitHub", err.Error())
				return nil, err
			}

			for _, userOrg := range userOrgs {

				orgWhitelisted := WhitelistedAuthIdentity{
					Provider: authInfo.Provider,
					UID:      fmt.Sprint(userOrg.GetID()),
					Login:    userOrg.GetLogin(),
					Type:     OrganizationType,
				}

				whitelistedCandidates = append(whitelistedCandidates, &orgWhitelisted)
			}

			var userIsWhitelisted bool
			for _, whitelistedCandidate := range whitelistedCandidates {

				if tx := db.Where(&whitelistedCandidate).Find(&WhitelistedAuthIdentity{}); tx.Error == nil {
					userIsWhitelisted = true
					break
				} else if !tx.RecordNotFound() {
					log.Errorln("failed to check whitelist in db", tx.Error.Error())
					return nil, err
				}
			}

			if !userIsWhitelisted {
				return nil, fmt.Errorf("sorry, you are not invited currently to this release")
			}
		}

		// If user email is not available in the primary user info (hidden email on profile)
		// get it with the help of the API (the user has given right to do that).
		if user.Email == nil {
			emails, _, err := client.Users.ListEmails(oauth2.NoContext, &github.ListOptions{})
			if err != nil {
				log.Errorln("failed to fetch user's emails from GitHub", err.Error())
				return nil, err
			}

			for _, email := range emails {
				if email.GetPrimary() {
					user.Email = email.Email
					break
				}
			}
		}

		ctx.Request = req.WithContext(context.WithValue(req.Context(), SignUp, true))

		{
			schema.Provider = provider.GetName()
			schema.UID = fmt.Sprint(*user.ID)
			schema.Name = user.GetName()
			schema.Email = user.GetEmail()
			schema.Image = user.GetAvatarURL()
		}
		if _, userID, err := ctx.Auth.UserStorer.Save(&schema, ctx); err == nil {
			if userID != "" {
				authInfo.UserID = userID
			}
		} else {
			log.Errorln("failed to store user in db", err.Error())
			return nil, err
		}

		if err = db.Where(authInfo).FirstOrCreate(authIdentity).Error; err == nil {
			return authInfo.ToClaims(), nil
		}

		log.Errorln("failed to create auth identity for user in db", err.Error())
		return nil, err
	}
}

// GetGithubUser returns github user by token
func getGithubUser(accessToken string) (*github.User, error) {
	client := NewGithubClient(accessToken)
	user, _, err := client.Users.Get(oauth2.NoContext, "")
	return user, err
}

func NewGithubClient(accessToken string) *github.Client {
	httpClient := oauth2.NewClient(
		oauth2.NoContext,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}),
	)

	return github.NewClient(httpClient)
}

func GetUserGithubToken(userID uint) (string, error) {
	token, err := TokenStore.Lookup(fmt.Sprint(userID), GithubTokenID)
	if err != nil {
		return "", emperror.Wrap(err, "failed to lookup user token")
	}

	if token == nil {
		return "", errors.New("github token not found for user")
	}

	return token.Value, nil
}

func NewGithubClientForUser(userID uint) (*github.Client, error) {
	accessToken, err := GetUserGithubToken(userID)
	if err != nil {
		return nil, err
	}

	return NewGithubClient(accessToken), nil
}

type organization struct {
	name     string
	id       int64
	role     string
	provider string
}

func getGithubOrganizations(token string) ([]organization, error) {
	httpClient := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	githubClient := github.NewClient(httpClient)

	memberships, _, err := githubClient.Organizations.ListOrgMemberships(oauth2.NoContext, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list organization memberships")
	}

	var orgs []organization
	for _, membership := range memberships {
		org := organization{
			name:     membership.GetOrganization().GetLogin(),
			id:       membership.GetOrganization().GetID(),
			role:     membership.GetRole(),
			provider: "github",
		}

		orgs = append(orgs, org)
	}

	return orgs, nil
}

func getGithubUserMeta(schema *auth.Schema) (*githubUserMeta, error) {
	githubClient := NewGithubClient(viper.GetString("github.token"))

	var dexClaims struct {
		Groups          []string
		FederatedClaims map[string]string
	}

	if err := mapstructure.Decode(schema.RawInfo, &dexClaims); err != nil {
		return nil, nil
	}

	githubUserID := cast.ToInt64(dexClaims.FederatedClaims["user_id"])

	githubUser, _, err := githubClient.Users.GetByID(context.Background(), githubUserID)
	if err != nil {
		return nil, err
	}

	var organizations []string
	for _, group := range dexClaims.Groups {
		if !strings.Contains(group, ":") {
			organizations = append(organizations, group)
		}
	}

	return &githubUserMeta{
		Login:         *githubUser.Login,
		AvatarURL:     githubUser.GetAvatarURL(),
		Organizations: organizations,
	}, nil
}
