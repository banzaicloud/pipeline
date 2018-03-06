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

// GetK8sConfig returns the Kubernetes config
func GetK8sConfig(c *gin.Context) (*[]byte, bool) {
	log := logger.WithFields(logrus.Fields{"tag": "GetKubernetesConfig"})
	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return nil, false
	}
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
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
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
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
			c.JSON(http.StatusBadRequest, htype.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Error parsing request",
				Error:   err.Error(),
			})
			return
		}
		values, err = yaml.JSONToYAML(parsedJSON)
		if err != nil {
			log.Errorf("can't convert json to yaml: %s", err)
			c.JSON(http.StatusBadRequest, htype.ErrorResponse{
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
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kubeconfig",
			Error:   err.Error(),
		})
		return
	}

	log.Debug("Custom values: ", string(values))
	release, err := helm.CreateDeployment(deployment.Name, deployment.ReleaseName, values, kubeConfig, commonCluster.GetName())
	if err != nil {
		//TODO distinguish error codes
		log.Error("Error during create deployment.", err.Error())
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
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
		log.Error("Error during create deployment.", err.Error())
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
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

// HelmDeploymentStatus checks the status of a deployment through the helm client API
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
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error listing deployments",
			Error:   err.Error(),
		})
		return
	}
	log.Infof("Deployment status: %d", status)
	c.JSON(http.StatusOK, htype.DeploymentStatusResponse{
		Status:  http.StatusOK,
		Message: "",
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
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error getting kubeconfig",
			Error:   err.Error(),
		})
	}
	// bind request body to struct
	var helmInstall htype.Install
	if err := c.BindJSON(&helmInstall); err != nil {
		// bind failed
		log.Errorf("Required field is empty: %s", err.Error())
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	err = helm.Install(&helmInstall, kubeConfig, commonCluster.GetName())
	if err != nil {
		log.Errorf("Unable to install chart: %s", err.Error())
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
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
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   err.Error(),
		})
		log.Info(message)
		return
	}
	c.JSON(http.StatusOK, htype.StatusResponse{
		Status:  http.StatusOK,
		Message: "Tiller is available",
	})
	return
}

//UpgradeDeployment - N/A
func UpgradeDeployment(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
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
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
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
