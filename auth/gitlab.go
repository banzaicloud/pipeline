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

	"github.com/goph/emperror"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/qor/auth"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
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

func getGitlabOrganizations(token string) ([]organization, error) {
	gitlabClient, err := NewGitlabClient(token)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create gitlab client")
	}

	minAccessLevel := gitlab.DeveloperPermissions
	groups, _, err := gitlabClient.Groups.ListGroups(&gitlab.ListGroupsOptions{MinAccessLevel: &minAccessLevel})

	if err != nil {
		return nil, emperror.Wrap(err, "failed to list groups from gitlab")
	}

	currentUser, _, err := gitlabClient.Users.CurrentUser()
	if err != nil {
		return nil, emperror.With(err, "unable to get current gitlab user")
	}
	var orgs []organization
	for _, group := range groups {
		role, err := getGroupAccesLevel(gitlabClient, group.ID, currentUser.ID)
		if err != nil {
			return nil, emperror.With(err, "groupID", group.ID, "userID", currentUser.ID)
		}
		org := organization{
			name:     group.Name,
			id:       int64(group.ID),
			role:     role,
			provider: ProviderGitlab,
		}

		orgs = append(orgs, org)
	}

	userOrg := organization{
		name:     currentUser.Username,
		role:     "admin",
		provider: ProviderGitlab,
	}

	orgs = append(orgs, userOrg)

	return orgs, nil
}

func getGroupAccesLevel(gitlabClient *gitlab.Client, groupID int, userID int) (string, error) {

	groupMember, _, err := gitlabClient.GroupMembers.GetGroupMember(groupID, userID)
	if err != nil {
		return "", emperror.With(err, "userID", userID, "groupID", groupID)
	}
	role := map[int]string{
		30: "DeveloperPermissions",
		40: "MaintainerPermissions",
		50: "OwnerPermissions",
	}

	return role[int(groupMember.AccessLevel)], nil
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
