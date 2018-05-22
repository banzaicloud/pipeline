package helm

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io"
	"k8s.io/helm/pkg/repo"
	"regexp"
	"strings"
)

const CatalogRepository = "catalogs"
const CatalogRepositoryUrl = "http://kubernetes-charts.banzaicloud.com/branch/spotguide"

var CatalogPath = "./" + CatalogRepository

type ApplicationDetails struct {
	Resources ApplicationResources `json:"resources"`
	Readme    string               `json:"readme"`
	Options   ApplicationOptions   `json:"options"`
}

type ApplicationOptions struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default bool   `json:"default"`
	Info    string `json:"info"`
	Key     string `json:"key"`
}

type ApplicationDependency struct {
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
	Resources ApplicationResources             `json:"resources"`
	Options   ApplicationOptions               `json:"options"`
	Depends   map[string]ApplicationDependency `json:"depends"`
}

type ApplicationResources struct {
	VCPU               int      `json:"vcpu"`
	Memory             int      `json:"memory"`
	Filters            []string `json:"filters"`
	OnDemandPercentage int      `json:"on_demand_percentage"`
	SameSize           bool     `json:"same_size"`
}

func InitCatalogRepository() error {
	//Init the cluster catalog from a well known repository
	helmEnv := createEnvSettings(CatalogPath)
	if err := ensureDirectories(helmEnv); err != nil {
		return errors.Wrap(err, "Initializing helm directories failed!")
	}
	cr, err := initRepo(CatalogRepository, CatalogRepositoryUrl, helmEnv)
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

func ListCatalogs(queryName, queryVersion, queryKeyword string) ([]repo.ChartVersions, error) {
	repoPath := fmt.Sprintf("%s/repository/repositories.yaml", CatalogPath)
	log.Debug("Helm repo path:", repoPath)

	f, err := repo.LoadRepositoriesFile(repoPath)
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return nil, nil
	}
	catalogs := make([]repo.ChartVersions, 0)
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
			if queryVersion == "latest" {
				catalogs = append(catalogs, repo.ChartVersions{i.Entries[n][0]})
			} else {
				catalogs = append(catalogs, i.Entries[n])
			}
		}

	}
	return catalogs, nil
}

// Fixed repo for catalog
func GetCatalogDetails(name string) (*ChartDetails, error) {
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
			log.Debug("Parsing spotguide.yaml")
			valuesContent := new(bytes.Buffer)
			if _, err := io.Copy(valuesContent, tarReader); err != nil {
				return nil, err
			}
			err := yaml.Unmarshal(valuesContent.Bytes(), so)
			if err != nil {
				return nil, err
			}
			return so, nil
		}
	}
	return so, nil
}
