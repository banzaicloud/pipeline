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
	"github.com/goph/emperror"
)

type organization struct {
	name     string
	id       int64
	role     string
	provider string
}

// GetSCMToken get scm token
func GetSCMToken(userID uint) (string, string, error) {
	scmToken, err := GetUserGithubToken(userID)
	if err == nil && scmToken != "" {
		return scmToken, GithubTokenID, nil
	}

	scmToken, err = GetUserGitlabToken(userID)
	if err == nil && scmToken != "" {
		return scmToken, GitlabTokenID, nil
	}

	return "", "", emperror.Wrap(err, "failed to fetch user's scm token")
}

// UpdateSCMToken update user token
func UpdateSCMToken(user *User, scmToken string, provider string) (string, error) {
	if scmToken != "" {
		err := SaveUserSCMToken(user, scmToken, provider)
		if err != nil {
			message := "failed to update user's access token"
			return message, emperror.Wrap(err, message)
		}
	} else {
		err := RemoveUserSCMToken(user, provider)
		if err != nil {
			message := "failed to remove user's access token"
			return message, emperror.Wrap(err, message)
		}
	}
	return "", nil
}
