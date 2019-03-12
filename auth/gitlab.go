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
	"github.com/mitchellh/mapstructure"
	"github.com/qor/auth"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

type gitlabUserMeta struct {
	Username  string
	AvatarURL string
}

func NewGitlabClient(accessToken string) *gitlab.Client {
	httpClient := oauth2.NewClient(
		oauth2.NoContext,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}),
	)

	return gitlab.NewClient(httpClient, accessToken)
}

func getGitlabUserMeta(schema *auth.Schema) (*gitlabUserMeta, error) {
	gitlabClient := NewGitlabClient(viper.GetString("gitlab.token"))

	var dexClaims struct {
		FederatedClaims map[string]string
	}

	if err := mapstructure.Decode(schema.RawInfo, &dexClaims); err != nil {
		return nil, nil
	}

	gitlabUserID := cast.ToInt64(dexClaims.FederatedClaims["user_id"])

	gitlabUser, _, err := gitlabClient.Users.GetUser(int(gitlabUserID), nil)
	if err != nil {
		return nil, err
	}

	return &gitlabUserMeta{
		Username:  gitlabUser.Username,
		AvatarURL: gitlabUser.AvatarURL,
	}, nil
}
