package helm

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"archive/tar"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"github.com/Masterminds/sprig"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/repo"
	"net/http"
	"os"
	"regexp"
)

var logger *logrus.Logger
var log *logrus.Entry

var ErrRepoNotFound = errors.New("helm repository not found!")

// Simple init for logging
func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"action": "Helm"})
}

func getChartOption(file *gzip.Reader) (*SpotguideOptions, error) {
	so := &SpotguideOptions{}
	tarReader := tar.NewReader(file)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.Contains(header.Name, "spotguide.json") {
			valuesContent := new(bytes.Buffer)
			if _, err := io.Copy(valuesContent, tarReader); err != nil {
				return nil, err
			}
			err := json.Unmarshal(valuesContent.Bytes(), so)
			if err != nil {
				return nil, err
			}
			return so, nil
		}
	}
	return so, nil
}

func downloadFile(url string) (*gzip.Reader, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	tarContent := new(bytes.Buffer)
	io.Copy(tarContent, resp.Body)
	gzf, err := gzip.NewReader(tarContent)
	if err != nil {
		return nil, err
	}
	return gzf, nil
}

//getChartFile Download file from chart repository
func getChartFile(file *gzip.Reader, fileName string) (string, error) {
	tarReader := tar.NewReader(file)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if strings.Contains(header.Name, fileName) {
			valuesContent := new(bytes.Buffer)
			if _, err := io.Copy(valuesContent, tarReader); err != nil {
				return "", err
			}
			base64Str := base64.StdEncoding.EncodeToString(valuesContent.Bytes())
			return base64Str, nil
		}
	}
	return "", nil
}

//DeleteAllDeployment deletes all Helm deployment
func DeleteAllDeployment(kubeconfig []byte) error {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteAllDeployment"})
	log.Info("Getting deployments....")
	filter := ""
	releaseResp, err := ListDeployments(&filter, kubeconfig)
	if err != nil {
		return err
	}
	log.Info("Starting deleting deployments")
	for _, r := range releaseResp.Releases {
		log.Info("Trying to delete deployment ", r.Name)
		err := DeleteDeployment(r.Name, kubeconfig)
		if err != nil {
			return err
		}
		log.Infof("Deployment %s successfully deleted", r.Name)
	}
	return nil
}

//ListDeployments lists Helm deployments
func ListDeployments(filter *string, kubeConfig []byte) (*rls.ListReleasesResponse, error) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagListDeployments})
	hClient, err := GetHelmClient(kubeConfig)
	// TODO doc the options here
	var sortBy = int32(2)
	var sortOrd = int32(1)
	ops := []helm.ReleaseListOption{
		helm.ReleaseListSort(sortBy),
		helm.ReleaseListOrder(sortOrd),
		//helm.ReleaseListLimit(limit),
		//helm.ReleaseListFilter(filter),
		//helm.ReleaseListStatuses(codes),
		//helm.ReleaseListNamespace(""),
	}
	if filter != nil {
		log.Debug("Apply filters: ", filter)
		ops = append(ops, helm.ReleaseListFilter(*filter))
	}
	if err != nil {
		return nil, err
	}
	resp, err := hClient.ListReleases(ops...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

//UpgradeDeployment upgrades a Helm deployment
func UpgradeDeployment(deploymentName, chartName string, values []byte, reuseValues bool, kubeConfig []byte, path string) (*rls.UpdateReleaseResponse, error) {
	//Map chartName as

	downloadedChartPath, err := downloadChartFromRepo(chartName, generateHelmRepoPath(path))
	if err != nil {
		return nil, err
	}
	chartRequested, err := chartutil.Load(downloadedChartPath)
	if err != nil {
		return nil, fmt.Errorf("Error loading chart: %v", err)
	}
	if req, err := chartutil.LoadRequirements(chartRequested); err == nil {
		if err := checkDependencies(chartRequested, req); err != nil {
			return nil, err
		}
	} else if err != chartutil.ErrRequirementsNotFound {
		return nil, fmt.Errorf("cannot load requirements: %v", err)
	}
	//Get cluster based or inCluster kubeconfig
	hClient, err := GetHelmClient(kubeConfig)
	if err != nil {
		return nil, err
	}
	upgradeRes, err := hClient.UpdateReleaseFromChart(
		deploymentName,
		chartRequested,
		helm.UpdateValueOverrides(values),
		helm.UpgradeDryRun(false),
		//helm.ResetValues(u.resetValues),
		helm.ReuseValues(reuseValues),
	)
	if err != nil {
		return nil, fmt.Errorf("upgrade failed: %v", err)
	}
	return upgradeRes, nil
}

//CreateDeployment creates a Helm deployment
func CreateDeployment(chartName string, releaseName string, valueOverrides []byte, kubeConfig []byte, path string) (*rls.InstallReleaseResponse, error) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagCreateDeployment})

	log.Infof("Deploying chart='%s', release name='%s'.", chartName, releaseName)
	downloadedChartPath, err := downloadChartFromRepo(chartName, generateHelmRepoPath(path))
	if err != nil {
		return nil, err
	}

	log.Infof("Loading chart '%s'", downloadedChartPath)
	chartRequested, err := chartutil.Load(downloadedChartPath)
	if err != nil {
		return nil, fmt.Errorf("Error loading chart: %v", err)
	}
	if req, err := chartutil.LoadRequirements(chartRequested); err == nil {
		if err := checkDependencies(chartRequested, req); err != nil {
			return nil, err
		}
	} else if err != chartutil.ErrRequirementsNotFound {
		return nil, fmt.Errorf("cannot load requirements: %v", err)
	}
	var namespace = "default"
	if len(strings.TrimSpace(releaseName)) == 0 {
		releaseName, _ = generateName("")
	}
	hClient, err := GetHelmClient(kubeConfig)
	if err != nil {
		return nil, err
	}
	installRes, err := hClient.InstallReleaseFromChart(
		chartRequested,
		namespace,
		helm.ValueOverrides(valueOverrides),
		helm.ReleaseName(releaseName),
		helm.InstallDryRun(false),
		helm.InstallReuseName(true),
		helm.InstallDisableHooks(false),
		helm.InstallTimeout(30),
		helm.InstallWait(false))
	if err != nil {
		return nil, fmt.Errorf("Error deploying chart: %v", err)
	}
	return installRes, nil
}

//DeleteDeployment deletes a Helm deployment
func DeleteDeployment(releaseName string, kubeConfig []byte) error {
	hClient, err := GetHelmClient(kubeConfig)
	if err != nil {
		return err
	}
	//TODO sophisticate commant options
	opts := []helm.DeleteOption{
		helm.DeletePurge(true),
	}
	_, err = hClient.DeleteRelease(releaseName, opts...)
	if err != nil {
		return err
	}
	return nil
}

//GetDeployment - N/A
func GetDeployment() {

}

// GetDeploymentStatus retrieves the status of the passed in release name.
// returns with an error if the release is not found or another error occurs
// in case of error the status is filled with information to classify the error cause
func GetDeploymentStatus(releaseName string, kubeConfig []byte) (int32, error) {

	helmClient, err := GetHelmClient(kubeConfig)

	if err != nil {
		// internal server error
		return http.StatusInternalServerError, errors.Wrap(err, "couldn't get the helm client")
	}

	releaseStatusResponse, err := helmClient.ReleaseStatus(releaseName)

	if err != nil {
		// the release cannot be found
		return http.StatusNotFound, errors.Wrap(err, "couldn't get the release status")
	}

	return int32(releaseStatusResponse.Info.Status.GetCode()), nil

}

func generateName(nameTemplate string) (string, error) {
	t, err := template.New("name-template").Funcs(sprig.TxtFuncMap()).Parse(nameTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	err = t.Execute(&b, nil)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func checkDependencies(ch *chart.Chart, reqs *chartutil.Requirements) error {
	missing := []string{}

	deps := ch.GetDependencies()
	for _, r := range reqs.Dependencies {
		found := false
		for _, d := range deps {
			if d.Metadata.Name == r.Name {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("found in requirements.yaml, but missing in charts/ directory: %s", strings.Join(missing, ", "))
	}
	return nil
}

func mergeValues(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = v
			continue
		}
		nextMap, ok := v.(map[string]interface{})
		// If it isn't another map, overwrite the value
		if !ok {
			dest[k] = v
			continue
		}
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = nextMap
			continue
		}
		// Edge case: If the key exists in the destination, but isn't a map
		destMap, isMap := dest[k].(map[string]interface{})
		// If the source map has a map for this key, prefer it
		if !isMap {
			dest[k] = v
			continue
		}
		// If we got to this point, it is a map in both, so merge them
		dest[k] = mergeValues(destMap, nextMap)
	}
	return dest
}

func ReposGet(clusterName string) ([]*repo.Entry, error) {
	repoPath := fmt.Sprintf("%s/repository/repositories.yaml", generateHelmRepoPath(clusterName))
	log.Debug("Helm repo path:", repoPath)

	f, err := repo.LoadRepositoriesFile(repoPath)
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return make([]*repo.Entry, 0), nil
	}

	return f.Repositories, nil
}

func ReposAdd(clusterName string, Hrepo *repo.Entry) error {

	settings := createEnvSettings(generateHelmRepoPath(clusterName))
	repoFile := settings.Home.RepositoryFile()
	var f *repo.RepoFile
	if _, err := os.Stat(repoFile); err != nil {
		log.Infof("Creating %s", repoFile)
		f = repo.NewRepoFile()
	} else {
		f, err = repo.LoadRepositoriesFile(repoFile)
		if err != nil {
			return errors.Wrap(err, "Cannot create a new ChartRepo")
		}
		log.Debugf("Profile file %q loaded.", repoFile)
	}

	for _, n := range f.Repositories {
		log.Debug("repo", n.Name)
		if n.Name == Hrepo.Name {
			return errors.New("Already added.")
		}
	}

	c := repo.Entry{
		Name:  Hrepo.Name,
		URL:   Hrepo.URL,
		Cache: settings.Home.CacheIndex(Hrepo.Name),
	}
	r, err := repo.NewChartRepository(&c, getter.All(settings))
	if err != nil {
		return errors.Wrap(err, "Cannot create a new ChartRepo")
	}
	log.Debug("New repo added:", Hrepo.Name)

	errIdx := r.DownloadIndexFile("")
	if errIdx != nil {
		return errors.Wrap(errIdx, "Repo index download failed")
	}
	f.Add(&c)
	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return errors.Wrap(errW, "Cannot write helm repo profile file")
	}
	return nil
}

func ReposDelete(clusterName, repoName string) error {
	repoPath := generateHelmRepoPath(clusterName)
	settings := createEnvSettings(repoPath)
	repoFile := settings.Home.RepositoryFile()
	log.Debug("Repo File:", repoFile)

	r, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	if !r.Remove(repoName) {
		return ErrRepoNotFound
	}
	if err := r.WriteFile(repoFile, 0644); err != nil {
		return err
	}

	if _, err := os.Stat(settings.Home.CacheIndex(repoName)); err == nil {
		err = os.Remove(settings.Home.CacheIndex(repoName))
		if err != nil {
			return err
		}
	}
	return nil

}

func ReposModify(clusterName, repoName string, newRepo *repo.Entry) error {
	log.Debug("ReposModify")
	repoPath := generateHelmRepoPath(clusterName)
	settings := createEnvSettings(repoPath)
	repoFile := settings.Home.RepositoryFile()
	log.Debug("Repo File:", repoFile)
	log.Debugf("New repo content: %#v", newRepo)

	f, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	if !f.Has(repoName) {
		return ErrRepoNotFound
	}

	f.Update(newRepo)

	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return errors.Wrap(errW, "Cannot write helm repo profile file")
	}
	return nil
}

func ReposUpdate(clusterName, repoName string) error {
	repoPath := generateHelmRepoPath(clusterName)
	settings := createEnvSettings(repoPath)
	repoFile := settings.Home.RepositoryFile()
	log.Debug("Repo File:", repoFile)

	f, err := repo.LoadRepositoriesFile(repoFile)

	if err != nil {
		return errors.Wrap(err, "Load ChartRepo")
	}

	for _, cfg := range f.Repositories {
		if cfg.Name == repoName {
			c, err := repo.NewChartRepository(cfg, getter.All(settings))
			if err != nil {
				return errors.Wrap(err, "Cannot get ChartRepo")
			}
			errIdx := c.DownloadIndexFile("")
			if errIdx != nil {
				return errors.Wrap(errIdx, "Repo index download failed")
			}
			return nil

		}
	}

	return ErrRepoNotFound
}

type ChartList struct {
	Name   string               `json:"name"`
	Charts []repo.ChartVersions `json:"charts"`
}

func ChartsGet(clusterName, queryName, queryRepo, queryVersion, queryKeyword string) ([]ChartList, error) {

	repoPath := fmt.Sprintf("%s/repository/repositories.yaml", generateHelmRepoPath(clusterName))
	log.Debug("Helm repo path:", repoPath)

	f, err := repo.LoadRepositoriesFile(repoPath)
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return nil, nil
	}
	cl := make([]ChartList, 0)

	for _, r := range f.Repositories {

		log.Debugf("Repository: %s", r.Name)
		i, errIndx := repo.LoadIndexFile(r.Cache)
		if errIndx != nil {
			return nil, errIndx
		}
		repoMatched, _ := regexp.MatchString(queryRepo, strings.ToLower(r.Name))
		if repoMatched || queryRepo == "" {
			log.Debugf("Repository: %s Matched", r.Name)
			c := ChartList{
				Name:   r.Name,
				Charts: make([]repo.ChartVersions, 0),
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
						c.Charts = append(c.Charts, repo.ChartVersions{i.Entries[n][0]})
					} else {
						c.Charts = append(c.Charts, i.Entries[n])
					}
				}

			}
			cl = append(cl, c)

		}
	}
	return cl, nil
}

type SpotguideOptions struct {
	Name    string `json:name`
	Type    string `json:type`
	Default bool   `json:"default"`
	Info    string `json:"info"`
	Key     string `json:"key"`
}

type ChartDetails struct {
	Name    string             `json:"name"`
	Repo    string             `json:"repo"`
	Chart   *repo.ChartVersion `json:"chart"`
	Values  string             `json:"values"`
	Readme  string             `json:"readme"`
	Options *SpotguideOptions  `json:"options"`
}

func ChartGet(clusterName, chartRepo, chartName, chartVersion string) (*ChartDetails, error) {

	repoPath := fmt.Sprintf("%s/repository/repositories.yaml", generateHelmRepoPath(clusterName))
	log.Debug("Helm repo path:", repoPath)
	chartD := &ChartDetails{}
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
							reader, err := downloadFile(chartSource)
							if err != nil {
								return nil, err
							}
							valuesStr, err := getChartFile(reader, "values.yaml")
							if err != nil {
								return nil, err
							}
							options, err := getChartOption(reader)
							if err != nil {
								return nil, err
							}
							log.Debugf("values hash: %s", valuesStr)

							readmeStr, err := getChartFile(reader, "README.md")
							if err != nil {
								return nil, err
							}
							log.Debugf("readme hash: %s", readmeStr)
							chartD = &ChartDetails{
								Name:    chartName,
								Repo:    chartRepo,
								Chart:   s,
								Values:  valuesStr,
								Readme:  readmeStr,
								Options: options,
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
