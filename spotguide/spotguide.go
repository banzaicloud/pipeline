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
	"github.com/banzaicloud/pipeline/spotguide/scm"
	yaml2 "github.com/ghodss/yaml"
	"github.com/google/go-github/github"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
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

var IgnoredPaths = []string{".circleci", ".github"} // nolint: gochecknoglobals

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
	scmFactory                scm.SCMFactory
	sharedLibraryOrganization *auth.Organization
}

func CreateSharedSpotguideOrganization(db *gorm.DB, scm string, sharedLibraryOrganization string) (*auth.Organization, error) {
	// insert shared organization to DB if not exists
	var sharedOrg *auth.Organization

	switch scm {
	case "github":
		sharedOrg = &auth.Organization{Name: sharedLibraryOrganization, Provider: auth.ProviderGithub}
	case "gitlab":
		sharedOrg = &auth.Organization{Name: sharedLibraryOrganization, Provider: auth.ProviderGitlab}
	}

	if err := db.Where(sharedOrg).FirstOrCreate(sharedOrg).Error; err != nil {
		return nil, emperror.Wrap(err, "failed to create shared organization")
	}
	return sharedOrg, nil
}

func NewSpotguideManager(db *gorm.DB, pipelineVersionString string, scmFactory scm.SCMFactory, sharedLibraryOrganization *auth.Organization) *SpotguideManager {

	pipelineVersion, _ := semver.NewVersion(pipelineVersionString)

	return &SpotguideManager{
		db:                        db,
		pipelineVersion:           pipelineVersion,
		scmFactory:                scmFactory,
		sharedLibraryOrganization: sharedLibraryOrganization,
	}
}

func (s *SpotguideManager) isSpotguideReleaseAllowed(release scm.RepositoryRelease) bool {
	version, err := semver.NewVersion(release.GetTag())
	if err != nil {
		log.Warn("failed to parse spotguide release tag: ", err)
		return false
	}

	supported := true
	prerelease := version.Prerelease() != "" || release.IsPreRelease()

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
		return fmt.Errorf("failed to scrape shared spotguides")
	}

	sharedSCM, err := s.scmFactory.CreateSharedSCM()
	if err != nil {
		return emperror.Wrap(err, "failed to create SCM client")
	}

	return s.scrapeSpotguides(s.sharedLibraryOrganization, sharedSCM)
}

func (s *SpotguideManager) ScrapeSpotguides(orgID uint, userID uint) error {

	org, err := auth.GetOrganizationById(orgID)
	if err != nil {
		return emperror.Wrap(err, "failed to resolve organization from id")
	}

	userSCM, err := s.scmFactory.CreateUserSCM(userID)
	if err != nil {
		return emperror.Wrap(err, "failed to create SCM client")
	}

	return s.scrapeSpotguides(org, userSCM)
}

func (s *SpotguideManager) scrapeSpotguides(org *auth.Organization, scm scm.SCM) error {

	allowPrivate := viper.GetBool(config.SpotguideAllowPrivateRepos)

	allRepositories, err := scm.ListRepositoriesByTopic(org.Name, SpotguideGithubTopic, allowPrivate)

	if err != nil {
		return emperror.Wrap(err, "failed to list spotguide repositories")
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
		owner := repository.GetOwner()
		name := repository.GetName()

		releases, err := scm.ListRepositoryReleases(owner, name)
		if err != nil {
			return emperror.Wrap(err, "failed to list repo releases")
		}

		for _, release := range releases {

			if !s.isSpotguideReleaseAllowed(release) {
				continue
			}

			tag := release.GetTag()

			spotguideRaw, err := scm.DownloadFile(owner, name, SpotguideYAMLPath, tag)
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

			readme, err := scm.DownloadFile(owner, name, ReadmePath, tag)
			if err != nil {
				log.Warnf("failed to scrape the readme of '%s/%s' at version '%s': %s", owner, name, tag, err)
			}

			icon, err := scm.DownloadFile(owner, name, IconPath, tag)
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

	log.Infof("Finished scraping spotguides for organization '%s'", org.Name)

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

func (s *SpotguideManager) LaunchSpotguide(request *LaunchRequest, org *auth.Organization, user *auth.User) error {
	sourceRepo, err := s.GetSpotguide(org.ID, request.SpotguideName, request.SpotguideVersion)
	if err != nil {
		return emperror.Wrap(err, "failed to find spotguide repo")
	}

	// LaunchRequest might not have the version
	request.SpotguideVersion = sourceRepo.Version

	err = createSecrets(request, org.ID, user.ID)
	if err != nil {
		return emperror.Wrap(err, "failed to create secrets for spotguide")
	}

	userSCM, err := s.scmFactory.CreateUserSCM(user.ID)
	if err != nil {
		return emperror.Wrap(err, "failed to create SCM client")
	}

	err = userSCM.CreateRepository(request.RepoOrganization, request.RepoName, request.RepoPrivate, user.ID)
	if err != nil {
		return emperror.Wrap(err, "failed to create repository")
	}

	log.Infof("created spotguide repository: %s/%s", request.RepoOrganization, request.RepoName)

	cicdClient := auth.NewCICDClient(user.APIToken)

	err = enableCICD(cicdClient, request, org.Name)
	if err != nil {
		return emperror.Wrap(err, "failed to enable CI/CD for spotguide")
	}

	// Prepare the spotguide content
	spotguideContent, err := getSpotguideContent(userSCM, request, sourceRepo)
	if err != nil {
		return emperror.Wrap(err, "failed to prepare spotguide git content")
	}

	err = userSCM.AddContentToRepository(request.RepoOrganization, request.RepoName, spotguideContent)
	if err != nil {
		return emperror.Wrap(err, "failed to add spotguide content to repository")
	}

	log.Infof("added spotguide content to repository: %s/%s", request.RepoOrganization, request.RepoName)

	return nil
}

func preparePipelineYAML(request *LaunchRequest, sourceRepo *SpotguideRepo, pipelineYAML []byte) ([]byte, error) {
	// Create repo config that drives the CICD flow from LaunchRequest
	repoConfig, err := createCICDRepoConfig(pipelineYAML, request)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to initialize repo config")
	}

	repoConfigRaw, err := yaml.Marshal(repoConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to marshal repo config")
	}

	return repoConfigRaw, nil
}

func getSpotguideContent(sourceSCM scm.SCM, request *LaunchRequest, sourceRepo *SpotguideRepo) ([]scm.RepositoryFile, error) {
	// Download source repo zip
	sourceRepoParts := strings.Split(sourceRepo.Name, "/")
	sourceRepoOwner := sourceRepoParts[0]
	sourceRepoName := sourceRepoParts[1]

	repoBytes, err := sourceSCM.DownloadRelease(sourceRepoOwner, sourceRepoName, request.SpotguideVersion)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to download source spotguide repository release")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(repoBytes), int64(len(repoBytes)))
	if err != nil {
		return nil, emperror.Wrap(err, "failed to extract source spotguide repository release")
	}

	var repoFiles []scm.RepositoryFile

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
			return nil, emperror.Wrap(err, "failed to extract source spotguide repository release")
		}
		defer file.Close()

		content, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to extract source spotguide repository release")
		}

		// Prepare pipeline.yaml
		if path == PipelineYAMLPath {
			content, err = preparePipelineYAML(request, sourceRepo, content)
			if err != nil {
				return nil, emperror.Wrap(err, "failed to prepare pipeline.yaml")
			}
		}

		var encoding, fileContent string

		if strings.HasSuffix(http.DetectContentType(content), "charset=utf-8") {
			fileContent = string(content)
			encoding = scm.EncodingText
		} else {
			fileContent = base64.StdEncoding.EncodeToString(content)
			encoding = scm.EncodingBase64
		}

		repoFile := scm.RepositoryFile{
			Path:     path,
			Encoding: encoding,
			Content:  fileContent,
		}

		repoFiles = append(repoFiles, repoFile)
	}

	return repoFiles, nil
}

func createSecrets(request *LaunchRequest, orgID uint, userID uint) error {

	repoTag := "repo:" + request.RepoFullname()

	for _, secretRequest := range request.Secrets {

		secretRequest.Tags = append(secretRequest.Tags, repoTag)

		if _, err := secret.Store.Store(orgID, secretRequest); err != nil {
			return emperror.WrapWith(err, "failed to create spotguide secret", "name", secretRequest.Name)
		}
	}

	log.Infof("created secrets for spotguide: %s", request.RepoFullname())

	return nil
}

func enableCICD(cicdClient cicd.Client, request *LaunchRequest, org string) error {

	_, err := cicdClient.RepoListOpts(true, true)
	if err != nil {
		return emperror.Wrap(err, "failed to sync CICD repositories")
	}

	_, err = cicdClient.RepoPost(request.RepoOrganization, request.RepoName, org)
	if err != nil {
		return emperror.Wrap(err, "failed to enable CICD repository")
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
		return emperror.Wrap(err, "failed to patch CICD repository")
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
		return nil, emperror.Wrap(err, "failed to unmarshal initial config")
	}

	// Configure cluster
	if err := cicdRepoConfigCluster(request, repoConfig); err != nil {
		return nil, emperror.Wrap(err, "failed to add cluster details")
	}

	// Configure secrets
	if err := cicdRepoConfigSecrets(request, repoConfig); err != nil {
		return nil, emperror.Wrap(err, "failed to add secrets to steps")
	}

	// Configure pipeline
	if err := cicdRepoConfigPipeline(request, repoConfig); err != nil {
		return nil, emperror.Wrap(err, "failed to merge values")
	}

	return repoConfig, nil
}

func cicdRepoConfigCluster(request *LaunchRequest, repoConfig *cicdRepoConfig) error {

	clusterJSON, err := json.Marshal(request.Cluster)
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
			err = json.Unmarshal(clusterJSON, &clusterStep.Cluster)
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
	if err := json.Unmarshal(clusterJSON, &repoConfig.Cluster); err != nil {
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

func isIgnoredPath(path string) bool {
	for _, ignoredPath := range IgnoredPaths {
		if path == ignoredPath || strings.HasPrefix(path, ignoredPath+string(os.PathSeparator)) {
			return true
		}
	}

	return false
}
