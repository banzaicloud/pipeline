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

	"github.com/google/go-github/github"
	"github.com/goph/emperror"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/qor/auth"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type githubUserMeta struct {
	Login     string
	AvatarURL string
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

	return &githubUserMeta{
		Login:     *githubUser.Login,
		AvatarURL: githubUser.GetAvatarURL(),
	}, nil
}
