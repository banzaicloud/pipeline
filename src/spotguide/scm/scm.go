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

package scm

import (
	"fmt"
	"net/http"
	"time"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/google/go-github/github"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/src/auth"
)

const InitialCommitMessage = "initial Banzai Cloud Pipeline commit"
const RepoDescription = "Spotguide by Banzai Cloud"

// File encodings
const EncodingBase64 = "base64"
const EncodingText = "text"

type Repository struct {
	owner string
	name  string
}

func (r Repository) GetOwner() string {
	return r.owner
}

func (r Repository) GetName() string {
	return r.name
}

func (r Repository) GetFullName() string {
	return fmt.Sprint(r.owner, "/", r.name)
}

type RepositoryRelease struct {
	tag        string
	body       string
	preRelease bool
}

func (r RepositoryRelease) GetTag() string {
	return r.tag
}

func (r RepositoryRelease) GetBody() string {
	return r.body
}

func (r RepositoryRelease) IsPreRelease() bool {
	return r.preRelease
}

type RepositoryFile struct {
	Path     string
	Content  string
	Encoding string
}

type SCM interface {
	DownloadFile(owner, repo, file, tag string) ([]byte, error)
	DownloadRelease(owner, repo, tag string) ([]byte, error)
	ListRepositoriesByTopic(owner, topic string, allowPrivate bool) ([]Repository, error)
	ListRepositoryReleases(owner, name string) ([]RepositoryRelease, error)
	CreateRepository(owner, name string, private bool, userID uint) error
	AddContentToRepository(owner, name string, content []RepositoryFile) error
}

type SCMFactory interface {
	CreateSharedSCM() (SCM, error)
	CreateUserSCM(userID uint) (SCM, error)
}

type GitHubSCMFactory struct {
	sharedSCMToken string
	scmTokenStore  auth.SCMTokenStore
}

func (f GitHubSCMFactory) newClient(accessToken string) *github.Client {
	httpClient := oauth2.NewClient(
		oauth2.NoContext,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}),
	)
	httpClient.Timeout = time.Second * 10

	return github.NewClient(httpClient)
}

func (f GitHubSCMFactory) CreateSharedSCM() (SCM, error) {
	githubClient := f.newClient(f.sharedSCMToken)
	return NewGitHubSCM(githubClient), nil
}

func (f GitHubSCMFactory) CreateUserSCM(userID uint) (SCM, error) {
	scmToken, err := f.scmTokenStore.GetSCMTokenByProvider(userID, auth.GithubTokenID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get GitHub token")
	}

	if scmToken == "" {
		return nil, errors.New("user's github token is not set")
	}

	return NewGitHubSCM(f.newClient(scmToken)), nil
}

type GitLabSCMFactory struct {
	sharedSCMToken string
	scmTokenStore  auth.SCMTokenStore
}

func (f GitLabSCMFactory) newClient(accessToken string) (*gitlab.Client, error) {
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	gitlabClient := gitlab.NewClient(httpClient, accessToken)

	gitlabURL := global.Config.Gitlab.URL
	err := gitlabClient.SetBaseURL(gitlabURL)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "gitlabBaseURL", gitlabURL)
	}

	return gitlabClient, nil
}

func (f GitLabSCMFactory) CreateSharedSCM() (SCM, error) {
	gitlabClient, err := f.newClient(f.sharedSCMToken)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create GitLab client")
	}

	return NewGitLabSCM(gitlabClient), nil
}

func (f GitLabSCMFactory) CreateUserSCM(userID uint) (SCM, error) {
	scmToken, err := f.scmTokenStore.GetSCMTokenByProvider(userID, auth.GitlabTokenID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get gitlab token")
	}

	if scmToken == "" {
		return nil, errors.New("user's gitlab token is not set")
	}

	gitlabClient, err := f.newClient(scmToken)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create GitLab client")
	}

	return NewGitLabSCM(gitlabClient), nil
}

func NewSCMFactory(scmProvider string, sharedSCMToken string, scmTokenStore auth.SCMTokenStore) (SCMFactory, error) {
	switch scmProvider {
	case "github":
		return &GitHubSCMFactory{
			sharedSCMToken: sharedSCMToken,
			scmTokenStore:  scmTokenStore,
		}, nil
	case "gitlab":
		return &GitLabSCMFactory{
			sharedSCMToken: sharedSCMToken,
			scmTokenStore:  scmTokenStore,
		}, nil
	default:
		return nil, fmt.Errorf("Unknown SCM provider configured: %s", scmProvider)
	}
}
