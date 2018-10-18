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

package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/helm"
	pkgCommmon "github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/repo"
)

// ChartQuery describes a query to get available helm chart's list
type ChartQuery struct {
	Name    string `form:"name"`
	Repo    string `form:"repo"`
	Version string `form:"version"`
	Keyword string `form:"keyword"`
}

// GetK8sConfig returns the Kubernetes config
func GetK8sConfig(c *gin.Context) ([]byte, bool) {
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return nil, false
	}
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error getting config: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kubeconfig",
			Error:   err.Error(),
		})
		return nil, false
	}
	return kubeConfig, true
}

// CreateDeployment creates a Helm deployment
func CreateDeployment(c *gin.Context) {
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}
	parsedRequest, err := parseCreateUpdateDeploymentRequest(c, commonCluster)
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}
	release, err := helm.CreateDeployment(parsedRequest.deploymentName,
		parsedRequest.deploymentVersion,
		parsedRequest.deploymentPackage,
		parsedRequest.namespace,
		parsedRequest.deploymentReleaseName,
		parsedRequest.dryRun,
		parsedRequest.values,
		parsedRequest.odPcts,
		parsedRequest.kubeConfig,
		helm.GenerateHelmRepoEnv(parsedRequest.organizationName),
		helm.DefaultInstallOptions...,
	)
	if err != nil {
		//TODO distinguish error codes
		log.Errorf("Error during create deployment. %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error creating deployment",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Create deployment succeeded")

	releaseContent := release.GetRelease()

	releaseName := releaseContent.GetName()
	releaseNotes := base64.StdEncoding.EncodeToString([]byte(releaseContent.GetInfo().GetStatus().GetNotes()))
	resources, err := helm.ParseReleaseManifest(releaseContent.Manifest, []string{})
	if err != nil {
		log.Errorf("Error during parsing release manifest. %s", err.Error())
	}

	log.Debug("Release name: ", releaseName)
	log.Debug("Release notes: ", releaseNotes)
	log.Debug("Resources:", resources)
	response := pkgHelm.CreateUpdateDeploymentResponse{
		ReleaseName: releaseName,
		Notes:       releaseNotes,
		Resources:   resources,
	}
	c.JSON(http.StatusCreated, response)
	return
}

// ListDeployments lists a Helm deployment
func ListDeployments(c *gin.Context) {
	kubeConfig, ok := GetK8sConfig(c)
	if ok != true {
		return
	}

	log.Info("Get deployments")
	response, err := helm.ListDeployments(nil, c.Query("tag"), kubeConfig)
	if err != nil {
		log.Error("Error listing deployments: ", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing deployments",
			Error:   err.Error(),
		})
		return
	}

	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	chartsResponse, err := helm.ChartsGet(helmEnv, "", "", "", "")
	if err != nil {
		log.Error("Error listing charts for deployments: ", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing charts for deployments",
			Error:   err.Error(),
		})
		return
	}

	// Index known charts
	supportedCharts := map[string]repo.ChartVersions{}
	for _, charts := range chartsResponse {
		for _, chart := range charts.Charts {
			for _, chartVersion := range chart {
				supportedCharts[chartVersion.Name] = append(supportedCharts[chartVersion.Name], chartVersion)
			}
		}
	}
	releases := ListHelmReleases(c, response, supportedCharts)

	c.JSON(http.StatusOK, releases)
	return
}

// HelmDeploymentStatus checks the status of a deployment through the helm client API
func HelmDeploymentStatus(c *gin.Context) {

	name := c.Param("name")
	log.Infof("getting status for deployment: [%s]", name)

	kubeConfig, ok := GetK8sConfig(c)

	if !ok {
		log.Debug("could not get the k8s config")
		return
	}

	status, err := helm.GetDeploymentStatus(name, kubeConfig)
	// we have the status code in the status, regardless the error!

	var (
		statusCode int
		msg        string
	)

	if err != nil {
		// the helm client returned with error
		statusCode = int(status)
		msg = err.Error()
	} else {
		// the helm client returned with the status of the deployment
		if val, ok := release.Status_Code_name[status]; ok {
			log.Infof("deployment status: [%s]", val)
			statusCode = http.StatusOK
			msg = val
		}
	}

	log.Infof("deployment status for [%s] is [%d]", name, status)
	c.JSON(statusCode, pkgHelm.DeploymentStatusResponse{
		Status:  statusCode,
		Message: msg,
	})
}

// GetDeployment returns the details of a helm deployment
func GetDeployment(c *gin.Context) {
	name := c.Param("name")
	tag := c.Query("tag")
	log.Infof("getting details for deployment: [%s]", name)

	kubeConfig, ok := GetK8sConfig(c)

	if !ok {
		log.Errorf("could not get the k8s config for querying the details of deployment: [%s]", name)
		return
	}

	deployment, err := helm.GetDeployment(name, kubeConfig)
	if err == nil && tag != "" && !helm.DeploymentHasTag(deployment, tag) {
		notFoundError := &helm.DeploymentNotFoundError{HelmError: fmt.Errorf("tag not found")}
		err = notFoundError
		for version := deployment.Version - 1; version > 0; version-- {
			deployment, err = helm.GetDeploymentByVersion(name, kubeConfig, version)
			if err != nil || helm.DeploymentHasTag(deployment, tag) {
				break
			} else {
				err = notFoundError
			}
		}
	}

	if err == nil {
		c.JSON(http.StatusOK, deployment)
	} else {

		httpStatusCode := http.StatusInternalServerError
		if _, ok := err.(*helm.DeploymentNotFoundError); ok {
			httpStatusCode = http.StatusNotFound
		} else {
			log.Error("Error during getting deployment details: ", err.Error())
		}

		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error getting deployment",
			Error:   err.Error(),
		})
	}
}

// GetDeploymentResources returns the resources of a helm deployment
func GetDeploymentResources(c *gin.Context) {
	name := c.Param("name")
	log.Infof("getting resources for deployment: [%s]", name)

	resourceTypesStr := c.Query("resourceTypes")
	resourceTypes := make([]string, 0)
	if len(resourceTypesStr) != 0 {
		resourceTypes = append(resourceTypes, strings.Split(resourceTypesStr, ",")...)
	}

	kubeConfig, ok := GetK8sConfig(c)

	if !ok {
		log.Errorf("could not get the k8s config for querying the resources of deployment: [%s]", name)
		return
	}

	deploymentResourcesResponse, err := helm.GetDeploymentK8sResources(name, kubeConfig, resourceTypes)
	if err != nil {
		log.Error("Error during getting deployment resources: ", err.Error())

		httpStatusCode := http.StatusInternalServerError
		if _, ok := err.(*helm.DeploymentNotFoundError); ok {
			httpStatusCode = http.StatusNotFound
		}

		c.JSON(httpStatusCode, pkgCommmon.ErrorResponse{
			Code:    httpStatusCode,
			Message: "Error getting deployment resources",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, pkgHelm.GetDeploymentResourcesResponse{
		DeploymentResources: deploymentResourcesResponse,
	})

}

// InitHelmOnCluster installs Helm on AKS cluster and configure the Helm client
func InitHelmOnCluster(c *gin.Context) {
	log.Info("Start helm install")

	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}

	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error during getting kubeconfig: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kubeconfig",
			Error:   err.Error(),
		})
		return
	}
	// bind request body to struct
	var helmInstall pkgHelm.Install
	if err := c.BindJSON(&helmInstall); err != nil {
		// bind failed
		log.Errorf("Required field is empty: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	err = helm.Install(&helmInstall, kubeConfig)
	if err != nil {
		log.Errorf("Unable to install chart: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error installing helm",
			Error:   err.Error(),
		})
		return
	}
	message := "helm initialising"
	c.JSON(http.StatusCreated, pkgHelm.InstallResponse{
		Status:  http.StatusCreated,
		Message: message,
	})
	log.Info(message)
	return
}

// GetTillerStatus checks if tiller ready to accept deployments
func GetTillerStatus(c *gin.Context) {
	name := c.Param("name")
	log.Infof("Retrieving status for deployment: %s", name)
	kubeConfig, ok := GetK8sConfig(c)
	if ok != true {
		return
	}
	// --- [ List deployments ] ---- //
	_, err := helm.ListDeployments(nil, "", kubeConfig)
	if err != nil {
		message := "Error connecting to tiller"
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   err.Error(),
		})
		log.Errorln(message, err.Error())
		return
	}
	c.JSON(http.StatusOK, pkgHelm.StatusResponse{
		Status:  http.StatusOK,
		Message: "Tiller is available",
	})
	return
}

//UpgradeDeployment - Upgrades helm deployment, if --reuse-value is specified reuses the last release's value.
func UpgradeDeployment(c *gin.Context) {
	name := c.Param("name")
	log.Infof("Upgrading deployment: %s", name)
	commonCluster, ok := getClusterFromRequest(c)
	if ok != true {
		return
	}
	parsedRequest, err := parseCreateUpdateDeploymentRequest(c, commonCluster)
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	release, err := helm.UpgradeDeployment(name, parsedRequest.deploymentName,
		parsedRequest.deploymentVersion, parsedRequest.deploymentPackage, parsedRequest.values,
		parsedRequest.reuseValues, parsedRequest.kubeConfig, helm.GenerateHelmRepoEnv(parsedRequest.organizationName))
	if err != nil {
		log.Errorf("Error during upgrading deployment. %s", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error upgrading deployment",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Upgrade deployment succeeded")

	releaseNotes := base64.StdEncoding.EncodeToString([]byte(release.GetRelease().GetInfo().GetStatus().GetNotes()))

	log.Debug("Release notes: ", releaseNotes)
	response := pkgHelm.CreateUpdateDeploymentResponse{
		ReleaseName: name,
		Notes:       releaseNotes,
	}
	c.JSON(http.StatusCreated, response)
	return
}

//DeleteDeployment deletes a Helm deployment
func DeleteDeployment(c *gin.Context) {
	name := c.Param("name")
	log.Infof("Delete deployment: %s", name)
	kubeConfig, ok := GetK8sConfig(c)
	if ok != true {
		return
	}
	err := helm.DeleteDeployment(name, kubeConfig)
	if err != nil {
		// error during delete deployment
		log.Errorf("Error deleting deployment: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error deleting deployment",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, pkgHelm.DeleteResponse{
		Status:  http.StatusOK,
		Message: "Deployment deleted!",
		Name:    name,
	})
}

type parsedDeploymentRequest struct {
	deploymentName        string
	deploymentVersion     string
	deploymentPackage     []byte
	deploymentReleaseName string
	reuseValues           bool
	namespace             string
	values                []byte
	kubeConfig            []byte
	organizationName      string
	dryRun                bool
	odPcts                map[string]int
	// TODO: validate - 1. if odPcts are set, deploymentName must be set // if dryRun is set, odPcts shouldn't be set // map keys should match resource names in helm template
}

func parseCreateUpdateDeploymentRequest(c *gin.Context, commonCluster cluster.CommonCluster) (*parsedDeploymentRequest, error) {
	pdr := new(parsedDeploymentRequest)

	organization, err := auth.GetOrganizationById(commonCluster.GetOrganizationId())
	if err != nil {
		return nil, errors.Wrap(err, "Error during getting organization. ")
	}

	pdr.organizationName = organization.Name

	var deployment *pkgHelm.CreateUpdateDeploymentRequest
	err = c.BindJSON(&deployment)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing request:")
	}
	log.Info("Parse deployment succeeded")

	log.Debugf("Parsing chart %s with version %s and release name %s", deployment.Name, deployment.Version, deployment.ReleaseName)

	pdr.deploymentName = deployment.Name
	pdr.deploymentVersion = deployment.Version
	pdr.deploymentPackage = deployment.Package
	pdr.deploymentReleaseName = deployment.ReleaseName
	pdr.reuseValues = deployment.ReUseValues
	pdr.namespace = deployment.Namespace
	pdr.dryRun = deployment.DryRun
	pdr.odPcts = deployment.OdPcts

	if deployment.Values != nil {
		pdr.values, err = yaml.Marshal(deployment.Values)
		if err != nil {
			return nil, errors.Wrap(err, "Can't parse Values:")
		}
	}
	pdr.kubeConfig, err = commonCluster.GetK8sConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Error getting kubeconfig:")
	}
	log.Debug("Custom values: ", string(pdr.values))
	return pdr, nil
}

//HelmReposGet listing helm repositories in the cluster
func HelmReposGet(c *gin.Context) {

	log.Info("Get helm repository")

	response, err := helm.ReposGet(helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name))
	if err != nil {
		log.Errorf("Error during get helm repo list: %s", err.Error())
		c.JSON(http.StatusInternalServerError, pkgCommmon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error listing helm repos",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response)
	return
}

//HelmReposAdd add a new helm repository
func HelmReposAdd(c *gin.Context) {
	log.Info("Add helm repository")

	var r *repo.Entry
	err := c.BindJSON(&r)
	if err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	_, err = helm.ReposAdd(helmEnv, r)
	if err != nil {
		log.Errorf("Error adding helm repo: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error adding helm repo",
			Error:   err.Error(),
		})
		return
	}

	sendResponseWithRepo(c, helmEnv, r.Name)

	return
}

//HelmReposDelete delete the helm repository
func HelmReposDelete(c *gin.Context) {
	log.Info("Delete helm repository")

	repoName := c.Param("name")
	log.Debugf("repoName: %s", repoName)
	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	err := helm.ReposDelete(helmEnv, repoName)
	if err != nil {
		log.Error("Error during get helm repo delete.", err.Error())
		if err.Error() == helm.ErrRepoNotFound.Error() {
			c.JSON(http.StatusOK, pkgHelm.DeleteResponse{
				Status:  http.StatusOK,
				Message: err.Error(),
				Name:    repoName})
			return

		}
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error deleting helm repos",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, pkgHelm.DeleteResponse{
		Status:  http.StatusOK,
		Message: "resource deleted successfully.",
		Name:    repoName})
	return
}

//HelmReposModify modify the helm repository
func HelmReposModify(c *gin.Context) {
	log.Info("modify helm repository")

	repoName := c.Param("name")
	log.Debugf("repoName: %s", repoName)

	var newRepo *repo.Entry
	err := c.BindJSON(&newRepo)
	if err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error parsing request",
			Error:   err.Error(),
		})
		return
	}
	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	errModify := helm.ReposModify(helmEnv, repoName, newRepo)
	if errModify != nil {
		if errModify == helm.ErrRepoNotFound {
			c.JSON(http.StatusNotFound, pkgCommmon.ErrorResponse{
				Code:    http.StatusNotFound,
				Error:   errModify.Error(),
				Message: "repo not found",
			})
			return

		}
		log.Errorf("Error during helm repo modified. %s", errModify.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   errModify.Error(),
			Message: "repo modification failed",
		})
		return
	}

	sendResponseWithRepo(c, helmEnv, newRepo.Name)

	return
}

// HelmReposUpdate update the helm repo
func HelmReposUpdate(c *gin.Context) {
	log.Info("update helm repository")

	repoName := c.Param("name")
	log.Debugf("repoName: %s", repoName)
	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	errUpdate := helm.ReposUpdate(helmEnv, repoName)
	if errUpdate != nil {
		log.Errorf("Error during helm repo update. %s", errUpdate.Error())
		c.JSON(http.StatusNotFound, pkgCommmon.ErrorResponse{
			Code:    http.StatusNotFound,
			Error:   errUpdate.Error(),
			Message: "repository update failed",
		})
		return
	}

	sendResponseWithRepo(c, helmEnv, repoName)

	return
}

//HelmCharts get available helm chart's list
func HelmCharts(c *gin.Context) {
	log.Info("Get helm repository charts")

	var query ChartQuery
	err := c.BindQuery(&query)
	if err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error parsing request",
			Error:   err.Error(),
		})
		return
	}

	log.Info(query)
	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	response, err := helm.ChartsGet(helmEnv, query.Name, query.Repo, query.Version, query.Keyword)
	if err != nil {
		log.Error("Error during get helm repo chart list.", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing helm repo charts",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, response)
	return
}

//HelmChart get helm chart details
func HelmChart(c *gin.Context) {
	log.Info("Get helm chart")

	log.Debugf("%#v", c)
	chartRepo := c.Param("reponame")
	log.Debugln("chartRepo:", chartRepo)

	chartName := c.Param("name")
	log.Debugln("chartName:", chartName)

	chartVersion := c.DefaultQuery("version", "")
	log.Debugln("version:", chartVersion)

	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	response, err := helm.ChartGet(helmEnv, chartRepo, chartName, chartVersion)
	if err != nil {
		log.Error("Error during get helm chart information.", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during get helm chart information.",
			Error:   err.Error(),
		})
		return
	}
	if response == nil {
		c.JSON(http.StatusNotFound, pkgCommmon.ErrorResponse{
			Code:    http.StatusNotFound,
			Error:   "Chart Not Found!",
			Message: "Chart Not Found!",
		})
		return
	}

	c.JSON(http.StatusOK, response)
	return
}

func sendResponseWithRepo(c *gin.Context, helmEnv environment.EnvSettings, repoName string) {

	entries, err := helm.ReposGet(helmEnv)
	if err != nil {
		log.Errorf("Error during getting helm repo: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommmon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during getting helm repo",
			Error:   err.Error(),
		})
		return
	}

	for _, entry := range entries {
		if entry.Name == repoName {
			c.JSON(http.StatusOK, entry)
			return
		}
	}

	c.JSON(http.StatusNotFound, pkgCommmon.ErrorResponse{
		Code:    http.StatusNotFound,
		Message: "Helm repo not found",
	})
}

// ListHelmReleases list helm releases
func ListHelmReleases(c *gin.Context, response *rls.ListReleasesResponse, optparam interface{}) []pkgHelm.ListDeploymentResponse {

	// Get WhiteList set
	releaseWhitelist, ok := GetWhitelistSet(c)
	if !ok {
		log.Warnf("whitelist data is not valid: %#v", releaseWhitelist)
	}
	releaseScanLogReject, ok := GetReleaseScanLog(c)
	if !ok {
		log.Warnf("scanlog data is not valid: %#v", releaseScanLogReject)
	}

	releases := make([]pkgHelm.ListDeploymentResponse, 0)
	if response != nil && len(response.Releases) > 0 {
		for _, r := range response.Releases {

			createdAt := time.Unix(r.Info.FirstDeployed.Seconds, 0)
			updated := time.Unix(r.Info.LastDeployed.Seconds, 0)
			chartName := r.GetChart().GetMetadata().GetName()

			body := pkgHelm.ListDeploymentResponse{
				Name:         r.Name,
				Chart:        helm.GetVersionedChartName(r.Chart.Metadata.Name, r.Chart.Metadata.Version),
				ChartName:    chartName,
				ChartVersion: r.GetChart().GetMetadata().GetVersion(),
				Version:      r.Version,
				UpdatedAt:    updated,
				Status:       r.Info.Status.Code.String(),
				Namespace:    r.Namespace,
				CreatedAt:    createdAt,
			}
			optparamType := fmt.Sprintf("%T", optparam)
			if optparamType == "map[string]repo.ChartVersions" {
				supportedCharts := optparam.(map[string]repo.ChartVersions)
				body.Supported = supportedCharts[chartName] != nil
			}
			//Add WhiteListed flag if present
			if _, ok := releaseWhitelist[r.Name]; ok {
				body.WhiteListed = ok
			}
			if _, ok := releaseScanLogReject[r.Name]; ok {
				body.Rejected = ok
			}
			if optparamType == "map[string]bool" {
				releaseMap := optparam.(map[string]bool)
				if _, ok := releaseMap[r.Name]; ok {
					releases = append(releases, body)
				}
			} else {
				releases = append(releases, body)
			}
		}
	} else {
		log.Info("There are no installed charts.")
	}
	return releases
}
