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

package spotguide

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/client"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/drone/drone-go/drone"
	yaml2 "github.com/ghodss/yaml"
	"github.com/google/go-github/github"
	"github.com/goph/emperror"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const SpotguideGithubTopic = "spotguide"
const SpotguideGithubOrganization = "banzaicloud"
const SpotguideYAMLPath = ".banzaicloud/spotguide.yaml"
const PipelineYAMLPath = ".banzaicloud/pipeline.yaml"
const ReadmePath = ".banzaicloud/README.md"
const IconPath = ".banzaicloud/icon.svg"
const CreateClusterStep = "create_cluster"
const SpotguideRepoTableName = "spotguide_repos"

var ctx = context.Background()

func init() {
	// Subscribe to organization creations and sync spotguides into the newly created organizations
	// TODO move this to a global place and more visible
	authEventEmitter.NotifyOrganizationRegistered(internalScrapeSpotguides)
}

type SpotguideYAML struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description,omitempty"`
	Tags        []string                  `json:"tags,omitempty"`
	Resources   client.RequestedResources `json:"resources"`
	Questions   []Question                `json:"questions"`
}

// Question is an opaque struct from Pipeline's point of view
type Question map[string]interface{}

type SpotguideRepo struct {
	ID               uint      `json:"id" gorm:"primary_key"`
	OrganizationID   uint      `json:"organizationId" gorm:"unique_index:name_and_version"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	Name             string    `json:"name" gorm:"unique_index:name_and_version"`
	DisplayName      string    `json:"displayName" gorm:"-"`
	Icon             string    `json:"icon,omitempty" gorm:"type:mediumtext"`
	Readme           string    `json:"readme" gorm:"type:mediumtext"`
	Version          string    `json:"version" gorm:"unique_index:name_and_version"`
	SpotguideYAMLRaw []byte    `json:"-" gorm:"type:text"`
	SpotguideYAML    `gorm:"-"`
}

func (SpotguideRepo) TableName() string {
	return SpotguideRepoTableName
}

func (r SpotguideRepo) Key() SpotguideRepoKey {
	return SpotguideRepoKey{
		OrganizationID: r.OrganizationID,
		Name:           r.Name,
		Version:        r.Version,
	}
}

type SpotguideRepoKey struct {
	OrganizationID uint
	Name           string
	Version        string
}

func (SpotguideRepoKey) TableName() string {
	return SpotguideRepoTableName
}

func (r *SpotguideRepo) AfterFind() error {
	err := yaml2.Unmarshal(r.SpotguideYAMLRaw, &r.SpotguideYAML)
	r.DisplayName = r.SpotguideYAML.Name
	return err
}

type LaunchRequest struct {
	SpotguideName    string                        `json:"spotguideName" binding:"required"`
	SpotguideVersion string                        `json:"spotguideVersion,omitempty"`
	RepoOrganization string                        `json:"repoOrganization" binding:"required"`
	RepoName         string                        `json:"repoName" binding:"required"`
	RepoPrivate      bool                          `json:"repoPrivate"`
	Cluster          *client.CreateClusterRequest  `json:"cluster" binding:"required"`
	Secrets          []*secret.CreateSecretRequest `json:"secrets,omitempty"`
	Pipeline         map[string]interface{}        `json:"pipeline,omitempty"`
}

func (r LaunchRequest) RepoFullname() string {
	return r.RepoOrganization + "/" + r.RepoName
}

func downloadGithubFile(githubClient *github.Client, owner, repo, file, tag string) ([]byte, error) {
	reader, err := githubClient.Repositories.DownloadContents(ctx, owner, repo, file, &github.RepositoryContentGetOptions{
		Ref: tag,
	})
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func internalScrapeSpotguides(orgID uint) {
	if err := ScrapeSpotguides(orgID); err != nil {
		log.Warnf("failed to scrape Spotguide repositories for org [%d]: %s", orgID, err)
	}
}

func isSpotguideReleaseAllowed(release *github.RepositoryRelease) bool {
	version, err := semver.NewVersion(release.GetTagName())
	if err != nil {
		log.Warn("Failed to parse spotguide release tag: ", err)
		return false
	}
	return version.Prerelease() == "" || viper.GetBool(config.SpotguideAllowPrereleases)
}

func ScrapeSpotguides(orgID uint) error {

	db := config.DB()

	githubClient := auth.NewGithubClient(viper.GetString("github.token"))

	var allRepositories []github.Repository
	query := fmt.Sprintf("org:%s topic:%s", SpotguideGithubOrganization, SpotguideGithubTopic)
	listOpts := github.ListOptions{PerPage: 100}
	for {
		reposRes, resp, err := githubClient.Search.Repositories(ctx, query, &github.SearchOptions{
			Sort:        "created",
			Order:       "asc",
			ListOptions: listOpts,
		})
		if err != nil {
			return emperror.Wrap(err, "failed to list github repositories")
		}
		allRepositories = append(allRepositories, reposRes.Repositories...)

		if resp.NextPage == 0 {
			break
		}

		listOpts.Page = resp.NextPage
	}

	where := SpotguideRepo{
		OrganizationID: orgID,
	}

	var oldSpotguides []SpotguideRepo
	if err := db.Where(&where).Find(&oldSpotguides).Error; err != nil {
		if err != nil {
			return emperror.Wrap(err, "failed to list old spotguides")
		}
	}

	oldSpotguidesIndexed := map[SpotguideRepoKey]SpotguideRepo{}
	for _, sg := range oldSpotguides {
		oldSpotguidesIndexed[sg.Key()] = sg
	}

	for _, repository := range allRepositories {
		owner := repository.GetOwner().GetLogin()
		name := repository.GetName()

		releases, _, err := githubClient.Repositories.ListReleases(ctx, owner, name, &github.ListOptions{})
		if err != nil {
			return emperror.Wrap(err, "failed to list github repo releases")
		}

		for _, release := range releases {

			if !isSpotguideReleaseAllowed(release) {
				continue
			}

			tag := release.GetTagName()

			spotguideRaw, err := downloadGithubFile(githubClient, owner, name, SpotguideYAMLPath, tag)
			if err != nil {
				log.Warnf("failed to scrape repository '%s/%s' at version '%s': %s", owner, name, tag, err)
				continue
			}

			// syntax check spotguide.yaml
			err = yaml2.Unmarshal(spotguideRaw, &SpotguideYAML{})
			if err != nil {
				log.Warnf("failed to scrape repository '%s/%s' at version '%s': %s", owner, name, tag, err)
				continue
			}

			readme, err := downloadGithubFile(githubClient, owner, name, ReadmePath, tag)
			if err != nil {
				log.Warnf("failed to scrape repository '%s/%s' at version '%s': %s", owner, name, tag, err)
				continue
			}

			model := SpotguideRepo{
				OrganizationID:   orgID,
				Name:             repository.GetFullName(),
				SpotguideYAMLRaw: spotguideRaw,
				Readme:           string(readme),
				Version:          tag,
			}

			where := model.Key()

			err = db.Where(&where).Assign(&model).FirstOrCreate(&SpotguideRepo{}).Error

			if err != nil {
				return err
			}

			delete(oldSpotguidesIndexed, model.Key())
		}
	}

	for spotguideRepoKey := range oldSpotguidesIndexed {
		err := db.Where(&spotguideRepoKey).Delete(SpotguideRepo{}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func GetSpotguides(orgID uint) ([]*SpotguideRepo, error) {
	db := config.DB()
	where := SpotguideRepo{OrganizationID: orgID}
	spotguides := []*SpotguideRepo{}
	err := db.Find(&spotguides, where).Error
	return spotguides, err
}

func GetSpotguide(orgID uint, name, version string) ([]SpotguideRepo, error) {
	db := config.DB()
	where := SpotguideRepo{OrganizationID: orgID, Name: name, Version: version}
	repo := []SpotguideRepo{}
	err := db.Find(&repo, where).Error
	return repo, err
}

func LaunchSpotguide(request *LaunchRequest, httpRequest *http.Request, orgID, userID uint) error {

	sourceRepos, err := GetSpotguide(orgID, request.SpotguideName, request.SpotguideVersion)
	if err != nil || len(sourceRepos) == 0 {
		return errors.Wrap(err, "failed to find spotguide repo")
	}

	sourceRepo := &sourceRepos[0]

	// LaunchRequest might not have the version
	request.SpotguideVersion = sourceRepo.Version

	err = createSecrets(request, orgID, userID)
	if err != nil {
		return errors.Wrap(err, "failed to create secrets for spotguide")
	}

	githubClient, err := auth.NewGithubClientForUser(userID)
	if err != nil {
		return errors.Wrap(err, "failed to create GitHub client")
	}

	err = createGithubRepo(githubClient, request, userID, sourceRepo)
	if err != nil {
		return errors.Wrap(err, "failed to create GitHub repository")
	}

	err = enableCICD(request, httpRequest)
	if err != nil {
		return errors.Wrap(err, "failed to enable CI/CD for spotguide")
	}

	err = addSpotguideContent(githubClient, request, userID, sourceRepo)
	if err != nil {
		return errors.Wrap(err, "failed to add spotguide content to repository")
	}

	return nil
}

func preparePipelineYAML(request *LaunchRequest, sourceRepo *SpotguideRepo, pipelineYAML []byte) ([]byte, error) {
	// Create repo config that drives the CICD flow from LaunchRequest
	repoConfig, err := createDroneRepoConfig(pipelineYAML, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize repo config")
	}

	repoConfigRaw, err := yaml.Marshal(repoConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal repo config")
	}

	return repoConfigRaw, nil
}

func getSpotguideContent(githubClient *github.Client, request *LaunchRequest, sourceRepo *SpotguideRepo) ([]github.TreeEntry, error) {
	// Download source repo zip
	sourceRepoParts := strings.Split(sourceRepo.Name, "/")
	sourceRepoOwner := sourceRepoParts[0]
	sourceRepoName := sourceRepoParts[1]

	sourceRelease, _, err := githubClient.Repositories.GetReleaseByTag(ctx, sourceRepoOwner, sourceRepoName, request.SpotguideVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find source spotguide repository release")
	}

	resp, err := http.Get(sourceRelease.GetZipballURL())
	if err != nil {
		return nil, errors.Wrap(err, "failed to download source spotguide repository release")
	}

	defer resp.Body.Close()
	repoBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download source spotguide repository release")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(repoBytes), int64(len(repoBytes)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract source spotguide repository release")
	}

	// List the files here that needs to be created in this commit and create a tree from them
	entries := []github.TreeEntry{}

	for _, zf := range zipReader.File {
		if zf.FileInfo().IsDir() {
			continue
		}

		file, err := zf.Open()
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract source spotguide repository release")
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract source spotguide repository release")
		}

		file.Close()

		path := strings.SplitN(zf.Name, "/", 2)[1]

		// Prepare pipeline.yaml
		if path == PipelineYAMLPath {
			content, err = preparePipelineYAML(request, sourceRepo, content)
			if err != nil {
				return nil, errors.Wrap(err, "failed to prepare pipeline.yaml")
			}
		}

		// The GitHub API accepts blobs as utf-8 by default, and we can change the encoding only in the
		// CreateBlob call, so if the file is utf-8 let's spare an API call, otherwise create the blob
		// with base64 encoding specified.
		var blobSHA, blobContent *string

		if strings.HasSuffix(http.DetectContentType(content), "charset=utf-8") {

			blobContent = github.String(string(content))

		} else {

			blob, _, err := githubClient.Git.CreateBlob(ctx, request.RepoOrganization, request.RepoName, &github.Blob{
				Content:  github.String(base64.StdEncoding.EncodeToString(content)),
				Encoding: github.String("base64"),
			})
			if err != nil {
				return nil, errors.Wrap(err, "failed to create blob for spotguide repository: "+path)
			}

			blobSHA = blob.SHA
		}

		entry := github.TreeEntry{
			Type:    github.String("blob"),
			Mode:    github.String("100644"),
			Path:    github.String(path),
			SHA:     blobSHA,
			Content: blobContent,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func createGithubRepo(githubClient *github.Client, request *LaunchRequest, userID uint, sourceRepo *SpotguideRepo) error {

	repo := github.Repository{
		Name:        github.String(request.RepoName),
		Description: github.String("Spotguide by BanzaiCloud"),
		Private:     github.Bool(request.RepoPrivate),
	}

	// If the user's name is used as organization name, it has to be cleared in repo create.
	// See: https://developer.github.com/v3/repos/#create
	orgName := request.RepoOrganization
	if auth.GetUserNickNameById(userID) == orgName {
		orgName = ""
	}

	_, _, err := githubClient.Repositories.Create(ctx, orgName, &repo)
	if err != nil {
		return errors.Wrap(err, "failed to create spotguide repository")
	}

	log.Infof("Created spotguide repository: %s", request.RepoFullname())
	return nil
}

func addSpotguideContent(githubClient *github.Client, request *LaunchRequest, userID uint, sourceRepo *SpotguideRepo) error {

	// An initial files have to be created with the API to be able to use the fresh repo
	createFile := &github.RepositoryContentFileOptions{
		Content: []byte("# Say hello to Spotguides!"),
		Message: github.String("initial import"),
	}

	contentResponse, _, err := githubClient.Repositories.CreateFile(ctx, request.RepoOrganization, request.RepoName, "README.md", createFile)

	if err != nil {
		return errors.Wrap(err, "failed to initialize spotguide repository")
	}

	// Prepare the spotguide commit
	spotguideEntries, err := getSpotguideContent(githubClient, request, sourceRepo)
	if err != nil {
		return errors.Wrap(err, "failed to prepare spotguide git content")
	}

	tree, _, err := githubClient.Git.CreateTree(ctx, request.RepoOrganization, request.RepoName, contentResponse.GetSHA(), spotguideEntries)

	if err != nil {
		return errors.Wrap(err, "failed to create git tree for spotguide repository")
	}

	// Create a commit from the tree
	contentResponse.Commit.SHA = contentResponse.SHA

	commit := &github.Commit{
		Message: github.String("initial Banzai Cloud Pipeline commit"),
		Parents: []github.Commit{contentResponse.Commit},
		Tree:    tree,
	}

	newCommit, _, err := githubClient.Git.CreateCommit(ctx, request.RepoOrganization, request.RepoName, commit)

	if err != nil {
		return errors.Wrap(err, "failed to create git commit for spotguide repository")
	}

	// Attach the commit to the master branch.
	// This can be changed later to another branch + create PR.
	// See: https://github.com/google/go-github/blob/master/example/commitpr/main.go#L62
	ref, _, err := githubClient.Git.GetRef(ctx, request.RepoOrganization, request.RepoName, "refs/heads/master")
	if err != nil {
		return errors.Wrap(err, "failed to get git ref for spotguide repository")
	}

	ref.Object.SHA = newCommit.SHA

	_, _, err = githubClient.Git.UpdateRef(ctx, request.RepoOrganization, request.RepoName, ref, false)

	if err != nil {
		return errors.Wrap(err, "failed to update git ref for spotguide repository")
	}

	return nil
}

func createSecrets(request *LaunchRequest, orgID, userID uint) error {

	repoTag := "repo:" + request.RepoFullname()

	for _, secretRequest := range request.Secrets {

		secretRequest.Tags = append(secretRequest.Tags, repoTag)

		if _, err := secret.Store.Store(orgID, secretRequest); err != nil {
			return errors.Wrap(err, "failed to create spotguide secret: "+secretRequest.Name)
		}
	}

	log.Infof("Created secrets for spotguide: %s", request.RepoFullname())

	return nil
}

func enableCICD(request *LaunchRequest, httpRequest *http.Request) error {

	droneClient, err := auth.NewDroneClient(httpRequest)
	if err != nil {
		return errors.Wrap(err, "failed to create Drone client")
	}

	_, err = droneClient.RepoListOpts(true, true)
	if err != nil {
		return errors.Wrap(err, "failed to sync Drone repositories")
	}

	_, err = droneClient.RepoPost(request.RepoOrganization, request.RepoName)
	if err != nil {
		return errors.Wrap(err, "failed to enable Drone repository")
	}

	isSpotguide := true
	spotguideSource := request.RepoFullname()
	repoPatch := drone.RepoPatch{IsSpotguide: &isSpotguide, SpotguideSource: &spotguideSource}
	_, err = droneClient.RepoPatch(request.RepoOrganization, request.RepoName, &repoPatch)
	if err != nil {
		return errors.Wrap(err, "failed to patch Drone repository")
	}

	return nil
}

func createDroneRepoConfig(initConfig []byte, request *LaunchRequest) (*droneRepoConfig, error) {
	repoConfig := new(droneRepoConfig)
	if err := yaml.Unmarshal(initConfig, repoConfig); err != nil {
		return nil, err
	}

	// Configure cluster
	if err := droneRepoConfigCluster(request, repoConfig); err != nil {
		return nil, err
	}

	// Configure secrets
	if err := droneRepoConfigSecrets(request, repoConfig); err != nil {
		return nil, err
	}

	// Configure pipeline
	if err := droneRepoConfigPipeline(request, repoConfig); err != nil {
		return nil, err
	}

	return repoConfig, nil
}

func droneRepoConfigCluster(request *LaunchRequest, repoConfig *droneRepoConfig) error {

	for i, step := range repoConfig.Pipeline {

		// Find CreateClusterStep step and transform it
		if step.Key == CreateClusterStep {

			clusterStep, err := copyToDroneContainer(step.Value)
			if err != nil {
				return err
			}

			// Merge the cluster from the request into the existing cluster value
			cluster, err := json.Marshal(request.Cluster)
			if err != nil {
				return err
			}

			err = json.Unmarshal(cluster, &clusterStep.Cluster)
			if err != nil {
				return err
			}

			newClusterStep, err := droneContainerToMapSlice(clusterStep)
			if err != nil {
				return err
			}

			repoConfig.Pipeline[i].Value = newClusterStep

			return nil
		}
	}

	log.Info("create_cluster step not present in pipeline.yaml, skipping it's transformation")

	return nil
}

func droneRepoConfigSecrets(request *LaunchRequest, repoConfig *droneRepoConfig) error {

	if len(request.Secrets) == 0 {
		return nil
	}

	for _, plugin := range repoConfig.Pipeline {
		for _, secret := range request.Secrets {
			step, err := copyToDroneContainer(plugin.Value)
			if err != nil {
				return err
			}
			step.Secrets = append(step.Secrets, secret.Name)
		}
	}

	return nil
}

func droneRepoConfigPipeline(request *LaunchRequest, repoConfig *droneRepoConfig) error {

	for i, step := range repoConfig.Pipeline {

		stepName := step.Key.(string)

		// Find 'stepName' step and transform it if there are any incoming Values
		if stepToMergeIn, ok := request.Pipeline[stepName]; ok {

			pipelineStep, err := yamlMapSliceToMap(step.Value)
			if err != nil {
				return err
			}

			err = mergo.Merge(&pipelineStep, stepToMergeIn)
			if err != nil {
				return err
			}

			newPipelineStep, err := mapToYamlMapSlice(pipelineStep)
			if err != nil {
				return err
			}

			repoConfig.Pipeline[i].Value = newPipelineStep
		}
	}

	return nil
}
