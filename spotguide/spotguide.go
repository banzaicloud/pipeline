package spotguide

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/ghodss/yaml"
	"github.com/google/go-github/github"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

const SpotguideGithubTopic = "spotguide"
const SpotguideGithubOrganization = "banzaicloud"
const SpotguideYAMLPath = ".banzaicloud/spotguide.yaml"
const PipelineYAMLPath = ".banzaicloud/pipeline.yaml"

var ctx = context.Background()

type Spotguide struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Resources   Resources `json:"resources"`
	Questions   Questions `json:"questions"`
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

type Questions struct {
}

type Repo struct {
	ID           uint       `gorm:"primary_key" json:"-"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `sql:"index" json:"-"`
	Name         string     `json:"name"`
	Icon         string     `json:"-"`
	PipelineRaw  []byte     `json:"-"`
	SpotguideRaw []byte     `json:"-"`
	Spotguide    Spotguide  `gorm:"-" json:"spotguide"`
}

func (Repo) TableName() string {
	return "spotguide_repos"
}

func (s *Repo) AfterFind() error {
	return yaml.Unmarshal(s.SpotguideRaw, &s.Spotguide)
}

type LaunchRequest struct {
	SpotguideName    string   `json:"spotguideName"`
	RepoOrganization string   `json:"repoOrganization"`
	RepoName         string   `json:"repoName"`
	Secrets          []Secret `json:"secrets"`
}

type Secret struct {
	Name string `json:"name"`
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

func downloadGithubFile(githubClient *github.Client, owner, repo, file string) ([]byte, error) {
	reader, err := githubClient.Repositories.DownloadContents(ctx, owner, repo, file, nil)
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

				pipelineRaw, err := downloadGithubFile(githubClient, owner, name, PipelineYAMLPath)
				if err != nil {
					return emperror.Wrap(err, "failed to download pipeline YAML")
				}

				spotguideRaw, err := downloadGithubFile(githubClient, owner, name, SpotguideYAMLPath)
				if err != nil {
					return emperror.Wrap(err, "failed to download spotguide YAML")
				}

				model := Repo{
					Name:         repository.GetFullName(),
					PipelineRaw:  pipelineRaw,
					SpotguideRaw: spotguideRaw,
				}

				err = db.Where(&model).Assign(&model).FirstOrCreate(&Repo{}).Error

				if err != nil {
					return err
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

func GetSpotguide(name string) (*Repo, error) {
	db := config.DB()
	spotguide := Repo{}
	err := db.Where("name = ?", name).Find(&spotguide).Error
	return &spotguide, err
}

// curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -v http://localhost:9090/api/v1/orgs/1/spotguides -d '{"repoName":"spotguide-test", "repoOrganization":"banzaicloud-test", "spotguideName":"banzaicloud/spotguide-nodejs-mongodb"}'
func LaunchSpotguide(request *LaunchRequest, httpRequest *http.Request, orgID, userID uint) error {

	sourceRepo, err := GetSpotguide(request.SpotguideName)
	if err != nil {
		return errors.Wrap(err, "Failed to find spotguide repo")
	}

	err = createGithubRepo(request, userID, sourceRepo)
	if err != nil {
		return errors.Wrap(err, "Failed to create GitHub repository")
	}

	err = createSecrets(request, orgID, userID)
	if err != nil {
		return errors.Wrap(err, "Failed to create secrets for spotguide")
	}

	err = enableCICD(request, httpRequest)
	if err != nil {
		return errors.Wrap(err, "Failed to enable CI/CD for spotguide")
	}

	return nil
}

func createGithubRepo(request *LaunchRequest, userID uint, sourceRepo *Repo) error {
	githubClient, err := newGithubClientForUser(userID)
	if err != nil {
		return errors.Wrap(err, "failed to create GitHub client")
	}

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

	_, _, err = githubClient.Repositories.Create(ctx, orgName, &repo)
	if err != nil {
		return errors.Wrap(err, "failed to create spotguide repository")
	}

	log.Infof("Created spotguide repository: %s/%s", request.RepoOrganization, request.RepoName)

	// An initial files has to be created with the API to be able to use the fresh repo
	createFile := &github.RepositoryContentFileOptions{
		Content: []byte("# Say hello to Spotguides!"),
		Message: github.String("initial import"),
	}

	contentResponse, _, err := githubClient.Repositories.CreateFile(ctx, request.RepoOrganization, request.RepoName, "README.md", createFile)

	if err != nil {
		return errors.Wrap(err, "failed to initialize spotguide repository")
	}

	// Create repo config that drives the CICD flow from LaunchRequest
	repoConfig, err := createDroneRepoConfig(sourceRepo.PipelineRaw, request)
	if err != nil {
		return errors.Wrap(err, "failed to initialize repo config")
	}

	repoConfigRaw, err := yaml.Marshal(repoConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal repo config")
	}

	// List the files here that needs to be created in this commit and create a tree from them
	entries := []github.TreeEntry{
		{
			Type:    github.String("blob"),
			Path:    github.String(PipelineYAMLPath),
			Content: github.String(string(repoConfigRaw)),
			Mode:    github.String("100644"),
		},
		{
			Type:    github.String("blob"),
			Path:    github.String(SpotguideYAMLPath),
			Content: github.String(string(sourceRepo.SpotguideRaw)),
			Mode:    github.String("100644"),
		},
	}

	tree, _, err := githubClient.Git.CreateTree(ctx, request.RepoOrganization, request.RepoName, contentResponse.GetSHA(), entries)

	if err != nil {
		return errors.Wrap(err, "failed to create git tree for spotguide repository")
	}

	// Create a commit from the tree
	contentResponse.Commit.SHA = contentResponse.SHA

	commit := &github.Commit{
		Message: github.String("my first commit from the go client"),
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

	secretTag := "spotguide:" + request.RepoName

	for _, s := range request.Secrets {

		request := secret.CreateSecretRequest{
			Name:   request.RepoName + "-" + s.Name,
			Tags:   []string{secretTag},
			Values: map[string]string{},
		}

		if _, err := secret.Store.Store(orgID, &request); err != nil {
			return err
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
		return errors.Wrap(err, "failed to sync enable Drone repository")
	}

	return nil
}

func createDroneRepoConfig(initConfig []byte, request *LaunchRequest) (*droneRepoConfig, error) {
	repoConfig := new(droneRepoConfig)
	if err := yaml.Unmarshal(initConfig, repoConfig); err != nil {
		return nil, err
	}

	// Configure secrets
	if err := droneRepoConfigSecrets(request, repoConfig); err != nil {
		return nil, err
	}

	return repoConfig, nil
}

func droneRepoConfigSecrets(request *LaunchRequest, repoConfig *droneRepoConfig) error {
	for _, secret := range request.Secrets {
		for _, container := range repoConfig.Pipeline.Containers {
			container.Secrets.Secrets = append(container.Secrets.Secrets, &droneSecret{secret.Name, secret.Name})
		}
	}

	return nil
}
