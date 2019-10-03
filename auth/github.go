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
	"time"

	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/qor/auth"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type githubUserMeta struct {
	Login     string
	AvatarURL string
}

func NewGithubClient(accessToken string) *github.Client {
	httpClient := oauth2.NewClient(
		oauth2.NoContext,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}),
	)
	httpClient.Timeout = time.Second * 10

	return github.NewClient(httpClient)
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
