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

package spotguide

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/google/go-github/github"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/xanzy/go-gitlab"
)

const InitialCommitMessage = "initial Banzai Cloud Pipeline commit"
const RepoDescription = "Spotguide by Banzai Cloud"

type scmRepository struct {
	owner string
	name  string
}

func (r scmRepository) GetOwner() string {
	return r.owner
}

func (r scmRepository) GetName() string {
	return r.name
}

func (r scmRepository) GetFullName() string {
	return fmt.Sprint(r.owner, "/", r.name)
}

type scmRepositoryRelease struct {
	tag        string
	body       string
	preRelease bool
}

func (r scmRepositoryRelease) GetTag() string {
	return r.tag
}

func (r scmRepositoryRelease) GetBody() string {
	return r.body
}

func (r scmRepositoryRelease) IsPreRelease() bool {
	return r.preRelease
}

type scm interface {
	DownloadFile(owner, repo, file, tag string) ([]byte, error)
	DownloadRelease(owner, repo, tag string) ([]byte, error)
	ListRepositoriesByTopic(owner, topic string) ([]scmRepository, error)
	ListRepositoryReleases(owner, name string) ([]scmRepositoryRelease, error)
	CreateRepository(owner, name string, private bool, userID uint) error
	AddContentToRepository(owner, name string, content []repoFile) error
}

type gitHubSCM struct {
	client *github.Client
}

func newGitHubSCM(client *github.Client) scm {
	return &gitHubSCM{client: client}
}

func (scm *gitHubSCM) DownloadFile(owner, repo, file, tag string) ([]byte, error) {
	reader, err := scm.client.Repositories.DownloadContents(ctx, owner, repo, file, &github.RepositoryContentGetOptions{
		Ref: tag,
	})
	if err != nil {
		return nil, emperror.Wrap(err, "failed to download file from GitHub")
	}

	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	return data, emperror.Wrap(err, "failed to download file from GitHub")
}

func (scm *gitHubSCM) DownloadRelease(owner, repo, tag string) ([]byte, error) {
	sourceRelease, _, err := scm.client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to find source spotguide repository release")
	}

	// Support private repositories via downloading with an authenticated client
	downloadRequest, err := http.NewRequest(http.MethodGet, sourceRelease.GetZipballURL(), nil)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create source spotguide repository release download request")
	}

	repoBytes := bytes.NewBuffer(nil)
	_, err = scm.client.Do(ctx, downloadRequest, repoBytes)

	return repoBytes.Bytes(), emperror.Wrap(err, "failed to download source spotguide repository release")
}

func (scm *gitHubSCM) ListRepositoriesByTopic(owner, topic string) ([]scmRepository, error) {

	var repositories []scmRepository

	query := fmt.Sprintf("org:%s topic:%s fork:true", owner, topic)
	if !viper.GetBool(config.SpotguideAllowPrivateRepos) {
		query += " is:public"
	}

	listOpts := github.ListOptions{PerPage: 100}

	for {
		reposRes, resp, err := scm.client.Search.Repositories(ctx, query, &github.SearchOptions{
			Sort:        "created",
			Order:       "asc",
			ListOptions: listOpts,
		})

		if err != nil {
			// Empty organization, no repositories
			if resp != nil && resp.StatusCode == http.StatusUnprocessableEntity {
				return repositories, nil
			}

			return nil, emperror.Wrap(err, "failed to list github repositories")
		}

		for _, githubRepo := range reposRes.Repositories {
			repo := scmRepository{
				owner: githubRepo.GetOwner().GetLogin(),
				name:  githubRepo.GetName(),
			}
			repositories = append(repositories, repo)
		}

		if resp.NextPage == 0 {
			break
		}

		listOpts.Page = resp.NextPage
	}

	return repositories, nil
}

func (scm *gitHubSCM) ListRepositoryReleases(owner, name string) ([]scmRepositoryRelease, error) {

	var releases []scmRepositoryRelease

	listOpts := github.ListOptions{PerPage: 100}

	for {
		githubReleases, resp, err := scm.client.Repositories.ListReleases(ctx, owner, name, &listOpts)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list github repository releases")
		}

		for _, githubRelease := range githubReleases {
			release := scmRepositoryRelease{
				tag:        githubRelease.GetTagName(),
				body:       githubRelease.GetBody(),
				preRelease: githubRelease.GetPrerelease(),
			}
			releases = append(releases, release)
		}

		if resp.NextPage == 0 {
			break
		}

		listOpts.Page = resp.NextPage
	}

	return releases, nil
}

func (scm *gitHubSCM) CreateRepository(owner, name string, private bool, userID uint) error {
	repo := github.Repository{
		Name:        github.String(name),
		Description: github.String(RepoDescription),
		Private:     github.Bool(private),
	}

	// If the user's name is used as organization name, it has to be cleared in repo create.
	// See: https://developer.github.com/v3/repos/#create
	orgName := owner
	if auth.GetUserNickNameById(userID) == owner {
		orgName = ""
	}

	_, _, err := scm.client.Repositories.Create(ctx, orgName, &repo)
	if err != nil {
		return emperror.Wrap(err, "failed to create spotguide repository")
	}

	log.Infof("created spotguide repository: %s/%s", owner, name)
	return nil
}

func (scm *gitHubSCM) AddContentToRepository(owner, name string, spotguideContent []repoFile) error {
	// List the files here that needs to be created in this commit and create a tree from them
	entries := []github.TreeEntry{}

	for _, repoFile := range spotguideContent {

		// The GitHub API accepts blobs as utf-8 by default, and we can change the encoding only in the
		// CreateBlob call, so if the file is utf-8 let's spare an API call, otherwise create the blob
		// with base64 encoding specified.
		var blobSHA, blobContent *string

		if repoFile.encoding == EncodingBase64 {
			blob, _, err := scm.client.Git.CreateBlob(ctx, owner, name, &github.Blob{
				Content:  github.String(repoFile.content),
				Encoding: github.String(repoFile.encoding),
			})
			if err != nil {
				return emperror.Wrapf(err, "failed to create blob for spotguide repository: %s", repoFile.path)
			}

			blobSHA = blob.SHA
		} else {
			blobContent = github.String(string(repoFile.content))
		}

		entry := github.TreeEntry{
			Type:    github.String("blob"),
			Mode:    github.String("100644"),
			Path:    github.String(repoFile.path),
			SHA:     blobSHA,
			Content: blobContent,
		}

		entries = append(entries, entry)
	}

	// A file has to be created with the API to be able to use the fresh repo
	contentOptions := &github.RepositoryContentFileOptions{
		Content: []byte("# Say hello to Spotguides!"),
		Message: github.String("initial import"),
	}

	contentResponse, _, err := scm.client.Repositories.CreateFile(ctx, owner, name, "README.md", contentOptions)

	if err != nil {
		return emperror.Wrap(err, "failed to initialize spotguide repository")
	}

	tree, _, err := scm.client.Git.CreateTree(ctx, owner, name, contentResponse.GetSHA(), entries)

	if err != nil {
		return emperror.Wrap(err, "failed to create git tree for spotguide repository")
	}

	// Create a commit from the tree
	contentResponse.Commit.SHA = contentResponse.SHA

	commit := &github.Commit{
		Message: github.String(InitialCommitMessage),
		Parents: []github.Commit{contentResponse.Commit},
		Tree:    tree,
	}

	newCommit, _, err := scm.client.Git.CreateCommit(ctx, owner, name, commit)

	if err != nil {
		return emperror.Wrap(err, "failed to create git commit for spotguide repository")
	}

	// Attach the commit to the master branch.
	// This can be changed later to another branch + create PR.
	// See: https://github.com/google/go-github/blob/master/example/commitpr/main.go#L62
	ref, _, err := scm.client.Git.GetRef(ctx, owner, name, "refs/heads/master")
	if err != nil {
		return emperror.Wrap(err, "failed to get git ref for spotguide repository")
	}

	ref.Object.SHA = newCommit.SHA

	_, _, err = scm.client.Git.UpdateRef(ctx, owner, name, ref, false)

	if err != nil {
		return emperror.Wrap(err, "failed to update git ref for spotguide repository")
	}

	return nil
}

type gitLabSCM struct {
	client *gitlab.Client
}

func newGitLabSCM(client *gitlab.Client) scm {
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
		SHA: gitlab.String(tag),
	}

	archive, _, err := scm.client.Repositories.Archive(fmt.Sprint(owner, "/", repo), opt)

	return archive, emperror.Wrap(err, "failed to download source spotguide repository release")
}

func (scm *gitLabSCM) ListRepositoriesByTopic(owner, topic string) ([]scmRepository, error) {

	// TODO move this outside, also for github
	var visibility *gitlab.VisibilityValue
	if !viper.GetBool(config.SpotguideAllowPrivateRepos) {
		visibility = gitlab.Visibility(gitlab.PublicVisibility)
	}

	opt := &gitlab.ListGroupProjectsOptions{
		OrderBy:    gitlab.String("created_at"),
		Sort:       gitlab.String("asc"),
		Visibility: visibility,
	}

	var repositories []scmRepository

	for {
		projects, resp, err := scm.client.Groups.ListGroupProjects(owner, opt)

		if err != nil {
			return nil, emperror.Wrap(err, "failed to list GitLab projects")
		}

		for _, project := range projects {

			if utils.Contains(project.TagList, topic) {
				repo := scmRepository{
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
func (scm *gitLabSCM) ListRepositoryReleases(owner, name string) ([]scmRepositoryRelease, error) {
	opt := &gitlab.ListTagsOptions{}

	pid := fmt.Sprint(owner, "/", name)

	var releases []scmRepositoryRelease

	for {
		gitlabTags, resp, err := scm.client.Tags.ListTags(pid, opt)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list github repository tags")
		}

		for _, gitlabTag := range gitlabTags {
			release := scmRepositoryRelease{
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

	log.Infof("created spotguide repository: %s/%s", owner, name)
	return nil
}

func (scm *gitLabSCM) AddContentToRepository(owner, name string, content []repoFile) error {

	var actions []*gitlab.CommitAction

	for _, repoFile := range content {

		action := &gitlab.CommitAction{
			Action:   gitlab.FileCreate,
			FilePath: repoFile.path,
			Content:  repoFile.content,
			Encoding: repoFile.encoding,
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
