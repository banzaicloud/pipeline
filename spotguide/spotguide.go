package spotguide

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/secret"
	yaml "github.com/ghodss/yaml"
	"github.com/google/go-github/github"
	"github.com/prometheus/common/log"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

const SpotguideGithubTopic = "spotguide"
const SpotguideGithubOrganization = "banzaicloud"
const SpotguidePath = ".banzaicloud/spotguide.yaml"

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

func ScrapeSpotguides() error {

	db := config.DB()

	githubClient := newGithubClient(viper.GetString("github.token"))

	repositories, _, err := githubClient.Repositories.ListByOrg(ctx, SpotguideGithubOrganization, &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 200},
	})

	if err != nil {
		return err
	}

	for _, repository := range repositories {
		for _, topic := range repository.Topics {
			if topic == SpotguideGithubTopic {
				owner := repository.GetOwner().GetLogin()
				name := repository.GetName()

				reader, err := githubClient.Repositories.DownloadContents(ctx, owner, name, SpotguidePath, nil)
				if err != nil {
					return err
				}

				spotguideRaw, err := ioutil.ReadAll(reader)
				if err != nil {
					return err
				}

				model := Repo{
					Name:         repository.GetFullName(),
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

// curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -v http://localhost:9090/api/v1/orgs/1/spotguides -d '{"repoName":"spotguide-test", "repoOrganization":"banzaicloud"}'
func LaunchSpotguide(request *LaunchRequest, orgID, userID uint) {

	err := createGithubRepo(request, userID)
	if err != nil {
		log.Errorln("Failed to create GitHub repository", err.Error())
		return
	}

	err = createSecrets(request, orgID, userID)
	if err != nil {
		log.Errorln("Failed to create secrets for spotguide", err.Error())
		return
	}
}

func createGithubRepo(request *LaunchRequest, userID uint) error {
	githubClient, err := newGithubClientForUser(userID)
	if err != nil {
		return err
	}

	repo := github.Repository{
		Name:        github.String(request.RepoName),
		Description: github.String("Spotguide by BanzaiCloud"),
	}

	_, _, err = githubClient.Repositories.Create(ctx, request.RepoOrganization, &repo)
	if err != nil {
		return err
	}

	log.Infof("Created spotguide repository: %s/%s", request.RepoOrganization, request.RepoName)

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
