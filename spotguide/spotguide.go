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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/drone/drone-go/drone"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/client"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/google/go-github/github"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

const SpotguideGithubTopic = "spotguide"
const SpotguideGithubOrganization = "banzaicloud"
const SpotguideYAMLPath = ".banzaicloud/spotguide.yaml"
const PipelineYAMLPath = ".banzaicloud/pipeline.yaml"
const CreateClusterStep = "create_cluster"
const DeployApplicationStep = "deploy_application"

var ctx = context.Background()

type Spotguide struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Tags        []string   `json:"tags"`
	Resources   Resources  `json:"resources"`
	Questions   []Question `json:"questions"`
}

type Resources struct {
	CPU         int      `json:"sumCpu"`
	Memory      int      `json:"sumMem"`
	Filters     []string `json:"filters"`
	SameSize    bool     `json:"sameSize"`
	OnDemandPct int      `json:"onDemandPct"`
	MinNodes    int      `json:"minNodes"`
	MaxNodes    int      `json:"maxNodes"`
}

type Question map[string]interface{}

type Repo struct {
	ID           uint       `gorm:"primary_key" json:"-"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"-" gorm:"index"`
	Name         string     `json:"name" gorm:"unique_index:name_and_version"`
	Icon         string     `json:"-"`
	SpotguideRaw []byte     `json:"-" gorm:"type:text"`
	Spotguide    Spotguide  `json:"spotguide" gorm:"-"`
	Version      string     `json:"version" gorm:"unique_index:name_and_version"`
}

func (Repo) TableName() string {
	return "spotguide_repos"
}

func (s *Repo) AfterFind() error {
	return yaml.Unmarshal(s.SpotguideRaw, &s.Spotguide)
}

type LaunchRequest struct {
	SpotguideName    string                       `json:"spotguideName" binding:"required"`
	SpotguideVersion string                       `json:"spotguideVersion"`
	RepoOrganization string                       `json:"repoOrganization" binding:"required"`
	RepoName         string                       `json:"repoName" binding:"required"`
	Cluster          *client.CreateClusterRequest `json:"cluster"`
	Secrets          []secret.CreateSecretRequest `json:"secrets"`
	Values           map[string]interface{}       `json:"values"` // Values passed to the Helm deployment in the 'deploy_application' step
}

func (r LaunchRequest) RepoFullname() string {
	return r.RepoOrganization + "/" + r.RepoName
}

func getUserGithubToken(userID uint) (string, error) {
	token, err := auth.TokenStore.Lookup(fmt.Sprint(userID), auth.GithubTokenID)
	if err != nil {
		return "", err
	}
	if token == nil {
		return "", fmt.Errorf("Github token not found for user")
	}
	return token.Value, nil
}

func newGithubClientForUser(userID uint) (*github.Client, error) {
	accessToken, err := getUserGithubToken(userID)
	if err != nil {
		return nil, err
	}

	return newGithubClient(accessToken), nil
}

func newGithubClient(accessToken string) *github.Client {
	httpClient := oauth2.NewClient(
		ctx,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}),
	)

	return github.NewClient(httpClient)
}

func downloadGithubFile(githubClient *github.Client, owner, repo, file, tag string) ([]byte, error) {
	reader, err := githubClient.Repositories.DownloadContents(ctx, owner, repo, file, &github.RepositoryContentGetOptions{
		Ref: tag,
	})
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(reader)
}

func ScrapeSpotguides() error {

	db := config.DB()

	githubClient := newGithubClient(viper.GetString("github.token"))

	var allRepositories []*github.Repository
	listOpts := github.ListOptions{PerPage: 100}
	for {
		repositories, resp, err := githubClient.Repositories.ListByOrg(ctx, SpotguideGithubOrganization, &github.RepositoryListByOrgOptions{
			ListOptions: listOpts,
		})

		if err != nil {
			return emperror.Wrap(err, "failed to list github repositories")
		}

		allRepositories = append(allRepositories, repositories...)

		if resp.NextPage == 0 {
			break
		}

		listOpts.Page = resp.NextPage
	}

	for _, repository := range allRepositories {
		for _, topic := range repository.Topics {
			if topic == SpotguideGithubTopic {
				owner := repository.GetOwner().GetLogin()
				name := repository.GetName()

				releases, _, err := githubClient.Repositories.ListReleases(ctx, owner, name, &github.ListOptions{})
				if err != nil {
					return emperror.Wrap(err, "failed to list github repo releases")
				}
				for _, release := range releases {
					tag := release.GetTagName()

					spotguideRaw, err := downloadGithubFile(githubClient, owner, name, SpotguideYAMLPath, tag)
					if err != nil {
						return emperror.Wrap(err, "failed to download spotguide YAML")
					}

					model := Repo{
						Name:         repository.GetFullName(),
						SpotguideRaw: spotguideRaw,
						Version:      tag,
					}

					err = db.Where(&model).Assign(&model).FirstOrCreate(&Repo{}).Error

					if err != nil {
						return err
					}
				}

				break
			}
		}
	}

	return nil
}

func GetSpotguides() ([]*Repo, error) {
	db := config.DB()
	spotguides := []*Repo{}
	err := db.Find(&spotguides).Error
	return spotguides, err
}

func GetSpotguide(name, version string) (repo *Repo, err error) {
	db := config.DB()
	repo = &Repo{}
	if version == "" {
		err = db.Where("name = ?", name).Last(repo).Error
	} else {
		err = db.Where("name = ? AND version = ?", name, version).Find(repo).Error
	}
	return
}

// curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -v http://localhost:9090/api/v1/orgs/1/spotguides -d '{"repoName":"spotguide-test", "repoOrganization":"banzaicloud-test", "spotguideName":"banzaicloud/spotguide-nodejs-mongodb"}'
func LaunchSpotguide(request *LaunchRequest, httpRequest *http.Request, orgID, userID uint) error {

	sourceRepo, err := GetSpotguide(request.SpotguideName, request.SpotguideVersion)
	if err != nil {
		return errors.Wrap(err, "Failed to find spotguide repo")
	}

	// LaunchRequest might not have the version
	request.SpotguideVersion = sourceRepo.Version

	err = createSecrets(request, orgID, userID)
	if err != nil {
		return errors.Wrap(err, "Failed to create secrets for spotguide")
	}

	githubClient, err := newGithubClientForUser(userID)
	if err != nil {
		return errors.Wrap(err, "Failed to create GitHub client")
	}

	err = createGithubRepo(githubClient, request, userID, sourceRepo)
	if err != nil {
		return errors.Wrap(err, "Failed to create GitHub repository")
	}

	err = enableCICD(request, httpRequest)
	if err != nil {
		return errors.Wrap(err, "Failed to enable CI/CD for spotguide")
	}

	err = addSpotguideContent(githubClient, request, userID, sourceRepo)
	if err != nil {
		return errors.Wrap(err, "Failed to add spotguide content to repository")
	}

	return nil
}

func preparePipelineYAML(request *LaunchRequest, sourceRepo *Repo, pipelineYAML []byte) ([]byte, error) {
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

func getSpotguideContent(githubClient *github.Client, request *LaunchRequest, sourceRepo *Repo) ([]github.TreeEntry, error) {
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

		path := strings.SplitN(zf.Name, "/", 2)[1]

		// We don't want to prepare yet, use the same pipeline.yml
		if path == PipelineYAMLPath {
			content, err = preparePipelineYAML(request, sourceRepo, content)
			if err != nil {
				return nil, errors.Wrap(err, "failed to prepare pipeline.yaml")
			}
		}

		entry := github.TreeEntry{
			Type:    github.String("blob"),
			Path:    github.String(path),
			Content: github.String(string(content)),
			Mode:    github.String("100644"),
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func createGithubRepo(githubClient *github.Client, request *LaunchRequest, userID uint, sourceRepo *Repo) error {

	repo := github.Repository{
		Name:        github.String(request.RepoName),
		Description: github.String("Spotguide by BanzaiCloud"),
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

	log.Infof("Created spotguide repository: %s/%s", request.RepoOrganization, request.RepoName)
	return nil
}

func addSpotguideContent(githubClient *github.Client, request *LaunchRequest, userID uint, sourceRepo *Repo) error {

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
		Message: github.String("adding spotguide structure"),
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

		if _, err := secret.Store.Store(orgID, &secretRequest); err != nil {
			return errors.Wrap(err, "failed to create spotguide secret: "+secretRequest.Name)
		}
	}

	log.Infof("Created secrets for spotguide: %s/%s", request.RepoOrganization, request.RepoName)

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
	repoPatch := drone.RepoPatch{IsSpotguide: &isSpotguide}
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

	// Configure values
	if err := droneRepoConfigValues(request, repoConfig); err != nil {
		return nil, err
	}

	return repoConfig, nil
}

func droneRepoConfigCluster(request *LaunchRequest, repoConfig *droneRepoConfig) error {

	// Find CreateClusterStep step and transform it if there are is an incoming Cluster
	if clusterStep, ok := repoConfig.Pipeline[CreateClusterStep]; ok && request.Cluster != nil {

		// Merge the cluster from the request into the existing cluster value
		cluster, err := json.Marshal(request.Cluster)
		if err != nil {
			return err
		}

		err = json.Unmarshal(cluster, &clusterStep.Cluster)
		if err != nil {
			return err
		}
	} else {
		log.Info("create_cluster step not present in pipeline.yaml, skipping transformation")
	}
	return nil
}

func droneRepoConfigSecrets(request *LaunchRequest, repoConfig *droneRepoConfig) error {

	if len(request.Secrets) == 0 {
		return nil
	}

	for _, plugin := range repoConfig.Pipeline {
		for _, secret := range request.Secrets {
			plugin.Secrets = append(plugin.Secrets, secret.Name)
		}
	}

	return nil
}

func droneRepoConfigValues(request *LaunchRequest, repoConfig *droneRepoConfig) error {
	// Find DeployApplicationStep step and transform it if there are any incoming Values
	if deployStep, ok := repoConfig.Pipeline[DeployApplicationStep]; ok && len(request.Values) > 0 {

		// Merge the values from the request into the existing values
		values, err := json.Marshal(request.Values)
		if err != nil {
			return err
		}

		err = json.Unmarshal(values, &deployStep.Deployment.Values)
		if err != nil {
			return err
		}
	} else {
		log.Info("deploy_application step not present in pipeline.yaml, skipping transformation")
	}
	return nil
}
