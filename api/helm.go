package api

import (
	"fmt"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/timeconv"
	"net/http"
	"os"
)

//DeploymentType definition to describe a Helm deployment
type DeploymentType struct {
	Name        string      `json:"name" binding:"required"`
	ReleaseName string      `json:"releasename"`
	Version     string      `json:"version"`
	Values      interface{} `json:"values"`
}

// CreateDeployment creates a Helm deployment
func CreateDeployment(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagCreateDeployment})
	log.Info("Start create deployment")

	// --- [ Get cluster ] --- //
	log.Info("Get cluster")
	cloudCluster, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	}

	banzaiUtils.LogInfo(banzaiConstants.TagCreateDeployment, "Get cluster succeeded")

	banzaiUtils.LogInfo(banzaiConstants.TagCreateDeployment, "Bind json into DeploymentType struct")
	var deployment DeploymentType
	if err := c.BindJSON(&deployment); err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateDeployment, "Bind failed")
		banzaiUtils.LogInfo(banzaiConstants.TagCreateDeployment, "Required field is empty."+err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Required field is empty",
			cloud.JsonKeyError:   err,
		})
		return
	}

	banzaiUtils.LogDebug(banzaiConstants.TagCreateDeployment, fmt.Sprintf("Creating chart %s with version %s and release name %s", deployment.Name, deployment.Version, deployment.ReleaseName))

	var values []byte = nil
	if deployment.Values != "" {
		parsedJSON, err := yaml.Marshal(deployment.Values)
		if err != nil {
			banzaiUtils.LogError(banzaiConstants.TagCreateDeployment, "Can't parse Values:", err)
		}
		values, err = yaml.JSONToYAML(parsedJSON)
		if err != nil {
			banzaiUtils.LogError(banzaiConstants.TagCreateDeployment, "Can't convert JSON to YAML:", err)
			return
		}
	}
	// --- [ Get K8S Config ] --- //
	kubeConfig, err := cloud.GetK8SConfig(cloudCluster, c)
	if err != nil {
		return
	}
	banzaiUtils.LogInfo(banzaiConstants.TagCreateDeployment, "Getting K8S Config Succeeded")

	banzaiUtils.LogDebug(banzaiConstants.TagCreateDeployment, "Custom values:", string(values))
	banzaiUtils.LogInfo(banzaiConstants.TagCreateDeployment, "Create deployment")
	release, err := helm.CreateDeployment(deployment.Name, deployment.ReleaseName, values, kubeConfig, cloudCluster.Name)
	if err != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagCreateDeployment, "Error during create deployment.", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: fmt.Sprintf("%s", err),
		})
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateDeployment, "Create deployment succeeded")
	}

	releaseName := release.Release.Name
	releaseNotes := release.Release.Info.Status.Notes

	banzaiUtils.LogDebug(banzaiConstants.TagCreateDeployment, "Release name:", releaseName)
	banzaiUtils.LogDebug(banzaiConstants.TagCreateDeployment, "Release notes:", releaseNotes)

	//Get ingress with deployment prefix TODO
	//Get local ingress address?

	cloud.SetResponseBodyJson(c, http.StatusCreated, gin.H{
		cloud.JsonKeyStatus:      http.StatusCreated,
		cloud.JsonKeyReleaseName: releaseName,
		cloud.JsonKeyNotes:       releaseNotes,
	})
	return
}

// ListDeployments lists a Helm deployment
func ListDeployments(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagListDeployments, "Start listing deployments")

	// --- [ Get cluster ] ---- //
	banzaiUtils.LogInfo(banzaiConstants.TagListDeployments, "Get cluster")
	cloudCluster, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	}
	banzaiUtils.LogInfo(banzaiConstants.TagListDeployments, "Getting cluster succeeded")

	// --- [ Get K8S Config ] --- //
	kubeConfig, err := cloud.GetK8SConfig(cloudCluster, c)
	if err != nil {
		return
	}
	banzaiUtils.LogInfo(banzaiConstants.TagListDeployments, "Getting K8S Config Succeeded")

	// --- [ Get deployments ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagListDeployments, "Get deployments")
	response, err := helm.ListDeployments(nil, kubeConfig)
	if err != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagListDeployments, "Error getting deployments. ", err)
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: fmt.Sprintf("%s", err),
		})
		return
	}
	var releases []gin.H
	if len(response.Releases) > 0 {
		for _, r := range response.Releases {
			body := gin.H{
				"name":    r.Name,
				"chart":   fmt.Sprintf("%s-%s", r.Chart.Metadata.Name, r.Chart.Metadata.Version),
				"version": r.Version,
				"updated": timeconv.String(r.Info.LastDeployed),
				"status":  r.Info.Status.Code.String()}
			releases = append(releases, body)
		}
	} else {
		msg := "There is no installed charts."
		banzaiUtils.LogInfo(banzaiConstants.TagListDeployments, msg)
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyMessage: msg,
		})
		return
	}

	cloud.SetResponseBodyJson(c, http.StatusOK, releases)
	return
}

// Check the status of a deployment through the helm client API
func HelmDeploymentStatus(c *gin.Context) {
	// todo error handling - design it, refine it, refactor it

	name := c.Param("name")
	banzaiUtils.LogInfof("HelmDeploymentStatus", "Retrieving status for deployment: %s", name)

	cloudCluster, err := cloud.GetClusterFromDB(c)
	helmDeploymentStatusErrorResponse(c, errors.Wrap(err, "couldn't get the cluster from db"))

	kubeConfig, err := cloud.GetK8SConfig(cloudCluster, c)
	helmDeploymentStatusErrorResponse(c, errors.Wrap(err, "couldn't get the k8s config"))

	status, err := helm.GetDeploymentStatus(name, kubeConfig)

	if err != nil {

		banzaiUtils.LogError("HelmDeploymentStatus", err.Error())
		// convert the status code back - this is specific to the underlying call!
		code, _ := strconv.Atoi(status)

		cloud.SetResponseBodyJson(c, code, gin.H{
			cloud.JsonKeyStatus:  code,
			cloud.JsonKeyMessage: fmt.Sprint(http.StatusText(code), "\n", err.Error()),
		})

		return
	}

	if status != "" {
		banzaiUtils.LogInfof("HelmDeploymentStatus", "Deployment status: %s", status)
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyStatus:  http.StatusOK,
			cloud.JsonKeyMessage: "Deployment status: " + status,
		})
	}

}

// InitHelmInCluster installs Helm on AKS cluster and configure the Helm client
func InitHelmOnCluster(c *gin.Context) {
	banzaiUtils.LogInfo(banzaiConstants.TagHelmInstall, "Start helm install")

	// get cluster from database
	cl, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	}

	kce := fmt.Sprintf("./statestore/%s/config", cl.Name)
	banzaiUtils.LogInfof(banzaiConstants.TagHelmInstall, "Set $KUBECONFIG env to %s", kce)
	os.Setenv("KUBECONFIG", kce)

	// bind request body to struct
	var helmInstall banzaiHelm.Install
	if err := c.BindJSON(&helmInstall); err != nil {
		// bind failed
		banzaiUtils.LogError(banzaiConstants.TagHelmInstall, "Required field is empty: "+err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Required field is empty",
			cloud.JsonKeyError:   err,
		})
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagHelmInstall, "Bind succeeded")
	}

	resp := helm.Install(&helmInstall, cl.Name)
	cloud.SetResponseBodyJson(c, resp.StatusCode, resp)

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
