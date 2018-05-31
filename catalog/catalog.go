package catalog

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io"
	"k8s.io/helm/pkg/repo"
	"k8s.io/helm/pkg/strvals"
	"regexp"
	"strings"
)

const CatalogRepository = "catalog_repository"
const CatalogRepositoryUrl = "http://kubernetes-charts.banzaicloud.com/branch/spotguide"

var CatalogPath = "./" + CatalogRepository

type ApplicationDetails struct {
	Resources ApplicationResources `json:"resources"`
	Readme    string               `json:"readme"`
	Options   ApplicationOptions   `json:"options"`
}

type ApplicationOptions struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  bool   `json:"default"`
	Info     string `json:"info"`
	Readonly bool   `json:"readonly"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

type ApplicationDependency struct {
	Name      string           `json:"name"`
	Type      string           `json:"type"`
	Values    []string         `json:"values"`
	Namespace string           `json:"namespace"`
	Chart     ApplicationChart `json:"chart"`
}

type ApplicationChart struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	Version    string `json:"version"`
}

type SpotguideFile struct {
	Resources ApplicationResources    `json:"resources"`
	Options   []ApplicationOptions    `json:"options"`
	Depends   []ApplicationDependency `json:"depends"`
}

type ApplicationResources struct {
	VCPU               int      `json:"vcpu"`
	Memory             int      `json:"memory"`
	Filters            []string `json:"filters"`
	OnDemandPercentage int      `json:"on_demand_percentage"`
	SameSize           bool     `json:"same_size"`
}

var logger *logrus.Logger
var log *logrus.Entry

func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"action": "Helm"})
}

func CreateValuesFromOption(options []ApplicationOptions) ([]byte, error) {
	base := map[string]interface{}{}
	for _, o := range options {
		set := o.Key + "=" + o.Value
		strvals.ParseIntoString(set, base)
	}
	return yaml.Marshal(base)
}

func InitCatalogRepository() error {
	//Init the cluster catalog from a well known repository
	helmEnv := helm.CreateEnvSettings(CatalogPath)
	if err := helm.EnsureDirectories(helmEnv); err != nil {
		return errors.Wrap(err, "Initializing helm directories failed!")
	}
	cr, err := helm.InitRepo(CatalogRepository, CatalogRepositoryUrl, helmEnv)
	if err != nil {
		return err
	}
	repoFile := helmEnv.Home.RepositoryFile()
	f := repo.NewRepoFile()
	f.Add(cr)
	if err := f.WriteFile(repoFile, 0644); err != nil {
		return errors.Wrap(err, "cannot create file")
	}
	return nil
}

func ListCatalogs(queryName, queryVersion, queryKeyword string) ([]repo.ChartVersion, error) {
	repoPath := fmt.Sprintf("%s/repository/repositories.yaml", CatalogPath)
	log.Debug("Helm repo path:", repoPath)

	f, err := repo.LoadRepositoriesFile(repoPath)
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

// Fixed repo for catalog
func GetCatalogDetails(name string) (*CatalogDetails, error) {
	cd, err := ChartGet(CatalogPath, CatalogRepository, name, "")
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

type CatalogDetails struct {
	Name      string             `json:"name"`
	Repo      string             `json:"repo"`
	Chart     *repo.ChartVersion `json:"chart"`
	Values    string             `json:"values"`
	Readme    string             `json:"readme"`
	Spotguide *SpotguideFile     `json:"options"`
}

func ChartGet(path, chartRepo, chartName, chartVersion string) (*CatalogDetails, error) {

	repoPath := fmt.Sprintf("%s/repository/repositories.yaml", path)
	log.Debug("Helm repo path:", repoPath)
	chartD := &CatalogDetails{}
	f, err := repo.LoadRepositoriesFile(repoPath)
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
