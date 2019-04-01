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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/google/go-github/github"
	"github.com/goph/emperror"
)

type gitHubSCM struct {
	client *github.Client
}

func NewGitHubSCM(client *github.Client) SCM {
	return &gitHubSCM{client: client}
}

func (scm *gitHubSCM) DownloadFile(owner, repo, file, tag string) ([]byte, error) {
	reader, err := scm.client.Repositories.DownloadContents(context.Background(), owner, repo, file, &github.RepositoryContentGetOptions{
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
	sourceRelease, _, err := scm.client.Repositories.GetReleaseByTag(context.Background(), owner, repo, tag)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to find source spotguide repository release")
	}

	// Support private repositories via downloading with an authenticated client
	downloadRequest, err := http.NewRequest(http.MethodGet, sourceRelease.GetZipballURL(), nil)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create source spotguide repository release download request")
	}

	repoBytes := bytes.NewBuffer(nil)
	_, err = scm.client.Do(context.Background(), downloadRequest, repoBytes)

	return repoBytes.Bytes(), emperror.Wrap(err, "failed to download source spotguide repository release")
}

func (scm *gitHubSCM) ListRepositoriesByTopic(owner, topic string, allowPrivate bool) ([]Repository, error) {

	var repositories []Repository

	query := fmt.Sprintf("org:%s topic:%s fork:true", owner, topic)
	if !allowPrivate {
		query += " is:public"
	}

	listOpts := github.ListOptions{PerPage: 100}

	for {
		reposRes, resp, err := scm.client.Search.Repositories(context.Background(), query, &github.SearchOptions{
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
			repo := Repository{
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

func (scm *gitHubSCM) ListRepositoryReleases(owner, name string) ([]RepositoryRelease, error) {

	var releases []RepositoryRelease

	listOpts := github.ListOptions{PerPage: 100}

	for {
		githubReleases, resp, err := scm.client.Repositories.ListReleases(context.Background(), owner, name, &listOpts)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list github repository releases")
		}

		for _, githubRelease := range githubReleases {
			release := RepositoryRelease{
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

	_, _, err := scm.client.Repositories.Create(context.Background(), orgName, &repo)
	if err != nil {
		return emperror.Wrap(err, "failed to create spotguide repository")
	}

	return nil
}

func (scm *gitHubSCM) AddContentToRepository(owner, name string, spotguideContent []RepositoryFile) error {

	// A file has to be created with the API to be able to use the fresh repo
	contentOptions := &github.RepositoryContentFileOptions{
		Content: []byte("# Say hello to Spotguides!"),
		Message: github.String("initial import"),
	}

	contentResponse, _, err := scm.client.Repositories.CreateFile(context.Background(), owner, name, "README.md", contentOptions)
	if err != nil {
		return emperror.Wrap(err, "failed to initialize spotguide repository")
	}

	// List the files here that needs to be created in this commit and create a tree from them
	entries := []github.TreeEntry{}

	for _, repoFile := range spotguideContent {

		// The GitHub API accepts blobs as utf-8 by default, and we can change the encoding only in the
		// CreateBlob call, so if the file is utf-8 let's spare an API call, otherwise create the blob
		// with base64 encoding specified.
		var blobSHA, blobContent *string

		if repoFile.Encoding == EncodingBase64 {
			blob, _, err := scm.client.Git.CreateBlob(context.Background(), owner, name, &github.Blob{
				Content:  github.String(repoFile.Content),
				Encoding: github.String(repoFile.Encoding),
			})
			if err != nil {
				return emperror.Wrapf(err, "failed to create blob for spotguide repository: %s", repoFile.Path)
			}

			blobSHA = blob.SHA
		} else {
			blobContent = github.String(string(repoFile.Content))
		}

		entry := github.TreeEntry{
			Type:    github.String("blob"),
			Mode:    github.String("100644"),
			Path:    github.String(repoFile.Path),
			SHA:     blobSHA,
			Content: blobContent,
		}

		entries = append(entries, entry)
	}

	// Create a tree from the tree entries
	tree, _, err := scm.client.Git.CreateTree(context.Background(), owner, name, contentResponse.GetSHA(), entries)
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

	newCommit, _, err := scm.client.Git.CreateCommit(context.Background(), owner, name, commit)
	if err != nil {
		return emperror.Wrap(err, "failed to create git commit for spotguide repository")
	}

	// Attach the commit to the master branch.
	// This can be changed later to another branch + create PR.
	// See: https://github.com/google/go-github/blob/master/example/commitpr/main.go#L62
	ref, _, err := scm.client.Git.GetRef(context.Background(), owner, name, "refs/heads/master")
	if err != nil {
		return emperror.Wrap(err, "failed to get git ref for spotguide repository")
	}

	ref.Object.SHA = newCommit.SHA

	_, _, err = scm.client.Git.UpdateRef(context.Background(), owner, name, ref, false)
	if err != nil {
		return emperror.Wrap(err, "failed to update git ref for spotguide repository")
	}

	return nil
}
