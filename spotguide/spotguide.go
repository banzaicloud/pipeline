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
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/semver"
	"github.com/Masterminds/sprig"
	"github.com/banzaicloud/cicd-go/cicd"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/client"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/secret"
	yaml2 "github.com/ghodss/yaml"
	"github.com/google/go-github/github"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const SpotguideGithubTopic = "spotguide"
const SpotguideYAMLPath = ".banzaicloud/spotguide.yaml"
const PipelineYAMLPath = ".banzaicloud/pipeline.yaml"
const ReadmePath = ".banzaicloud/README.md"
const IconPath = ".banzaicloud/icon.svg"
const CreateClusterStep = "create_cluster"
const SpotguideRepoTableName = "spotguide_repos"

var IgnoredPaths = []string{".circleci", ".github"}

var ctx = context.Background()

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
	Icon             []byte    `json:"-" gorm:"type:mediumblob"`
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
	RepoLatent       bool                          `json:"repoLatent"`
	Cluster          *client.CreateClusterRequest  `json:"cluster" binding:"required"`
	Secrets          []*secret.CreateSecretRequest `json:"secrets,omitempty"`
	Pipeline         map[string]interface{}        `json:"pipeline,omitempty"`
}

func (r LaunchRequest) RepoFullname() string {
	return r.RepoOrganization + "/" + r.RepoName
}

type ReleaseBody struct {
	Pipeline string `json:"pipeline" yaml:"pipeline"`
}

// SpotguideManager is responsible to scrape spotguides on GitHub and persist them to database
type SpotguideManager struct {
	db                        *gorm.DB
	pipelineVersion           *semver.Version
	githubToken               string
	sharedLibraryOrganization *auth.Organization
}

func NewSpotguideManager(db *gorm.DB, pipelineVersionString string, githubToken string, sharedLibraryGitHubOrganization string) *SpotguideManager {
	sharedLibraryOrganization, err := auth.GetOrganizationByName(sharedLibraryGitHubOrganization)
	if err != nil {
		log.Errorf("shared spotguide organization (%s) is not found", sharedLibraryGitHubOrganization)
	}

	pipelineVersion, _ := semver.NewVersion(pipelineVersionString)

	return &SpotguideManager{
		db:                        db,
		pipelineVersion:           pipelineVersion,
		githubToken:               githubToken,
		sharedLibraryOrganization: sharedLibraryOrganization,
	}
}

func (s *SpotguideManager) isSpotguideReleaseAllowed(release *github.RepositoryRelease) bool {
	version, err := semver.NewVersion(release.GetTagName())
	if err != nil {
		log.Warn("failed to parse spotguide release tag: ", err)
		return false
	}

	supported := true
	prerelease := version.Prerelease() != "" || *release.Prerelease

	// try to parse release body as YAML
	rawBody := release.GetBody()
	body := ReleaseBody{}
	err = yaml2.Unmarshal([]byte(rawBody), &body)
	if s.pipelineVersion != nil && err == nil {
		// check whether this release has support for this pipeline version
		supportedConstraint, err := semver.NewConstraint(body.Pipeline)
		if err == nil {
			supported = supportedConstraint.Check(s.pipelineVersion)
		}
	}

	return supported && (!prerelease || viper.GetBool(config.SpotguideAllowPrereleases))
}

func (s *SpotguideManager) ScrapeSharedSpotguides() error {
	if s.sharedLibraryOrganization == nil {
		return errors.New("failed to scrape shared spotguides")
	}

	githubClient := auth.NewGithubClient(s.githubToken)
	return s.scrapeSpotguides(s.sharedLibraryOrganization, githubClient)
}

func (s *SpotguideManager) ScrapeSpotguides(orgID uint, userID uint) error {
	githubClient, err := auth.NewGithubClientForUser(userID)
	if err != nil {
		return emperror.Wrap(err, "failed to create GitHub client")
	}

	org, err := auth.GetOrganizationById(orgID)
	if err != nil {
		return emperror.Wrap(err, "failed to resolve organization from id")
	}

	return s.scrapeSpotguides(org, githubClient)
}

func (s *SpotguideManager) scrapeSpotguides(org *auth.Organization, githubClient *github.Client) error {
	var allRepositories []github.Repository
	query := fmt.Sprintf("org:%s topic:%s fork:true", org.Name, SpotguideGithubTopic)
	if !viper.GetBool(config.SpotguideAllowPrivateRepos) {
		query += " is:public"
	}
	listOpts := github.ListOptions{PerPage: 100}
	for {
		reposRes, resp, err := githubClient.Search.Repositories(ctx, query, &github.SearchOptions{
			Sort:        "created",
			Order:       "asc",
			ListOptions: listOpts,
		})

		if err != nil {
			// Empty organization, no repositories
			if resp.StatusCode == http.StatusUnprocessableEntity {
				return nil
			}

			return emperror.Wrap(err, "failed to list github repositories")
		}

		allRepositories = append(allRepositories, reposRes.Repositories...)

		if resp.NextPage == 0 {
			break
		}

		listOpts.Page = resp.NextPage
	}

	where := SpotguideRepo{
		OrganizationID: org.ID,
	}

	var oldSpotguides []SpotguideRepo
	if err := s.db.Where(&where).Find(&oldSpotguides).Error; err != nil {
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

			if !s.isSpotguideReleaseAllowed(release) {
				continue
			}

			tag := release.GetTagName()

			spotguideRaw, err := downloadGithubFile(githubClient, owner, name, SpotguideYAMLPath, tag)
			if err != nil {
				log.Warnf("failed to scrape spotguide.yaml of '%s/%s' at version '%s': %s", owner, name, tag, err)
				continue
			}

			// syntax check spotguide.yaml
			err = yaml2.Unmarshal(spotguideRaw, &SpotguideYAML{})
			if err != nil {
				log.Warnf("failed to parse spotguide.yaml of '%s/%s' at version '%s': %s", owner, name, tag, err)
				continue
			}

			readme, err := downloadGithubFile(githubClient, owner, name, ReadmePath, tag)
			if err != nil {
				log.Warnf("failed to scrape the readme of '%s/%s' at version '%s': %s", owner, name, tag, err)
			}

			icon, err := downloadGithubFile(githubClient, owner, name, IconPath, tag)
			if err != nil {
				log.Warnf("failed to scrape the icon of '%s/%s' at version '%s': %s", owner, name, tag, err)
			}

			model := SpotguideRepo{
				OrganizationID:   org.ID,
				Name:             repository.GetFullName(),
				SpotguideYAMLRaw: spotguideRaw,
				Readme:           string(readme),
				Icon:             icon,
				Version:          tag,
			}

			where := model.Key()

			err = s.db.Where(&where).Assign(&model).FirstOrCreate(&SpotguideRepo{}).Error

			if err != nil {
				return err
			}

			delete(oldSpotguidesIndexed, model.Key())
		}
	}

	for spotguideRepoKey := range oldSpotguidesIndexed {
		err := s.db.Where(&spotguideRepoKey).Delete(SpotguideRepo{}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SpotguideManager) GetSpotguides(orgID uint) (spotguides []*SpotguideRepo, err error) {
	query := s.db.Where(SpotguideRepo{OrganizationID: orgID})
	if s.sharedLibraryOrganization != nil {
		query = query.Or(SpotguideRepo{OrganizationID: s.sharedLibraryOrganization.ID})
	}

	err = query.Find(&spotguides).Error
	return spotguides, err
}

func (s *SpotguideManager) GetSpotguide(orgID uint, name, version string) (*SpotguideRepo, error) {
	query := s.db.Where(SpotguideRepo{OrganizationID: orgID, Name: name, Version: version})
	if s.sharedLibraryOrganization != nil {
		query = query.Or(SpotguideRepo{OrganizationID: s.sharedLibraryOrganization.ID, Name: name, Version: version})
	}

	spotguide := SpotguideRepo{}
	err := query.First(&spotguide).Error
	if err != nil {
		return nil, emperror.Wrap(err, "failed to find spotguide")
	}

	return &spotguide, nil
}

func (s *SpotguideManager) LaunchSpotguide(request *LaunchRequest, httpRequest *http.Request, orgID, userID uint) error {
	sourceRepo, err := s.GetSpotguide(orgID, request.SpotguideName, request.SpotguideVersion)
	if err != nil {
		return errors.Wrap(err, "failed to find spotguide repo")
	}

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
	repoConfig, err := createCICDRepoConfig(pipelineYAML, request)
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

	// Support private repositories via downloading with an authenticated client
	downloadRequest, err := http.NewRequest(http.MethodGet, sourceRelease.GetZipballURL(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create source spotguide repository release download request")
	}

	repoBytes := bytes.NewBuffer(nil)
	_, err = githubClient.Do(ctx, downloadRequest, repoBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download source spotguide repository release")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(repoBytes.Bytes()), int64(repoBytes.Len()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract source spotguide repository release")
	}

	// List the files here that needs to be created in this commit and create a tree from them
	entries := []github.TreeEntry{}

	for _, zf := range zipReader.File {
		if zf.FileInfo().IsDir() {
			continue
		}

		// First directory is the name of the repo
		path := strings.SplitN(zf.Name, "/", 2)[1]

		// Skip files inside ignored paths
		if isIgnoredPath(path) {
			continue
		}

		file, err := zf.Open()
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract source spotguide repository release")
		}
		defer file.Close()

		content, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract source spotguide repository release")
		}

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
		Description: github.String("Spotguide by Banzai Cloud"),
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

	log.Infof("created spotguide repository: %s", request.RepoFullname())
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

	log.Infof("created secrets for spotguide: %s", request.RepoFullname())

	return nil
}

func enableCICD(request *LaunchRequest, httpRequest *http.Request) error {

	cicdClient, err := auth.NewCICDClient(httpRequest)
	if err != nil {
		return errors.Wrap(err, "failed to create CICD client")
	}

	_, err = cicdClient.RepoListOpts(true, true)
	if err != nil {
		return errors.Wrap(err, "failed to sync CICD repositories")
	}

	_, err = cicdClient.RepoPost(request.RepoOrganization, request.RepoName)
	if err != nil {
		return errors.Wrap(err, "failed to enable CICD repository")
	}

	repoPatch := cicd.RepoPatch{
		IsSpotguide:     github.Bool(true),
		SpotguideSource: github.String(request.SpotguideName),
	}
	if request.RepoLatent {
		repoPatch.AllowTag = github.Bool(false)
		repoPatch.AllowPull = github.Bool(false)
		repoPatch.AllowPush = github.Bool(false)
		repoPatch.AllowDeploy = github.Bool(false)
	}
	_, err = cicdClient.RepoPatch(request.RepoOrganization, request.RepoName, &repoPatch)
	if err != nil {
		return errors.Wrap(err, "failed to patch CICD repository")
	}

	return nil
}

func createCICDRepoConfig(pipelineYAML []byte, request *LaunchRequest) (*cicdRepoConfig, error) {
	// Pre-process pipeline.yaml
	yamlTemplate, err := template.New("pipeline.yaml").
		Delims("{{{{", "}}}}").
		Funcs(sprig.TxtFuncMap()).
		Parse(string(pipelineYAML))
	if err != nil {
		return nil, emperror.Wrap(err, "failed to setup sprig template for pipeline.yaml")
	}
	buffer := bytes.NewBuffer(nil)

	data := map[string]map[string]interface{}{}
	cluster := map[string]interface{}{}
	pipeline := map[string]interface{}{}

	clusterDecoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{TagName: "json", Result: &cluster})
	if err != nil {
		return nil, emperror.Wrap(err, "failed to merge cluster into sprig template data")
	}

	err = clusterDecoder.Decode(request.Cluster)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to merge cluster into sprig template data")
	}

	pipelineDecoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{TagName: "json", Result: &pipeline})
	if err != nil {
		return nil, emperror.Wrap(err, "failed to merge pipeline into sprig template data")
	}

	err = pipelineDecoder.Decode(request.Pipeline)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to merge pipeline into sprig template data")
	}

	data["cluster"] = cluster
	data["pipeline"] = pipeline

	err = yamlTemplate.Execute(buffer, data)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to evaluate sprig template for pipeline.yaml")
	}

	repoConfig := new(cicdRepoConfig)
	if err := yaml.Unmarshal(buffer.Bytes(), repoConfig); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal initial config")
	}

	// Configure cluster
	if err := cicdRepoConfigCluster(request, repoConfig); err != nil {
		return nil, errors.Wrap(err, "failed to add cluster details")
	}

	// Configure secrets
	if err := cicdRepoConfigSecrets(request, repoConfig); err != nil {
		return nil, errors.Wrap(err, "failed to add secrets to steps")
	}

	// Configure pipeline
	if err := cicdRepoConfigPipeline(request, repoConfig); err != nil {
		return nil, errors.Wrap(err, "failed to merge values")
	}

	return repoConfig, nil
}

func cicdRepoConfigCluster(request *LaunchRequest, repoConfig *cicdRepoConfig) error {

	clusterJson, err := json.Marshal(request.Cluster)
	if err != nil {
		return err
	}

	for i, step := range repoConfig.Pipeline {

		// Find CreateClusterStep step and transform it
		if step.Key == CreateClusterStep {
			log.Debugf("merge cluster info to %q step", step.Key)

			clusterStep, err := copyToCICDContainer(step.Value)
			if err != nil {
				return err
			}

			// Merge the cluster from the request into the existing cluster value
			err = json.Unmarshal(clusterJson, &clusterStep.Cluster)
			if err != nil {
				return err
			}

			newClusterStep, err := cicdContainerToMapSlice(clusterStep)
			if err != nil {
				return err
			}

			repoConfig.Pipeline[i].Value = newClusterStep

			return nil
		}
	}

	log.Debug("merge cluster info to cluster block")

	// Merge the cluster from the request into the cluster block
	if err := json.Unmarshal(clusterJson, &repoConfig.Cluster); err != nil {
		return err
	}
	return nil
}

func cicdRepoConfigSecrets(request *LaunchRequest, repoConfig *cicdRepoConfig) error {
	if len(request.Secrets) == 0 {
		return nil
	}

	for _, plugin := range repoConfig.Pipeline {
		for _, secret := range request.Secrets {
			step, err := copyToCICDContainer(plugin.Value)
			if err != nil {
				return err
			}
			step.Secrets = append(step.Secrets, secret.Name)
		}
	}

	return nil
}

func cicdRepoConfigPipeline(request *LaunchRequest, repoConfig *cicdRepoConfig) error {
	for i, step := range repoConfig.Pipeline {
		stepName := step.Key.(string)

		// Find 'stepName' step and transform it if there are any incoming Values
		if stepToMergeIn, ok := request.Pipeline[stepName]; ok {
			pipelineStep, err := yamlMapSliceToMap(step.Value)
			if err != nil {
				return err
			}

			merged, err := merge(pipelineStep, stepToMergeIn)
			if err != nil {
				return err
			}

			newPipelineStep, err := mapToYamlMapSlice(merged)
			if err != nil {
				return err
			}

			repoConfig.Pipeline[i].Value = newPipelineStep
		}
	}

	return nil
}

func downloadGithubFile(githubClient *github.Client, owner, repo, file, tag string) ([]byte, error) {
	reader, err := githubClient.Repositories.DownloadContents(ctx, owner, repo, file, &github.RepositoryContentGetOptions{
		Ref: tag,
	})
	if err != nil {
		return nil, emperror.Wrap(err, "failed to download file from GitHub")
	}

	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	return data, emperror.Wrap(err, "failed to download file from GitHub")
}

func isIgnoredPath(path string) bool {
	for _, ignoredPath := range IgnoredPaths {
		if path == ignoredPath || strings.HasPrefix(path, ignoredPath+string(os.PathSeparator)) {
			return true
		}
	}

	return false
}
