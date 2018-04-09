package api

import (
	"fmt"
	htype "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/timeconv"
	"net/http"
)

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
	parsedRequest, err := parseCreateUpdateDeploymentRequest(c)
	if err != nil {
		log.Errorf(errors.ErrorStack(err))
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}
	release, err := helm.CreateDeployment(parsedRequest.deploymentName,
		parsedRequest.deploymentReleaseName, parsedRequest.values, parsedRequest.kubeConfig,
		parsedRequest.clusterName)
	if err != nil {
		//TODO distinguish error codes
		log.Errorf("Error during create deployment. %s", err.Error())
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
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
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
		log.Errorf(errors.ErrorStack(err))
		c.JSON(http.StatusBadRequest, htype.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during parsing request!",
			Error:   errors.Cause(err).Error(),
		})
		return
	}

	release, err := helm.UpgradeDeployment(name,
		parsedRequest.deploymentName, parsedRequest.values,
		parsedRequest.reuseValues, parsedRequest.kubeConfig, parsedRequest.clusterName)
	if err != nil {
		log.Errorf("Error during upgrading deployment. %s", err.Error())
		c.JSON(http.StatusInternalServerError, htype.ErrorResponse{
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

type parsedDeploymentRequest struct {
	deploymentName        string
	deploymentReleaseName string
	reuseValues           bool
	values                []byte
	kubeConfig            []byte
	clusterName           string
}

func parseCreateUpdateDeploymentRequest(c *gin.Context) (*parsedDeploymentRequest, error) {
	log := logger.WithFields(logrus.Fields{"tag": "parseCreateUpdateDeploymentRequest"})
	pdr := new(parsedDeploymentRequest)

	commonCluster, ok := GetCommonClusterFromRequest(c)
	if ok != true {
		return nil, errors.New("Get cluster failed!")
	}

	pdr.clusterName = commonCluster.GetName()

	var deployment *htype.CreateUpdateDeploymentRequest
	err := c.BindJSON(&deployment)
	if err != nil {
		return nil, errors.Annotate(err, "Error parsing request:")
	}
	log.Info("Parse deployment succeeded")

	log.Debugf("Parsing chart %s with version %s and release name %s", deployment.Name, deployment.Version, deployment.ReleaseName)

	pdr.deploymentName = deployment.Name
	pdr.deploymentReleaseName = deployment.ReleaseName
	pdr.reuseValues = deployment.ReUseValues

	if deployment.Values != "" {
		parsedJSON, err := yaml.Marshal(deployment.Values)
		if err != nil {
			return nil, errors.Annotate(err, "Can't parse Values:")
		}
		pdr.values, err = yaml.JSONToYAML(parsedJSON)
		if err != nil {
			return nil, errors.Annotate(err, "Can't convert json to yaml:")
		}
	}
	pdr.kubeConfig, err = commonCluster.GetK8sConfig()
	if err != nil {
		return nil, errors.Annotate(err, "Error getting kubeconfig:")
	}
	log.Debug("Custom values: ", string(pdr.values))
	return pdr, nil
}
