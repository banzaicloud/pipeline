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
	"github.com/banzaicloud/pipeline/utils"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
)

type gitLabSCM struct {
	client *gitlab.Client
}

func NewGitLabSCM(client *gitlab.Client) SCM {
	return &gitLabSCM{client: client}
}

func (scm *gitLabSCM) DownloadFile(owner, repo, file, tag string) ([]byte, error) {
	data, _, err := scm.client.RepositoryFiles.GetRawFile(fmt.Sprint(owner, "/", repo), file, &gitlab.GetRawFileOptions{
		Ref: gitlab.String(tag),
	})
	return data, emperror.Wrap(err, "failed to download file from GitLab")
}

func (scm *gitLabSCM) DownloadRelease(owner, repo, tag string) ([]byte, error) {
	opt := &gitlab.ArchiveOptions{
		SHA:    gitlab.String(tag),
		Format: gitlab.String("zip"),
	}

	archive, _, err := scm.client.Repositories.Archive(fmt.Sprint(owner, "/", repo), opt)

	return archive, emperror.Wrap(err, "failed to download source spotguide repository release")
}

func (scm *gitLabSCM) ListRepositoriesByTopic(owner, topic string, allowPrivate bool) ([]Repository, error) {

	// TODO move this outside, also for github
	var visibility *gitlab.VisibilityValue
	if !allowPrivate {
		visibility = gitlab.Visibility(gitlab.PublicVisibility)
	}

	opt := &gitlab.ListGroupProjectsOptions{
		OrderBy:    gitlab.String("created_at"),
		Sort:       gitlab.String("asc"),
		Visibility: visibility,
	}

	var repositories []Repository

	for {
		projects, resp, err := scm.client.Groups.ListGroupProjects(owner, opt)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list GitLab projects")
		}

		for _, project := range projects {
			if utils.Contains(project.TagList, topic) {
				repo := Repository{
					owner: owner,
					name:  project.Name,
				}
				repositories = append(repositories, repo)
			}
		}

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		opt.Page = resp.NextPage
	}

	return repositories, nil
}

// Currently GitLab releases are just tags
func (scm *gitLabSCM) ListRepositoryReleases(owner, name string) ([]RepositoryRelease, error) {
	opt := &gitlab.ListTagsOptions{}

	pid := fmt.Sprint(owner, "/", name)

	var releases []RepositoryRelease

	for {
		gitlabTags, resp, err := scm.client.Tags.ListTags(pid, opt)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list GitLab repository tags")
		}

		for _, gitlabTag := range gitlabTags {
			release := RepositoryRelease{
				tag:  gitlabTag.Release.TagName,
				body: gitlabTag.Release.Description,
			}
			releases = append(releases, release)
		}

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		opt.Page = resp.NextPage
	}

	return releases, nil
}

func (scm *gitLabSCM) CreateRepository(owner, name string, private bool, userID uint) error {
	// If not the user's name is used as organization name, we have to get and set the ID of the namespace
	// See: https://docs.gitlab.com/ee/api/projects.html#create-project
	var namespaceID *int
	if auth.GetUserNickNameById(userID) != owner {
		group, _, err := scm.client.Groups.GetGroup(owner)
		if err != nil {
			return errors.Wrap(err, "failed to create spotguide repository")
		}

		namespaceID = gitlab.Int(group.ID)
	}

	visibility := gitlab.PublicVisibility
	if private {
		visibility = gitlab.PrivateVisibility
	}

	// Create a new project
	projectOptions := &gitlab.CreateProjectOptions{
		NamespaceID: namespaceID,
		Name:        gitlab.String(name),
		Description: gitlab.String(RepoDescription),
		Visibility:  gitlab.Visibility(visibility),
	}

	_, _, err := scm.client.Projects.CreateProject(projectOptions)
	if err != nil {
		return errors.Wrap(err, "failed to create spotguide repository")
	}

	return nil
}

func (scm *gitLabSCM) AddContentToRepository(owner, name string, content []RepositoryFile) error {

	var actions []*gitlab.CommitAction

	for _, repoFile := range content {

		action := &gitlab.CommitAction{
			Action:   gitlab.FileCreate,
			FilePath: repoFile.Path,
			Content:  repoFile.Content,
			Encoding: repoFile.Encoding,
		}

		actions = append(actions, action)
	}

	commitOptions := &gitlab.CreateCommitOptions{
		Branch:        gitlab.String("master"),
		CommitMessage: gitlab.String(InitialCommitMessage),
		Actions:       actions,
	}

	pid := fmt.Sprint(owner, "/", name)

	_, _, err := scm.client.Commits.CreateCommit(pid, commitOptions)
	if err != nil {
		return errors.Wrap(err, "failed to create git commit for spotguide repository")
	}

	return nil
}
