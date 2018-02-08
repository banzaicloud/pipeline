package api

import (
	"fmt"
	htype "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/timeconv"
	"net/http"
)

func GetK8sConfig(c *gin.Context) (*[]byte, bool) {
	log := logger.WithFields(logrus.Fields{"tag": "GetKubernetesConfig"})
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return nil, false
	}
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
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
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}
	log.Info("Get cluster succeeded")
	var deployment *htype.CreateDeploymentRequest
	err := c.BindJSON(&deployment)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Parse deployment succeeded")

	log.Debugf("Creating chart %s with version %s and release name %s", deployment.Name, deployment.Version, deployment.ReleaseName)
	var values []byte
	if deployment.Values != "" {
		parsedJSON, err := yaml.Marshal(deployment.Values)
		if err != nil {
			log.Error("can't parse Values:", err)
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Error parsing request",
				Error:   err.Error(),
			})
			return
		}
		values, err = yaml.JSONToYAML(parsedJSON)
		if err != nil {
			log.Errorf("can't convert json to yaml: %s", err)
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Error parsing request",
				Error:   err.Error(),
			})
			return
		}
	}
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kubeconfig",
			Error:   err.Error(),
		})
		return
	}

	log.Debug("Custom values:", string(values))
	release, err := helm.CreateDeployment(deployment.Name, deployment.ReleaseName, values, kubeConfig)
	if err != nil {
		//TODO distinguish error codes
		log.Errorf("Error during create deployment.", err.Error())
		c.JSON(http.StatusBadRequest, ErrorResponse{
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
	response := htype.CreateDeploymentResponse{
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
		log.Errorf("Error during create deployment.", err.Error())
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing deployments",
			Error:   err.Error(),
		})
		return
	}
	var releases []htype.ListDeploymentResponse
	if len(response.Releases) > 0 {
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

// Check the status of a deployment through the helm client API Check what is this?
func HelmDeploymentStatus(c *gin.Context) {
	// todo error handling - design it, refine it, refactor it
	log := logger.WithFields(logrus.Fields{"tag": "DeploymentStatus"})
	name := c.Param("name")
	log.Infof("Retrieving status for deployment: %s", name)
	kubeConfig, ok := GetK8sConfig(c)
	if ok != true {
		return
	}
	status, err := helm.GetDeploymentStatus(name, kubeConfig)
	if err != nil {
		log.Errorf(err.Error())
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing deployments",
			Error:   err.Error(),
		})
		return
	}
	log.Infof("HelmDeploymentStatus", "Deployment status: %s", status)
	c.JSON(http.StatusOK, htype.DeploymentStatusResponse{
		Status:  http.StatusOK,
		Message: "",
	})
}

// InitHelmInCluster installs Helm on AKS cluster and configure the Helm client
func InitHelmOnCluster(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagHelmInstall})
	log.Info("Start helm install")

	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return
	}

	kubeConfig, err := commonCluster.GetK8sConfig()
	log.Error(err)
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Code:    http.StatusBadRequest,
		Message: "Error getting kubeconfig",
		Error:   err.Error(),
	})
	// bind request body to struct
	var helmInstall htype.Install
	if err := c.BindJSON(&helmInstall); err != nil {
		// bind failed
		log.Errorf("Required field is empty: %s", err.Error())
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	x := helm.Install(&helmInstall, kubeConfig, commonCluster.GetName())
	message := "helm initialising"
	c.JSON(http.StatusCreated, htype.InstallResponse{
		Status:  http.StatusCreated,
		Message: message,
	})
	log.Info(message)
	return
}

// FetchDeploymentStatus check the status of the Helm deployment
func FetchDeploymentStatus(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "Start fetching deployment status")

	name := c.Param("name")
	banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "Get deployment with name:", name)

	// --- [ Get cluster ]  --- //
	cloudCluster, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "Get cluster succeeded:", cloudCluster)
	}

	// --- [ Get K8S Config ] --- //
	kubeConfig, err := cloud.GetK8SConfig(cloudCluster, c)
	if err != nil {
		return
	}
	banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "Getting K8S Config Succeeded")

	// --- [ List deployments ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "List deployments")
	chart, err := helm.ListDeployments(&name, kubeConfig)
	if err != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagFetchDeploymentStatus, "Error during listing deployments:", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusServiceUnavailable, gin.H{
			cloud.JsonKeyStatus:  http.StatusServiceUnavailable,
			cloud.JsonKeyMessage: "Tiller not available",
		})
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "List deployments succeeded")
	}

	if chart.Count == 0 {
		msg := "Deployment not found"
		banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, msg)
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: msg,
		})
		return
	}

	if chart.Count > 1 {
		msg := "Multiple deployments found"
		banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, msg)
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: msg,
		})
		return
	}
	// TODO simplify the flow
	// --- [Check deployment state ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "Check deployment state")
	status, err := helm.CheckDeploymentState(cloudCluster, name)
	if err != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagFetchDeploymentStatus, "Error during check deployment state:", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "Error happened fetching status",
		})
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "Check deployment state")
	}

	msg := fmt.Sprintf("Deployment state is: %s", status)
	banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, msg)

	if status == "Running" {
		banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "Deployment status is: %s", status)
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyStatus:  http.StatusOK,
			cloud.JsonKeyMessage: msg,
		})
		return
	} else {
		cloud.SetResponseBodyJson(c, http.StatusNoContent, gin.H{
			cloud.JsonKeyStatus:  http.StatusNoContent,
			cloud.JsonKeyMessage: msg,
		})
		return
	}
	return
}

// GetTillerStatus checks if tiller ready to accept deployments
func GetTillerStatus(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagGetTillerStatus, "Start getting tiller status")

	// --- [ Get cluster ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagGetTillerStatus, "Get cluster")
	cloudCluster, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagGetTillerStatus, "Get cluster succeeded:", cloudCluster)
	}

	// --- [ Get K8S Config ] --- //
	kubeConfig, err := cloud.GetK8SConfig(cloudCluster, c)
	if err != nil {
		return
	}
	banzaiUtils.LogInfo(banzaiConstants.TagGetTillerStatus, "Getting K8S Config Succeeded")

	// --- [ List deployments ] ---- //
	_, err = helm.ListDeployments(nil, kubeConfig)
	if err != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagGetTillerStatus, "Error during getting deployments.", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusServiceUnavailable, gin.H{
			cloud.JsonKeyStatus:  http.StatusServiceUnavailable,
			cloud.JsonKeyMessage: "Tiller not available",
		})
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagGetTillerStatus, "Tiller available")
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyStatus:  http.StatusOK,
			cloud.JsonKeyMessage: "Tiller available",
		})
	}
	return
}

//UpgradeDeployment - N/A
func UpgradeDeployment(c *gin.Context) {
	return
}

//DeleteDeployment deletes a Helm deployment
func DeleteDeployment(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagDeleteDeployment, "Start delete deployment")

	name := c.Param("name")

	// --- [ Get cluster ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagDeleteDeployment, "Get cluster")
	cloudCluster, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	}
	// --- [ Get K8S Config ] --- //
	kubeConfig, err := cloud.GetK8SConfig(cloudCluster, c)
	if err != nil {
		return
	}
	banzaiUtils.LogInfo(banzaiConstants.TagDeleteDeployment, "Getting K8S Config Succeeded")

	// --- [Delete deployment] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagDeleteDeployment, "Delete deployment")
	err = helm.DeleteDeployment(name, kubeConfig)
	if err != nil {
		// error during delete deployment
		banzaiUtils.LogWarn(banzaiConstants.TagDeleteDeployment, err.Error())
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: fmt.Sprintf("%s", err),
		})
		return
	} else {
		// delete succeeded
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteDeployment, "Delete deployment succeeded")
	}
	cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
		cloud.JsonKeyStatus:  http.StatusOK,
		cloud.JsonKeyMessage: "success",
	})
	return
}
