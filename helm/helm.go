package helm

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/config"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"net/http"
)

var logger *logrus.Logger
var log *logrus.Entry

// Simple init for logging
func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"action": "Helm"})
}

//DeleteAllDeployment deletes all Helm deployment
func DeleteAllDeployment(kubeconfig *[]byte) error {
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
func ListDeployments(filter *string, kubeConfig *[]byte) (*rls.ListReleasesResponse, error) {
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
func UpgradeDeployment(deploymentName, chartName string, values map[string]interface{}, kubeConfig *[]byte) (string, error) {
	//Base maps for values
	base := map[string]interface{}{}
	//this is only to parse x=y format
	//if err := strvals.ParseInto(value, base); err != nil {
	//	return []byte{}, fmt.Errorf("failed parsing --set data: %s", err)
	//}
	base = mergeValues(base, values)
	updateValues, err := yaml.Marshal(base)
	if err != nil {
		return "", err
	}

	//Map chartName as

	chartRequested, err := chartutil.Load(chartName)
	if err != nil {
		return "", fmt.Errorf("Error loading chart: %v", err)
	}
	if req, err := chartutil.LoadRequirements(chartRequested); err == nil {
		if err := checkDependencies(chartRequested, req); err != nil {
			return "", err
		}
	} else if err != chartutil.ErrRequirementsNotFound {
		return "", fmt.Errorf("cannot load requirements: %v", err)
	}
	//Get cluster based or inCluster kubeconfig
	hClient, err := GetHelmClient(kubeConfig)
	if err != nil {
		return "", err
	}
	upgradeRes, err := hClient.UpdateReleaseFromChart(
		deploymentName,
		chartRequested,
		helm.UpdateValueOverrides(updateValues),
		helm.UpgradeDryRun(false),
		//helm.UpgradeRecreate(u.recreate),
		//helm.UpgradeForce(u.force),
		//helm.UpgradeDisableHooks(u.disableHooks),
		//helm.UpgradeTimeout(u.timeout),
		//helm.ResetValues(u.resetValues),
		//helm.ReuseValues(u.reuseValues),
		//helm.UpgradeWait(u.wait)
	)
	if err != nil {
		return "", fmt.Errorf("upgrade failed: %v", err)
	}
	return upgradeRes.Release.Name, nil
}

//CreateDeployment creates a Helm deployment
func CreateDeployment(chartName string, releaseName string, valueOverrides []byte, kubeConfig *[]byte, path string) (*rls.InstallReleaseResponse, error) {
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
func DeleteDeployment(releaseName string, kubeConfig *[]byte) error {
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
func GetDeploymentStatus(releaseName string, kubeConfig *[]byte) (int32, error) {

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
