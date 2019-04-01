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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/goph/emperror"
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
}

func (f GitHubSCMFactory) CreateSharedSCM() (SCM, error) {
	githubClient := auth.NewGithubClient(f.sharedSCMToken)
	return NewGitHubSCM(githubClient), nil
}

func (f GitHubSCMFactory) CreateUserSCM(userID uint) (SCM, error) {
	githubClient, err := auth.NewGithubClientForUser(userID)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create GitHub client")
	}

	return NewGitHubSCM(githubClient), nil
}

type GitLabSCMFactory struct {
	sharedSCMToken string
}

func (f GitLabSCMFactory) CreateSharedSCM() (SCM, error) {
	gitlabClient, err := auth.NewGitlabClient(f.sharedSCMToken)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create GitLab client")
	}

	return NewGitLabSCM(gitlabClient), nil
}

func (f GitLabSCMFactory) CreateUserSCM(userID uint) (SCM, error) {
	gitlabClient, err := auth.NewGitlabClientForUser(userID)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create GitLab client")
	}

	return NewGitLabSCM(gitlabClient), nil
}

func NewSCMFactory(scmProvider string, sharedSCMToken string) (SCMFactory, error) {
	switch scmProvider {
	case "github":
		return &GitHubSCMFactory{sharedSCMToken: sharedSCMToken}, nil
	case "gitlab":
		return &GitLabSCMFactory{sharedSCMToken: sharedSCMToken}, nil
	default:
		return nil, fmt.Errorf("Unknown SCM provider configured: %s", scmProvider)
	}
}
