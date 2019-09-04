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
	"fmt"
	"net/http"
	"time"

	"emperror.dev/emperror"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/qor/auth"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"github.com/xanzy/go-gitlab"
)

type gitlabUserMeta struct {
	Username  string
	AvatarURL string
}

func NewGitlabClient(accessToken string) (*gitlab.Client, error) {
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	gitlabClient := gitlab.NewClient(httpClient, accessToken)

	gitlabURL := viper.GetString("gitlab.baseURL")
	err := gitlabClient.SetBaseURL(gitlabURL)
	if err != nil {
		return nil, emperror.With(err, "gitlabBaseURL", gitlabURL)
	}

	return gitlabClient, nil
}

func NewGitlabClientForUser(userID uint) (*gitlab.Client, error) {
	accessToken, err := GetUserGitlabToken(userID)
	if err != nil {
		return nil, err
	}

	if accessToken == "" {
		return nil, errors.New("user's gitlab token is not set")
	}

	gitlabClient, err := NewGitlabClient(accessToken)
	if err != nil {
		return nil, err
	}
	return gitlabClient, nil
}

func GetUserGitlabToken(userID uint) (string, error) {
	token, err := TokenStore.Lookup(fmt.Sprint(userID), GitlabTokenID)
	if err != nil {
		return "", emperror.Wrap(err, "failed to lookup user token")
	}

	if token == nil {
		return "", nil
	}

	return token.Value, nil
}

func getGitlabUserMeta(schema *auth.Schema) (*gitlabUserMeta, error) {
	gitlabClient, err := NewGitlabClient("")
	if err != nil {
		return nil, err
	}

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
