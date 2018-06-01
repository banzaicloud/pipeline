package api

import (
	"fmt"
	"github.com/banzaicloud/banzai-types/components"
	htype "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/repo"
	"k8s.io/helm/pkg/timeconv"
	"net/http"
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
	log := logger.WithFields(logrus.Fields{"tag": "GetKubernetesConfig"})
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return nil, false
	}
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error getting config: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
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
	log := logger.WithFields(logrus.Fields{"tag": constants.TagCreateDeployment})
	parsedRequest, err := parseCreateUpdateDeploymentRequest(c)
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}
	release, err := helm.CreateDeployment(parsedRequest.deploymentName,
		parsedRequest.deploymentReleaseName,
		parsedRequest.values,
		parsedRequest.kubeConfig,
		helm.GenerateHelmRepoEnv(parsedRequest.organizationName))
	if err != nil {
		//TODO distinguish error codes
		log.Errorf("Error during create deployment. %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error creating deployment",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Create deployment succeeded")

	releaseName := release.Release.Name
	releaseNotes := release.Release.Info.Status.Notes

	log.Debug("Release name: ", releaseName)
	log.Debug("Release notes: ", releaseNotes)
	response := htype.CreateUpdateDeploymentResponse{
		ReleaseName: releaseName,
		Notes:       releaseNotes,
	}
	c.JSON(http.StatusCreated, response)
	return
}

// ListDeployments lists a Helm deployment
func ListDeployments(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagListDeployments})
	kubeConfig, ok := GetK8sConfig(c)
	if ok != true {
		return
	}

	log.Info("Get deployments")
	response, err := helm.ListDeployments(nil, kubeConfig)
	if err != nil {
		log.Error("Error during create deployment.", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing deployments",
			Error:   err.Error(),
		})
		return
	}
	var releases []htype.ListDeploymentResponse
	if response != nil && len(response.Releases) > 0 {
		for _, r := range response.Releases {
			body := htype.ListDeploymentResponse{
				Name:    r.Name,
				Chart:   fmt.Sprintf("%s-%s", r.Chart.Metadata.Name, r.Chart.Metadata.Version),
				Version: r.Version,
				Updated: timeconv.String(r.Info.LastDeployed),
				Status:  r.Info.Status.Code.String()}
			releases = append(releases, body)
		}
	} else {
		log.Info("There is no installed charts.")
	}
	c.JSON(http.StatusOK, releases)
	return
}

// HelmDeploymentStatus checks the status of a deployment through the helm client API
func HelmDeploymentStatus(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": "DeploymentStatus"})
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
	c.JSON(statusCode, htype.DeploymentStatusResponse{
		Status:  statusCode,
		Message: msg,
	})
}

// InitHelmOnCluster installs Helm on AKS cluster and configure the Helm client
func InitHelmOnCluster(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagHelmInstall})
	log.Info("Start helm install")

	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}

	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error during getting kubeconfig: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kubeconfig",
			Error:   err.Error(),
		})
		return
	}
	// bind request body to struct
	var helmInstall htype.Install
	if err := c.BindJSON(&helmInstall); err != nil {
		// bind failed
		log.Errorf("Required field is empty: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	err = helm.Install(&helmInstall, kubeConfig)
	if err != nil {
		log.Errorf("Unable to install chart: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error installing helm",
			Error:   err.Error(),
		})
		return
	}
	message := "helm initialising"
	c.JSON(http.StatusCreated, htype.InstallResponse{
		Status:  http.StatusCreated,
		Message: message,
	})
	log.Info(message)
	return
}

// GetTillerStatus checks if tiller ready to accept deployments
func GetTillerStatus(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "GetTillerStatus"})
	name := c.Param("name")
	log.Infof("Retrieving status for deployment: %s", name)
	kubeConfig, ok := GetK8sConfig(c)
	if ok != true {
		return
	}
	// --- [ List deployments ] ---- //
	_, err := helm.ListDeployments(nil, kubeConfig)
	if err != nil {
		message := "Error connecting to tiller"
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   err.Error(),
		})
		log.Error(message)
		return
	}
	c.JSON(http.StatusOK, htype.StatusResponse{
		Status:  http.StatusOK,
		Message: "Tiller is available",
	})
	return
}

//UpgradeDeployment - Upgrades helm deployment, if --reuse-value is specified reuses the last release's value.
func UpgradeDeployment(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "UpgradeDeployment"})
	name := c.Param("name")
	log.Infof("Upgrading deployment: %s", name)
	parsedRequest, err := parseCreateUpdateDeploymentRequest(c)
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	release, err := helm.UpgradeDeployment(name,
		parsedRequest.deploymentName, parsedRequest.values,
		parsedRequest.reuseValues, parsedRequest.kubeConfig, helm.GenerateHelmRepoEnv(parsedRequest.organizationName))
	if err != nil {
		log.Errorf("Error during upgrading deployment. %s", err.Error())
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error upgrading deployment",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Upgrade deployment succeeded")

	releaseNotes := release.Release.Info.Status.Notes

	log.Debug("Release notes: ", releaseNotes)
	response := htype.CreateUpdateDeploymentResponse{
		ReleaseName: name,
		Notes:       releaseNotes,
	}
	c.JSON(http.StatusCreated, response)
	return
}

//DeleteDeployment deletes a Helm deployment
func DeleteDeployment(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteDeployment"})
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
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error deleting deployment",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, htype.DeleteResponse{
		Status:  http.StatusOK,
		Message: "Deployment deleted!",
		Name:    name,
	})
}

type parsedDeploymentRequest struct {
	deploymentName        string
	deploymentReleaseName string
	reuseValues           bool
	values                []byte
	kubeConfig            []byte
	organizationName      string
}

func parseCreateUpdateDeploymentRequest(c *gin.Context) (*parsedDeploymentRequest, error) {
	log := logger.WithFields(logrus.Fields{"tag": "parseCreateUpdateDeploymentRequest"})
	pdr := new(parsedDeploymentRequest)

	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return nil, errors.New("Get cluster failed!")
	}

	organization, err := auth.GetOrganizationById(commonCluster.GetOrganizationId())
	if err != nil {
		return nil, errors.Wrap(err, "Error during getting organization. ")
	}

	pdr.organizationName = organization.Name

	var deployment *htype.CreateUpdateDeploymentRequest
	err = c.BindJSON(&deployment)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing request:")
	}
	log.Info("Parse deployment succeeded")

	log.Debugf("Parsing chart %s with version %s and release name %s", deployment.Name, deployment.Version, deployment.ReleaseName)

	pdr.deploymentName = deployment.Name
	pdr.deploymentReleaseName = deployment.ReleaseName
	pdr.reuseValues = deployment.ReUseValues

	if deployment.Values != "" {
		parsedJSON, err := yaml.Marshal(deployment.Values)
		if err != nil {
			return nil, errors.Wrap(err, "Can't parse Values:")
		}
		pdr.values, err = yaml.JSONToYAML(parsedJSON)
		if err != nil {
			return nil, errors.Wrap(err, "Can't convert json to yaml:")
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
	log := logger.WithFields(logrus.Fields{"tag": "HelmReposGet"})

	log.Info("Get helm repository")

	response, err := helm.ReposGet(helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name))
	if err != nil {
		log.Error("Error during get helm repo list.", err.Error())
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
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
	log := logger.WithFields(logrus.Fields{"tag": "HelmReposAdd"})
	log.Info("Add helm repository")

	var repo *repo.Entry
	err := c.BindJSON(&repo)
	if err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}

	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	_, err = helm.ReposAdd(helmEnv, repo)
	if err != nil {
		log.Errorf("Error adding helm repo: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error adding helm repo",
			Error:   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, htype.StatusResponse{
		Status:  http.StatusOK,
		Message: "resource successfully added.",
		Name:    repo.Name})
	return
}

//HelmReposDelete delete the helm repository
func HelmReposDelete(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "HelmReposDelete"})
	log.Info("Delete helm repository")

	repoName := c.Param("name")
	log.Debugf("repoName: %s", repoName)
	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	err := helm.ReposDelete(helmEnv, repoName)
	if err != nil {
		log.Error("Error during get helm repo delete.", err.Error())
		if err.Error() == helm.ErrRepoNotFound.Error() {
			c.JSON(http.StatusOK, htype.DeleteResponse{
				Status:  http.StatusOK,
				Message: err.Error(),
				Name:    repoName})
			return

		}
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "error deleting helm repos",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, htype.DeleteResponse{
		Status:  http.StatusOK,
		Message: "resource deleted successfully.",
		Name:    repoName})
	return
}

//HelmReposModify modify the helm repository
func HelmReposModify(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "HelmReposModify"})
	log.Info("modify helm repository")

	repoName := c.Param("name")
	log.Debugf("repoName: %s", repoName)

	var newRepo *repo.Entry
	err := c.BindJSON(&newRepo)
	if err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
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
			c.JSON(http.StatusNotFound, components.ErrorResponse{
				Code:    http.StatusNotFound,
				Error:   errModify.Error(),
				Message: "repo not found",
			})
			return

		}
		log.Error("Error during helm repo modified.", errModify.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Error:   errModify.Error(),
			Message: "repo modification failed",
		})
		return
	}

	c.JSON(http.StatusOK, htype.StatusResponse{
		Status:  http.StatusOK,
		Message: "resource modified successfully",
		Name:    repoName})
	return
}

// HelmReposUpdate update the helm repo
func HelmReposUpdate(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "ReposUpdate"})
	log.Info("delete helm repository")

	repoName := c.Param("name")
	log.Debugln("repoName:", repoName)
	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	errUpdate := helm.ReposUpdate(helmEnv, repoName)
	if errUpdate != nil {
		log.Error("Error during helm repo update.", errUpdate.Error())
		c.JSON(http.StatusNotFound, components.ErrorResponse{
			Code:    http.StatusNotFound,
			Error:   errUpdate.Error(),
			Message: "repository update failed",
		})
		return
	}

	c.JSON(http.StatusOK, htype.StatusResponse{
		Status:  http.StatusOK,
		Message: "repository updated successfully",
		Name:    repoName})
	return
}

//HelmCharts get available helm chart's list
func HelmCharts(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": "HelmCharts"})
	log.Info("Get helm repository charts")

	var query ChartQuery
	err := c.BindQuery(&query)
	if err != nil {
		log.Errorf("Error parsing request: %s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
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
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
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
	log := logger.WithFields(logrus.Fields{"tag": "HelmChart"})
	log.Info("Get helm chart")

	log.Debugf("%#v", c)
	chartRepo := c.Param("reponame")
	log.Debugln("chartRepo:", chartRepo)

	chartName := c.Param("name")
	log.Debugln("chartName:", chartName)

	chartVersion := c.Param("version")
	log.Debugln("version:", chartVersion)

	helmEnv := helm.GenerateHelmRepoEnv(auth.GetCurrentOrganization(c.Request).Name)
	response, err := helm.ChartGet(helmEnv, chartRepo, chartName, chartVersion)
	if err != nil {
		log.Error("Error during get helm chart information.", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during get helm chart information.",
			Error:   err.Error(),
		})
		return
	}
	if response == nil {
		c.JSON(http.StatusNotFound, components.ErrorResponse{
			Code:    http.StatusNotFound,
			Error:   "Chart Not Found!",
			Message: "Chart Not Found!",
		})
		return
	}

	c.JSON(http.StatusOK, response)
	return
}
