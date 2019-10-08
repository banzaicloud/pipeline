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

package helm

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"text/template"
	"time"

	"emperror.dev/emperror"
	"github.com/Masterminds/sprig"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

// DefaultNamespace default namespace
const DefaultNamespace = "default"

// SystemNamespace K8s system namespace
const SystemNamespace = "kube-system"

const versionAll = "all"

// ErrRepoNotFound describe an error if helm repository not found
// nolint: gochecknoglobals
var ErrRepoNotFound = errors.New("helm repository not found!")

// DefaultInstallOptions contains th default install options used for creating a new helm deployment
// nolint: gochecknoglobals
var DefaultInstallOptions = []helm.InstallOption{
	helm.InstallReuseName(true),
	helm.InstallDisableHooks(false),
	helm.InstallTimeout(300),
	helm.InstallWait(false),
	helm.InstallDryRun(false),
}

// DeploymentNotFoundError is returned when a Helm related operation is executed on
// a deployment (helm release) that doesn't exists
type DeploymentNotFoundError struct {
	HelmError error
}

func (e *DeploymentNotFoundError) Error() string {
	return fmt.Sprintf("deployment not found: %s", e.HelmError)
}

type chartDataIsTooBigError struct {
	size int64
}

func (e *chartDataIsTooBigError) Error() string {
	return "chart data is too big"
}

func (e *chartDataIsTooBigError) Context() []interface{} {
	return []interface{}{"maxAllowedSize", maxCompressedDataSize, "size", e.size}
}

const maxCompressedDataSize = 10485760
const maxDataSize = 10485760

// DownloadFile download file/unzip and untar and store it in memory
func DownloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	compressedContent := new(bytes.Buffer)

	if resp.ContentLength > maxCompressedDataSize {
		return nil, errors.WithStack(&chartDataIsTooBigError{resp.ContentLength})
	}

	_, copyErr := io.CopyN(compressedContent, resp.Body, maxCompressedDataSize)
	if copyErr != nil && copyErr != io.EOF {
		return nil, errors.Wrap(err, "failed to read from chart response")
	}

	gzf, err := gzip.NewReader(compressedContent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open chart gzip archive")
	}
	defer gzf.Close()

	tarContent := new(bytes.Buffer)
	_, copyErr = io.CopyN(tarContent, gzf, maxDataSize)
	if copyErr != nil && copyErr != io.EOF {
		return nil, errors.Wrap(copyErr, "failed to read from chart data archive")
	}

	return tarContent.Bytes(), nil
}

//DeleteAllDeployment deletes all Helm deployment
// namespaces - if provided than delete all helm deployments only from the provided namespaces
func DeleteAllDeployment(log logrus.FieldLogger, kubeconfig []byte, namespaces *v1.NamespaceList) error {
	log.Info("getting deployments....")
	filter := ""
	releaseResp, err := ListDeployments(&filter, "", kubeconfig)
	if err != nil {
		return emperror.Wrap(err, "failed to get deployments")
	}

	if releaseResp != nil {

		var nsFilter map[string]bool
		if namespaces != nil {
			nsFilter = make(map[string]bool, len(namespaces.Items))

			for _, ns := range namespaces.Items {
				nsFilter[ns.Name] = true
			}
		}

		// the returned release items are unique by release name and status
		// e.g. release name = release1, status = PENDING_UPGRADE
		//      release name = release1, status = DEPLOYED
		//
		// we need only the release name for deleting a release
		deletedDeployments := make(map[string]bool)
		for _, r := range releaseResp.GetReleases() {
			log := log.WithFields(logrus.Fields{"deployment": r.Name, "namespace": r.Namespace})

			_, ok := nsFilter[r.Namespace]
			if namespaces != nil && !ok {
				log.Info("skip deleting deployment")
				continue
			}

			if _, ok := deletedDeployments[r.Name]; !ok {
				log.Info("deleting deployment")

				err := DeleteDeployment(r.Name, kubeconfig)
				if err != nil {
					return emperror.WrapWith(err, "failed to delete deployment", "deployment", r.Name, "namespace", r.Name)
				}
				deletedDeployments[r.Name] = true

				log.Info("deployment successfully deleted")
			}
		}
	}
	return nil
}

// nolint: gochecknoglobals
var deploymentCache = cache.New(30*time.Minute, 5*time.Minute)

// ListDeployments lists Helm deployments
func ListDeployments(filter *string, tagFilter string, kubeConfig []byte) (*rls.ListReleasesResponse, error) {
	hClient, err := pkgHelm.NewClient(kubeConfig, log)
	if err != nil {
		return nil, err
	}
	defer hClient.Close()

	ops := []helm.ReleaseListOption{
		helm.ReleaseListSort(int32(rls.ListSort_LAST_RELEASED)),
		helm.ReleaseListOrder(int32(rls.ListSort_DESC)),
		helm.ReleaseListStatuses([]release.Status_Code{
			release.Status_DEPLOYED,
			release.Status_FAILED,
			release.Status_DELETING,
			release.Status_PENDING_INSTALL,
			release.Status_PENDING_UPGRADE,
			release.Status_PENDING_ROLLBACK}),
		// helm.ReleaseListLimit(limit),
		// helm.ReleaseListFilter(filter),
		// helm.ReleaseListNamespace(""),
	}
	if filter != nil {
		log.Debug("Apply filters: ", *filter)
		ops = append(ops, helm.ReleaseListFilter(*filter))
	}

	resp, err := hClient.ListReleases(ops...)
	if err != nil {
		return nil, err
	}

	if tagFilter != "" {

		clusterKey := string(sha1.New().Sum(kubeConfig))
		releasesKey := string(sha1.New().Sum([]byte(resp.String())))
		deploymentsKey := clusterKey + "-" + releasesKey

		type releaseWithDeployment struct {
			Deployment *pkgHelm.GetDeploymentResponse
			Release    *release.Release
		}
		var deployments []releaseWithDeployment

		deploymentsRaw, ok := deploymentCache.Get(deploymentsKey)
		if ok {
			deployments = deploymentsRaw.([]releaseWithDeployment)
		} else if resp != nil {
			for _, release := range resp.Releases {
				deployment, err := GetDeployment(release.Name, kubeConfig)
				if err != nil {
					return nil, err
				}
				deployments = append(deployments, releaseWithDeployment{Deployment: deployment, Release: release})
			}
			deploymentCache.Set(deploymentsKey, deployments, cache.DefaultExpiration)
		}

		filteredResp := &rls.ListReleasesResponse{}

		for _, deployment := range deployments {
			if DeploymentHasTag(deployment.Deployment, tagFilter) {
				filteredResp.Releases = append(filteredResp.Releases, deployment.Release)
				filteredResp.Count++
			}
		}

		resp = filteredResp
	}
	return resp, nil
}

func DeploymentHasTag(deployment *pkgHelm.GetDeploymentResponse, tagFilter string) bool {
	if banzaicloudRaw, ok := deployment.Values["banzaicloud"]; ok {
		banzaicloudValues, err := cast.ToStringMapE(banzaicloudRaw)
		if err != nil {
			return false
		}
		if tagsRaw, ok := banzaicloudValues["tags"]; ok {
			tags, err := cast.ToStringSliceE(tagsRaw)
			if err != nil {
				return false
			}
			for _, tag := range tags {
				if tag == tagFilter {
					return true
				}
			}
		}
	}
	return false
}

func GetRequestedChart(releaseName, chartName, chartVersion string, chartPackage []byte, env helm_env.EnvSettings) (requestedChart *chart.Chart, err error) {

	// If the request has a chart package sent by the user we install that
	if chartPackage != nil && len(chartPackage) != 0 {
		requestedChart, err = chartutil.LoadArchive(bytes.NewReader(chartPackage))
	} else {
		log.Infof("Deploying chart=%q, version=%q release name=%q", chartName, chartVersion, releaseName)
		var downloadedChartPath string
		downloadedChartPath, err = DownloadChartFromRepo(chartName, chartVersion, env)
		if err != nil {
			return nil, errors.Wrap(err, "error downloading chart")
		}

		requestedChart, err = chartutil.Load(downloadedChartPath)
	}

	if err != nil {
		return nil, errors.Wrap(err, "error loading chart")
	}

	if req, err := chartutil.LoadRequirements(requestedChart); err == nil {
		if err := checkDependencies(requestedChart, req); err != nil {
			return nil, errors.Wrap(err, "error checking chart dependencies")
		}
	} else if err != chartutil.ErrRequirementsNotFound {
		return nil, errors.Wrap(err, "cannot load requirements")
	}

	return requestedChart, err
}

// UpgradeDeployment upgrades a Helm deployment
func UpgradeDeployment(releaseName, chartName, chartVersion string, chartPackage []byte, values []byte, reuseValues bool, kubeConfig []byte, env helm_env.EnvSettings) (*rls.UpdateReleaseResponse, error) {

	chartRequested, err := GetRequestedChart(releaseName, chartName, chartVersion, chartPackage, env)
	if err != nil {
		return nil, fmt.Errorf("error loading chart: %v", err)
	}

	// Get cluster based on inCluster kubeconfig
	hClient, err := pkgHelm.NewClient(kubeConfig, log)
	if err != nil {
		return nil, err
	}
	defer hClient.Close()

	upgradeRes, err := hClient.UpdateReleaseFromChart(
		releaseName,
		chartRequested,
		helm.UpdateValueOverrides(values),
		helm.UpgradeDryRun(false),
		// helm.ResetValues(u.resetValues),
		helm.ReuseValues(reuseValues),
	)
	if err != nil {
		return nil, errors.Wrap(err, "upgrade failed")
	}

	return upgradeRes, nil
}

// CreateDeployment creates a Helm deployment in chosen namespace
func CreateDeployment(chartName, chartVersion string, chartPackage []byte, namespace string, releaseName string, dryRun bool, odPcts map[string]int, kubeConfig []byte, env helm_env.EnvSettings, overrideOpts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {

	chartRequested, err := GetRequestedChart(releaseName, chartName, chartVersion, chartPackage, env)
	if err != nil {
		return nil, fmt.Errorf("error loading chart: %v", err)
	}

	if len(strings.TrimSpace(releaseName)) == 0 {
		releaseName, _ = GenerateName("")
	}

	if namespace == "" {
		log.Warn("Deployment namespace was not set failing back to default")
		namespace = DefaultNamespace
	}

	var cmUpdated bool

	if !dryRun && odPcts != nil {
		if len(releaseName) == 0 {
			return nil, fmt.Errorf("release name cannot be empty when setting on-demand percentages")
		}
		err = updateSpotConfigMap(kubeConfig, odPcts, releaseName)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to update spot ConfigMap")
		}
		cmUpdated = true
	}

	hClient, err := pkgHelm.NewClient(kubeConfig, log)
	if err != nil {
		return nil, err
	}
	defer hClient.Close()

	basicOptions := []helm.InstallOption{
		helm.ReleaseName(releaseName),
		helm.InstallDryRun(dryRun),
	}
	installOptions := append(DefaultInstallOptions, basicOptions...)
	installOptions = append(installOptions, overrideOpts...)

	installRes, err := hClient.InstallReleaseFromChart(
		chartRequested,
		namespace,
		installOptions...,
	)
	if err != nil {
		if cmUpdated {
			err := cleanupSpotConfigMap(kubeConfig, odPcts, releaseName)
			if err != nil {
				log.Warn("failed to clean up spot config map")
			}
		}
		return nil, fmt.Errorf("Error deploying chart: %v", err)
	}
	return installRes, nil
}

func updateSpotConfigMap(kubeConfig []byte, odPcts map[string]int, releaseName string) error {
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to get kubernetes client from kubeconfig")
	}
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)
	cm, err := client.CoreV1().ConfigMaps(pipelineSystemNamespace).Get(common.SpotConfigMapKey, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			cm, err = client.CoreV1().ConfigMaps(pipelineSystemNamespace).Create(&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: common.SpotConfigMapKey,
				},
				Data: make(map[string]string),
			})
			if err != nil {
				return emperror.Wrap(err, "failed to create spot configmap")
			}
		} else {
			return emperror.Wrap(err, "failed to retrieve spot configmap")
		}
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	for res, pct := range odPcts {
		cm.Data[releaseName+"."+res] = fmt.Sprintf("%d", pct)
	}
	_, err = client.CoreV1().ConfigMaps(pipelineSystemNamespace).Update(cm)
	if err != nil {
		return emperror.Wrap(err, "failed to update spot configmap")
	}
	return nil
}

func cleanupSpotConfigMap(kubeConfig []byte, odPcts map[string]int, releaseName string) error {
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to get kubernetes client from kubeconfig")
	}
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)
	cm, err := client.CoreV1().ConfigMaps(pipelineSystemNamespace).Get(common.SpotConfigMapKey, metav1.GetOptions{})
	if err != nil {
		return emperror.Wrap(err, "failed to retrieve spot configmap")
	}

	if cm.Data == nil {
		return nil
	}
	for res := range odPcts {
		_, ok := cm.Data[releaseName+"."+res]
		if ok {
			delete(cm.Data, releaseName+"."+res)
		}
	}
	_, err = client.CoreV1().ConfigMaps(pipelineSystemNamespace).Update(cm)
	if err != nil {
		return emperror.Wrap(err, "failed to update spot configmap")
	}
	return nil
}

// DeleteDeployment deletes a Helm deployment
func DeleteDeployment(releaseName string, kubeConfig []byte) error {
	hClient, err := pkgHelm.NewClient(kubeConfig, log)
	if err != nil {
		return err
	}
	defer hClient.Close()
	// TODO sophisticate command options
	opts := []helm.DeleteOption{
		helm.DeletePurge(true),
	}
	_, err = hClient.DeleteRelease(releaseName, opts...)
	if err != nil {
		return err
	}
	return nil
}

// GetDeploymentK8sResources returns K8s resources of a helm deployment
func GetDeploymentK8sResources(releaseName string, kubeConfig []byte, resourceTypes []string) ([]pkgHelm.DeploymentResource, error) {
	hClient, err := pkgHelm.NewClient(kubeConfig, log)
	if err != nil {
		log.Errorf("Getting Helm client failed: %s", err.Error())
		return nil, err
	}
	defer hClient.Close()

	releaseContent, err := hClient.ReleaseContent(releaseName)

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, &DeploymentNotFoundError{HelmError: err}
		}
		return nil, err
	}

	return ParseReleaseManifest(releaseContent.Release.Manifest, resourceTypes)
}

func ParseReleaseManifest(manifest string, resourceTypes []string) ([]pkgHelm.DeploymentResource, error) {

	objects := strings.Split(manifest, "---")
	decode := scheme.Codecs.UniversalDeserializer().Decode
	deployments := make([]pkgHelm.DeploymentResource, 0)

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
			deployments = append(deployments, pkgHelm.DeploymentResource{
				Name: reflect.ValueOf(obj).Elem().FieldByName("Name").String(),
				Kind: reflect.ValueOf(obj).Elem().FieldByName("Kind").String(),
			})
		}

	}

	return deployments, nil
}

// GetDeployment returns the details of a helm deployment
func GetDeployment(releaseName string, kubeConfig []byte) (*pkgHelm.GetDeploymentResponse, error) {
	return GetDeploymentByVersion(releaseName, kubeConfig, 0)
}

// GetDeploymentByVersion returns the details of a helm deployment version
func GetDeploymentByVersion(releaseName string, kubeConfig []byte, version int32) (*pkgHelm.GetDeploymentResponse, error) {
	helmClient, err := pkgHelm.NewClient(kubeConfig, log)
	if err != nil {
		log.Errorf("Getting Helm client failed: %s", err.Error())
		return nil, err
	}
	defer helmClient.Close()

	releaseContent, err := helmClient.ReleaseContent(releaseName, helm.ContentReleaseVersion(version))

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, &DeploymentNotFoundError{HelmError: err}
		}
		return nil, err
	}

	createdAt := time.Unix(releaseContent.GetRelease().GetInfo().GetFirstDeployed().GetSeconds(), 0)
	updatedAt := time.Unix(releaseContent.GetRelease().GetInfo().GetLastDeployed().GetSeconds(), 0)
	chart := GetVersionedChartName(releaseContent.GetRelease().GetChart().GetMetadata().GetName(), releaseContent.GetRelease().GetChart().GetMetadata().GetVersion())

	notes := base64.StdEncoding.EncodeToString([]byte(releaseContent.GetRelease().GetInfo().GetStatus().GetNotes()))

	cfg, err := chartutil.CoalesceValues(releaseContent.GetRelease().GetChart(), releaseContent.GetRelease().GetConfig())
	if err != nil {
		log.Errorf("Retrieving deployment values failed: %s", err.Error())
		return nil, err
	}

	values := cfg.AsMap()

	return &pkgHelm.GetDeploymentResponse{
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

	helmClient, err := pkgHelm.NewClient(kubeConfig, log)

	if err != nil {
		// internal server error
		return http.StatusInternalServerError, errors.Wrap(err, "couldn't get the helm client")
	}
	defer helmClient.Close()

	releaseStatusResponse, err := helmClient.ReleaseStatus(releaseName)

	if err != nil {
		// the release cannot be found
		return http.StatusNotFound, errors.Wrap(err, "couldn't get the release status")
	}

	return int32(releaseStatusResponse.Info.Status.GetCode()), nil

}

func GenerateName(nameTemplate string) (string, error) {
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

func MergeValues(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
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
		dest[k] = MergeValues(destMap, nextMap)
	}
	return dest
}

// GetVersionedChartName returns chart name enriched with version number
func GetVersionedChartName(name, version string) string {
	return fmt.Sprintf("%s-%s", name, version)
}
