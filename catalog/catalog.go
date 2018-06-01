package catalog

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/repo"
	"k8s.io/helm/pkg/strvals"
	"regexp"
	"strings"
)

// CatalogRepository for universal catalog repo name
const CatalogRepository = "catalog"

// CatalogRepositoryUrl for universal catalog repo url
const CatalogRepositoryUrl = "http://kubernetes-charts.banzaicloud.com/branch/spotguide"

// CatalogPath TODO check if we need some special config/path
var CatalogPath = "./" + CatalogRepository

//TODO when the API fixed this needs to move to banzai-types

// ApplicationDetails for API response
type ApplicationDetails struct {
	Resources ApplicationResources `json:"resources"`
	Readme    string               `json:"readme"`
	Options   ApplicationOptions   `json:"options"`
}

// ApplicationOptions for API response
type ApplicationOptions struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  bool   `json:"default"`
	Info     string `json:"info"`
	Readonly bool   `json:"readonly"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

// ApplicationDependency for spotguide.yaml
type ApplicationDependency struct {
	Name      string           `json:"name"`
	Type      string           `json:"type"`
	Values    []string         `json:"values"`
	Namespace string           `json:"namespace"`
	Chart     ApplicationChart `json:"chart"`
}

// ApplicationChart for spotguide.yaml
type ApplicationChart struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	Version    string `json:"version"`
}

// SpotguideFile to parse spotguide.yaml
type SpotguideFile struct {
	Resources *ApplicationResources   `json:"resources"`
	Options   []ApplicationOptions    `json:"options"`
	Depends   []ApplicationDependency `json:"depends"`
}

// ApplicationResources to parse spotguide.yaml
type ApplicationResources struct {
	VCPU               int      `json:"vcpu"`
	Memory             int      `json:"memory"`
	Filters            []string `json:"filters"`
	OnDemandPercentage int      `json:"onDemandPercentage"`
	SameSize           bool     `json:"sameSize"`
}

// CatalogDetails for API response
type CatalogDetails struct {
	Name      string             `json:"name"`
	Repo      string             `json:"repo"`
	Chart     *repo.ChartVersion `json:"chart"`
	Values    string             `json:"values"`
	Readme    string             `json:"readme"`
	Spotguide *SpotguideFile     `json:"options"`
}

var log = config.Logger()

// CreateValuesFromOption helper to parse ApplicationOptions into chart values
func CreateValuesFromOption(options []ApplicationOptions) ([]byte, error) {
	base := map[string]interface{}{}
	for _, o := range options {
		set := o.Key + "=" + o.Value
		strvals.ParseIntoString(set, base)
	}
	return yaml.Marshal(base)
}

// GenerateGatalogEnv helper to generate Catalog repo env
func GenerateGatalogEnv(orgName string) helm_env.EnvSettings {
	return helm.CreateEnvSettings(fmt.Sprintf("%s/%s", CatalogPath, orgName))
}

// EnsureCatalog ensure Catalog repo is ready
func EnsureCatalog(env helm_env.EnvSettings) error {
	//Init the cluster catalog from a well known repository
	if err := helm.EnsureDirectories(env); err != nil {
		return errors.Wrap(err, "Initializing helm directories failed!")
	}
	catalogRepo := &repo.Entry{
		Name:  CatalogRepository,
		URL:   CatalogRepositoryUrl,
		Cache: env.Home.CacheIndex(CatalogRepository),
	}
	_, err := helm.ReposAdd(env, catalogRepo)
	if err != nil {
		return err
	}
	return nil
}

// ListCatalogs for API
func ListCatalogs(env helm_env.EnvSettings, queryName, queryVersion, queryKeyword string) ([]repo.ChartVersion, error) {
	if err := EnsureCatalog(env); err != nil {
		return nil, err
	}
	f, err := repo.LoadRepositoriesFile(env.Home.RepositoryFile())
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return nil, nil
	}
	catalogs := make([]repo.ChartVersion, 0)
	i, errIndx := repo.LoadIndexFile(f.Repositories[0].Cache)
	if errIndx != nil {
		return nil, errIndx
	}
	if queryKeyword == "" {
		queryKeyword = "spotguide"
	}
	for n := range i.Entries {
		log.Debugf("Chart: %s", n)
		chartMatched, _ := regexp.MatchString(queryName, strings.ToLower(n))

		kwString := strings.ToLower(strings.Join(i.Entries[n][0].Keywords, " "))
		log.Debugf("kwString: %s", kwString)

		kwMatched, _ := regexp.MatchString(queryKeyword, kwString)
		if (chartMatched || queryName == "") && (kwMatched || queryKeyword == "") {
			log.Debugf("Chart: %s Matched", n)
			catalogs = append(catalogs, *i.Entries[n][0])
		}

	}
	return catalogs, nil
}

// GetCatalogDetails for API
func GetCatalogDetails(env helm_env.EnvSettings, name string) (*CatalogDetails, error) {

	cd, err := ChartGet(env, CatalogRepository, name, "")
	if err != nil {
		return nil, err
	}
	return cd, nil
}

func getChartOption(file []byte) (*SpotguideFile, error) {
	so := &SpotguideFile{}
	tarReader := tar.NewReader(bytes.NewReader(file))
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.Contains(header.Name, "spotguide.json") {
			log.Debug("Parsing spotguide.json")
			valuesContent := new(bytes.Buffer)
			if _, err := io.Copy(valuesContent, tarReader); err != nil {
				return nil, err
			}
			err := json.Unmarshal(valuesContent.Bytes(), so)
			if err != nil {
				return nil, err
			}
			return so, nil
		} else if strings.Contains(header.Name, "spotguide.yaml") {
			log.Debug("Getting spotguide.yaml")
			valuesContent := new(bytes.Buffer)
			if _, err := io.Copy(valuesContent, tarReader); err != nil {
				return nil, err
			}
			log.Debug("Unmarshal spotguide.yaml")
			err := yaml.Unmarshal(valuesContent.Bytes(), so)
			if err != nil {
				return nil, err
			}
			return so, nil
		}

	}
	return nil, nil
}

// ChartGet modifiey helm.ChartGet to injet spotguide
func ChartGet(env helm_env.EnvSettings, chartRepo, chartName, chartVersion string) (*CatalogDetails, error) {
	chartD := &CatalogDetails{}
	f, err := repo.LoadRepositoriesFile(env.Home.RepositoryFile())
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return nil, nil
	}

	for _, r := range f.Repositories {

		log.Debugf("Repository: %s", r.Name)

		i, errIndx := repo.LoadIndexFile(r.Cache)
		if errIndx != nil {
			return nil, errIndx
		}

		if r.Name == chartRepo {

			for n := range i.Entries {
				log.Debugf("Chart: %s", n)
				if chartName == n {

					for _, s := range i.Entries[n] {
						if s.Version == chartVersion || chartVersion == "" {
							chartSource := s.URLs[0]
							log.Debugf("chartSource: %s", chartSource)
							reader, err := helm.DownloadFile(chartSource)
							if err != nil {
								return nil, err
							}
							valuesStr, err := helm.GetChartFile(reader, "values.yaml")
							if err != nil {
								return nil, err
							}
							spotguide, err := getChartOption(reader)
							if err != nil {
								return nil, err
							}
							log.Debugf("values hash: %s", valuesStr)

							readmeStr, err := helm.GetChartFile(reader, "README.md")
							if err != nil {
								return nil, err
							}
							log.Debugf("readme hash: %s", readmeStr)
							chartD = &CatalogDetails{
								Name:      chartName,
								Repo:      chartRepo,
								Chart:     s,
								Values:    valuesStr,
								Readme:    readmeStr,
								Spotguide: spotguide,
							}
							return chartD, nil

						}

					}
				}

			}

		}
	}
	return nil, nil
}
