package helm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	helm2 "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/pkg/errors"
	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/repo"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

// DefaultNamespace default namespace
const DefaultNamespace = "default"

// SystemNamespace K8s system namespace
const SystemNamespace = "kube-system"

const versionAll = "all"

// ErrRepoNotFound describe an error if helm repository not found
var ErrRepoNotFound = errors.New("helm repository not found!")

// DeploymentNotFoundError is returned when a Helm related operation is executed on
// a deployment (helm release) that doesn't exists
type DeploymentNotFoundError struct {
	HelmError error
}

func (e *DeploymentNotFoundError) Error() string {
	return fmt.Sprintf("deployment not found: %s", e.HelmError)
}

// DownloadFile download file/unzip and untar and store it in memory
func DownloadFile(url string) ([]byte, error) {
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
	rawContent, _ := ioutil.ReadAll(gzf)
	return rawContent, nil
}

//GetChartFile Download file from chart repository
func GetChartFile(file []byte, fileName string) (string, error) {
	tarReader := tar.NewReader(bytes.NewReader(file))
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		// We search for explicit path and the root directory is unknown.
		// Apply regexp (<anything>/filename prevent false match like /root_dir/chart/abc/README.md
		match, _ := regexp.MatchString("^([^/]*)/"+fileName+"$", header.Name)
		if match {
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
	log.Info("Getting deployments....")
	filter := ""
	releaseResp, err := ListDeployments(&filter, kubeconfig)
	if err != nil {
		return err
	}
	log.Info("Starting deleting deployments")
	if releaseResp != nil {
		for _, r := range releaseResp.Releases {
			log.Info("Trying to delete deployment ", r.Name)
			err := DeleteDeployment(r.Name, kubeconfig)
			if err != nil {
				return err
			}
			log.Infof("Deployment %s successfully deleted", r.Name)
		}
	}
	return nil
}

//ListDeployments lists Helm deployments
func ListDeployments(filter *string, kubeConfig []byte) (*rls.ListReleasesResponse, error) {
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
		log.Debug("Apply filters: ", *filter)
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
func UpgradeDeployment(releaseName, chartName, chartVersion string, values []byte, reuseValues bool, kubeConfig []byte, env helm_env.EnvSettings) (*rls.UpdateReleaseResponse, error) {
	//Map chartName as
	log.Infof("Deploying chart=%q, version=%q release name=%q", chartName, chartVersion, releaseName)
	downloadedChartPath, err := DownloadChartFromRepo(chartName, chartVersion, env)
	if err != nil {
		return nil, err
	}

	chartRequested, err := chartutil.Load(downloadedChartPath)
	if err != nil {
		return nil, fmt.Errorf("error loading chart: %v", err)
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
		releaseName,
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

//CreateDeployment creates a Helm deployment in chosen namespace
func CreateDeployment(chartName string, chartVersion, namespace string, releaseName string, valueOverrides []byte, kubeConfig []byte, env helm_env.EnvSettings) (*rls.InstallReleaseResponse, error) {

	log.Infof("Deploying chart=%q, version=%q release name=%q", chartName, chartVersion, releaseName)
	downloadedChartPath, err := DownloadChartFromRepo(chartName, chartVersion, env)
	if err != nil {
		return nil, err
	}

	log.Infof("Loading chart from %q", env.Home)

	chartRequested, err := chartutil.Load(downloadedChartPath)
	if err != nil {
		return nil, fmt.Errorf("error loading chart: %v", err)
	}
	if req, err := chartutil.LoadRequirements(chartRequested); err == nil {
		if err := checkDependencies(chartRequested, req); err != nil {
			return nil, err
		}
	} else if err != chartutil.ErrRequirementsNotFound {
		return nil, fmt.Errorf("cannot load requirements: %v", err)
	}
	if len(strings.TrimSpace(releaseName)) == 0 {
		releaseName, _ = generateName("")
	}
	if namespace == "" {
		log.Warn("Deployment namespace was not set failing back to default")
		namespace = DefaultNamespace
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
	//TODO sophisticate command options
	opts := []helm.DeleteOption{
		helm.DeletePurge(true),
	}
	_, err = hClient.DeleteRelease(releaseName, opts...)
	if err != nil {
		return err
	}
	return nil
}

// GetDeploymentsK8sResources returns K8s resources of a helm deployment
func GetDeploymentK8sResources(releaseName string, kubeConfig []byte, resourceTypes []string) ([]helm2.DeploymentResource, error) {
	helmClient, err := GetHelmClient(kubeConfig)
	if err != nil {
		log.Errorf("Getting Helm client failed: %s", err.Error())
		return nil, err
	}

	releaseContent, err := helmClient.ReleaseContent(releaseName)

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, &DeploymentNotFoundError{HelmError: err}
		}
		return nil, err
	}

	objects := strings.Split(releaseContent.Release.Manifest, "---")
	decode := scheme.Codecs.UniversalDeserializer().Decode
	deployments := make([]helm2.DeploymentResource, 0)

	for _, object := range objects {

		obj, _, err := decode([]byte(object), nil, nil)

		if err != nil {
			log.Warnf("Error while decoding YAML object. Err was: %s", err)
			continue
		}
		log.Infof("version: %v/%v kind: %v", obj.GetObjectKind().GroupVersionKind().Group, obj.GetObjectKind().GroupVersionKind().Version, obj.GetObjectKind().GroupVersionKind().Kind)

		selectResource := false
		if len(resourceTypes) == 0 {
			selectResource = true
		} else {
			for _, resourceType := range resourceTypes {
				if strings.ToLower(resourceType) == strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind) {
					selectResource = true
				}
			}
		}

		if selectResource {
			//TODO add all K8s resources
			switch o := obj.(type) {
			case *extensions.Deployment:
				deployments = append(deployments, helm2.DeploymentResource{
					Name: o.Name,
					Kind: o.Kind,
				})
			case *v1beta1.Deployment:
				deployments = append(deployments, helm2.DeploymentResource{
					Name: o.Name,
					Kind: o.Kind,
				})
			case *v1beta2.StatefulSet:
				deployments = append(deployments, helm2.DeploymentResource{
					Name: o.Name,
					Kind: o.Kind,
				})
			default:
				//o is unknown for us
			}
		}

	}

	return deployments, nil
}

// GetDeployment returns the details of a helm deployment
func GetDeployment(releaseName string, kubeConfig []byte) (*helm2.GetDeploymentResponse, error) {
	helmClient, err := GetHelmClient(kubeConfig)
	if err != nil {
		log.Errorf("Getting Helm client failed: %s", err.Error())
		return nil, err
	}

	releaseContent, err := helmClient.ReleaseContent(releaseName)

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, &DeploymentNotFoundError{HelmError: err}
		}
		return nil, err
	}

	createdAt := utils.ConvertSecondsToTime(time.Unix(releaseContent.GetRelease().GetInfo().GetFirstDeployed().GetSeconds(), 0))
	updatedAt := utils.ConvertSecondsToTime(time.Unix(releaseContent.GetRelease().GetInfo().GetLastDeployed().GetSeconds(), 0))
	chart := GetVersionedChartName(releaseContent.GetRelease().GetChart().GetMetadata().GetName(), releaseContent.GetRelease().GetChart().GetMetadata().GetVersion())

	notes := base64.StdEncoding.EncodeToString([]byte(releaseContent.GetRelease().GetInfo().GetStatus().GetNotes()))

	cfg, err := chartutil.CoalesceValues(releaseContent.GetRelease().GetChart(), releaseContent.GetRelease().GetConfig())
	if err != nil {
		log.Errorf("Retrieving deployment values failed: %s", err.Error())
		return nil, err
	}

	values := cfg.AsMap()

	return &helm2.GetDeploymentResponse{
		ReleaseName:  releaseContent.GetRelease().GetName(),
		Namespace:    releaseContent.GetRelease().GetNamespace(),
		Version:      releaseContent.GetRelease().GetVersion(),
		Description:  releaseContent.GetRelease().GetInfo().GetDescription(),
		Status:       releaseContent.GetRelease().GetInfo().GetStatus().GetCode().String(),
		Notes:        notes,
		CreatedAt:    createdAt,
		Updated:      updatedAt,
		Chart:        chart,
		ChartName:    releaseContent.GetRelease().GetChart().GetMetadata().GetName(),
		ChartVersion: releaseContent.GetRelease().GetChart().GetMetadata().GetVersion(),
		Values:       values,
	}, nil
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

// ReposGet returns repo
func ReposGet(env helm_env.EnvSettings) ([]*repo.Entry, error) {

	repoPath := env.Home.RepositoryFile()
	log.Debugf("Helm repo path: %s", repoPath)

	f, err := repo.LoadRepositoriesFile(repoPath)
	if err != nil {
		return nil, err
	}
	if len(f.Repositories) == 0 {
		return make([]*repo.Entry, 0), nil
	}

	return f.Repositories, nil
}

// ReposAdd adds repo(s)
func ReposAdd(env helm_env.EnvSettings, Hrepo *repo.Entry) (bool, error) {
	repoFile := env.Home.RepositoryFile()
	var f *repo.RepoFile
	if _, err := os.Stat(repoFile); err != nil {
		log.Infof("Creating %s", repoFile)
		f = repo.NewRepoFile()
	} else {
		f, err = repo.LoadRepositoriesFile(repoFile)
		if err != nil {
			return false, errors.Wrap(err, "Cannot create a new ChartRepo")
		}
		log.Debugf("Profile file %q loaded.", repoFile)
	}

	for _, n := range f.Repositories {
		log.Debugf("repo: %s", n.Name)
		if n.Name == Hrepo.Name {
			return false, nil
		}
	}

	c := repo.Entry{
		Name:  Hrepo.Name,
		URL:   Hrepo.URL,
		Cache: env.Home.CacheIndex(Hrepo.Name),
	}
	r, err := repo.NewChartRepository(&c, getter.All(env))
	if err != nil {
		return false, errors.Wrap(err, "Cannot create a new ChartRepo")
	}
	log.Debugf("New repo added: %s", Hrepo.Name)

	errIdx := r.DownloadIndexFile("")
	if errIdx != nil {
		return false, errors.Wrap(errIdx, "Repo index download failed")
	}
	f.Add(&c)
	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return false, errors.Wrap(errW, "Cannot write helm repo profile file")
	}
	return true, nil
}

// ReposDelete deletes repo(s)
func ReposDelete(env helm_env.EnvSettings, repoName string) error {
	repoFile := env.Home.RepositoryFile()
	log.Debugf("Repo File: %s", repoFile)

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

	if _, err := os.Stat(env.Home.CacheIndex(repoName)); err == nil {
		err = os.Remove(env.Home.CacheIndex(repoName))
		if err != nil {
			return err
		}
	}
	return nil

}

// ReposModify modifies repo(s)
func ReposModify(env helm_env.EnvSettings, repoName string, newRepo *repo.Entry) error {

	log.Debug("ReposModify")
	repoFile := env.Home.RepositoryFile()
	log.Debugf("Repo File: %s", repoFile)
	log.Debugf("New repo content: %#v", newRepo)

	f, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	if !f.Has(repoName) {
		return ErrRepoNotFound
	}

	var formerRepo *repo.Entry
	repos := f.Repositories
	for _, r := range repos {
		if r.Name == repoName {
			formerRepo = r
		}
	}

	if formerRepo != nil {
		if len(newRepo.Name) == 0 {
			newRepo.Name = formerRepo.Name
			log.Infof("new repo name field is empty, replaced with: %s", formerRepo.Name)
		}

		if len(newRepo.URL) == 0 {
			newRepo.URL = formerRepo.URL
			log.Infof("new repo url field is empty, replaced with: %s", formerRepo.URL)
		}

		if len(newRepo.Cache) == 0 {
			newRepo.Cache = formerRepo.Cache
			log.Infof("new repo cache field is empty, replaced with: %s", formerRepo.Cache)
		}
	}

	f.Update(newRepo)

	if errW := f.WriteFile(repoFile, 0644); errW != nil {
		return errors.Wrap(errW, "Cannot write helm repo profile file")
	}
	return nil
}

// ReposUpdate updates a repo(s)
func ReposUpdate(env helm_env.EnvSettings, repoName string) error {

	repoFile := env.Home.RepositoryFile()
	log.Debugf("Repo File: %s", repoFile)

	f, err := repo.LoadRepositoriesFile(repoFile)

	if err != nil {
		return errors.Wrap(err, "Load ChartRepo")
	}

	for _, cfg := range f.Repositories {
		if cfg.Name == repoName {
			c, err := repo.NewChartRepository(cfg, getter.All(env))
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

// ChartList describe a chart list
type ChartList struct {
	Name   string               `json:"name"`
	Charts []repo.ChartVersions `json:"charts"`
}

// ChartsGet returns chart list
func ChartsGet(env helm_env.EnvSettings, queryName, queryRepo, queryVersion, queryKeyword string) ([]ChartList, error) {
	log.Debugf("Helm repo path %s", env.Home.RepositoryFile())
	f, err := repo.LoadRepositoriesFile(env.Home.RepositoryFile())
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
				chartMatched, _ := regexp.MatchString("^"+queryName+"$", strings.ToLower(n))

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

// ChartDetails describes a chart details
type ChartDetails struct {
	Name     string          `json:"name"`
	Repo     string          `json:"repo"`
	Versions []*ChartVersion `json:"versions"`
}

// ChartVersion describes a chart verion
type ChartVersion struct {
	Chart  *repo.ChartVersion `json:"chart"`
	Values string             `json:"values"`
	Readme string             `json:"readme"`
}

// ChartGet returns chart details
func ChartGet(env helm_env.EnvSettings, chartRepo, chartName, chartVersion string) (details *ChartDetails, err error) {

	repoPath := env.Home.RepositoryFile()
	log.Debugf("Helm repo path: %s", repoPath)
	var f *repo.RepoFile
	f, err = repo.LoadRepositoriesFile(repoPath)
	if err != nil {
		return
	}

	if len(f.Repositories) == 0 {
		return
	}

	for _, repository := range f.Repositories {

		log.Debugf("Repository: %s", repository.Name)

		var i *repo.IndexFile
		i, err = repo.LoadIndexFile(repository.Cache)
		if err != nil {
			return
		}

		details = &ChartDetails{
			Name: chartName,
			Repo: chartRepo,
		}

		if repository.Name == chartRepo {

			for name, chartVersions := range i.Entries {
				log.Debugf("Chart: %s", name)
				if chartName == name {
					for _, v := range chartVersions {

						if v.Version == chartVersion || chartVersion == "" {

							var ver *ChartVersion
							ver, err = getChartVersion(v)
							if err != nil {
								return
							}
							details.Versions = []*ChartVersion{ver}

							return
						} else if chartVersion == versionAll {
							var ver *ChartVersion
							ver, err = getChartVersion(v)
							if err != nil {
								log.Warnf("error during getting chart[%s - %s]: %s", v.Name, v.Version, err.Error())
							} else {
								details.Versions = append(details.Versions, ver)
							}

						}

					}
					return
				}

			}

		}
	}
	return
}

func getChartVersion(v *repo.ChartVersion) (*ChartVersion, error) {
	log.Infof("get chart[%s - %s]", v.Name, v.Version)

	chartSource := v.URLs[0]
	log.Debugf("chartSource: %s", chartSource)
	reader, err := DownloadFile(chartSource)
	if err != nil {
		return nil, err
	}
	valuesStr, err := GetChartFile(reader, "values.yaml")
	if err != nil {
		return nil, err
	}
	log.Debugf("values hash: %s", valuesStr)

	readmeStr, err := GetChartFile(reader, "README.md")
	if err != nil {
		return nil, err
	}
	log.Debugf("readme hash: %s", readmeStr)

	return &ChartVersion{
		Chart:  v,
		Values: valuesStr,
		Readme: readmeStr,
	}, nil
}

// GetVersionedChartName returns chart name enriched with version number
func GetVersionedChartName(name, version string) string {
	return fmt.Sprintf("%s-%s", name, version)
}
