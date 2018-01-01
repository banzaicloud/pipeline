package helm

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/banzaicloud/pipeline/cloud"
	"github.com/ghodss/yaml"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

//ListDeployments lists Helm deployments
func ListDeployments(cluster *cluster.Cluster, filter *string) (*rls.ListReleasesResponse, error) {
	defer tearDown()
	kubeConfig, err := cloud.GetConfig(cluster, "")
	if err != nil {
		return nil, err
	}
	hClient, err := getHelmClient(kubeConfig)
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
func UpgradeDeployment(cluster *cluster.Cluster, deploymentName, chartName string, values map[string]interface{}) (string, error) {
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

	defer tearDown()
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
	kubeConfig := ""
	if cluster != nil {
		kubeConfig, err = cloud.GetConfig(cluster, "")
		if err != nil {
			return "", err
		}
	}
	hClient, err := getHelmClient(kubeConfig)
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
func CreateDeployment(cluster *cluster.Cluster, chartName string, releaseName string, valueOverrides []byte) (*rls.InstallReleaseResponse, error) {
	defer tearDown()
	chartRequested, err := chartutil.Load(chartName)
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

	kubeConfig, err := cloud.GetConfig(cluster, "")
	if err != nil {
		return nil, err
	}
	var namespace = "default"
	if len(strings.TrimSpace(releaseName)) == 0 {
		releaseName, _ = generateName("")
	}
	hClient, err := getHelmClient(kubeConfig)
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
func DeleteDeployment(cluster *cluster.Cluster, releaseName string) error {
	defer tearDown()
	kubeConfig, err := cloud.GetConfig(cluster, "")
	if err != nil {
		return err
	}
	hClient, err := getHelmClient(kubeConfig)
	if err != nil {
		return err
	}
	_, err = hClient.DeleteRelease(releaseName)
	if err != nil {
		return err
	}
	return nil
}

//GetDeployment - N/A
func GetDeployment() {

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
