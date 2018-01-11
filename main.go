package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	banzaiTypes "github.com/banzaicloud/banzai-types/components"
	banzaiSimpleTypes "github.com/banzaicloud/banzai-types/components/database"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/database"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	banzaiHelm "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cloud"
	"github.com/banzaicloud/pipeline/conf"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/monitor"
	"github.com/banzaicloud/pipeline/notify"
	"github.com/ghodss/yaml"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/spf13/viper"
	"k8s.io/helm/pkg/timeconv"

	"github.com/banzaicloud/pipeline/utils"
	"github.com/banzaicloud/pipeline/pods"
)

//nodeInstanceType=m3.medium -d nodeInstanceSpotPrice=0.04 -d nodeMin=1 -d nodeMax=3 -d image=ami-6d48500b

//DeploymentType definition to describe a Helm deployment
type DeploymentType struct {
	Name        string      `json:"name" binding:"required"`
	ReleaseName string      `json:"releasename"`
	Version     string      `json:"version"`
	Values      interface{} `json:"values"`
}

//TODO: minCount and Maxcount should be optional, but one of them should be present

//Version of Pipeline
var Version string

//GitRev of Pipeline
var GitRev string

func initDatabase() {
	host := viper.GetString("dev.host")
	port := viper.GetString("dev.port")
	user := viper.GetString("dev.user")
	password := viper.GetString("dev.password")
	dbName := viper.GetString("dev.dbname")
	database.Init(host, port, user, password, dbName)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		if GitRev == "" {
			fmt.Println("version:", Version)
		} else {
			fmt.Printf("version: %s-%s\n", Version, GitRev)
		}
		os.Exit(0)
	}

	conf.Init()
	auth.Init()

	banzaiUtils.LogInfo(banzaiConstants.TagInit, "Logger configured")

	initDatabase()

	banzaiUtils.LogInfo(banzaiConstants.TagInit, "Create table(s):",
		banzaiSimpleTypes.ClusterSimple.TableName(banzaiSimpleTypes.ClusterSimple{}),
		banzaiSimpleTypes.AmazonClusterSimple.TableName(banzaiSimpleTypes.AmazonClusterSimple{}),
		banzaiSimpleTypes.AzureClusterSimple.TableName(banzaiSimpleTypes.AzureClusterSimple{}))
	database.CreateTables(&banzaiSimpleTypes.ClusterSimple{}, &banzaiSimpleTypes.AmazonClusterSimple{}, &banzaiSimpleTypes.AzureClusterSimple{})

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://", "https://"}
	config.AllowMethods = []string{"PUT", "DELETE", "GET", "POST"}
	config.AllowHeaders = []string{"Origin", "Authorization", "Content-Type"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	router.Use(cors.New(config))

	v1 := router.Group("/api/v1/")
	{
		v1.POST("/clusters", CreateCluster)
		v1.GET("/status", Status)
		v1.GET("/clusters", FetchClusters)
		v1.GET("/clusters/:id", FetchCluster)
		v1.PUT("/clusters/:id", UpdateCluster)
		v1.DELETE("/clusters/:id", DeleteCluster)
		v1.HEAD("/clusters/:id", GetClusterStatus)
		v1.GET("/clusters/:id/config", FetchClusterConfig)
		v1.GET("/clusters/:id/deployments", ListDeployments)
		v1.POST("/clusters/:id/deployments", CreateDeployment)
		v1.HEAD("/clusters/:id/deployments", GetTillerStatus)
		v1.DELETE("/clusters/:id/deployments/:name", DeleteDeployment)
		v1.PUT("/clusters/:id/deployments/:name", UpgradeDeployment)
		v1.HEAD("/clusters/:id/deployments/:name", FetchDeploymentStatus)
		v1.POST("/clusters/:id/helminit", InitHelmOnCluster)

		v1.GET("/auth0test", auth.Auth0Groups(auth.ApiGroup), Auth0Test)
	}
	notify.SlackNotify("API is already running")
	router.Run(":9090")

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

// CreateDeployment creates a Helm deployment
func CreateDeployment(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagCreateDeployment, "Start create deployment")

	// --- [ Get cluster ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagCreateDeployment, "Get cluster")
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
	prefix := viper.GetString("dev.chartpath")
	chartPath := path.Join(prefix, deployment.Name)

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
	release, err := helm.CreateDeployment(chartPath, deployment.ReleaseName, values, kubeConfig)
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
	endpoint, err := cloud.GetK8SEndpoint(cloudCluster, c)
	if err != nil {
		cloud.SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			cloud.JsonKeyStatus:  http.StatusInternalServerError,
			cloud.JsonKeyMessage: fmt.Sprintf("%s", err),
		})
		return
	}

	deploymentUrl := fmt.Sprintf("http://%s:30080/zeppelin/", endpoint)
	notify.SlackNotify(fmt.Sprintf("Deployment Created: %s", deploymentUrl))
	cloud.SetResponseBodyJson(c, http.StatusCreated, gin.H{
		cloud.JsonKeyStatus:      http.StatusCreated,
		cloud.JsonKeyMessage:     fmt.Sprintf("%s", err),
		cloud.JsonKeyReleaseName: releaseName,
		cloud.JsonKeyUrl:         deploymentUrl,
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

// CreateCluster creates a K8S cluster in the cloud
func CreateCluster(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Cluster creation is stared")
	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Bind json into CreateClusterRequest struct")

	// bind request body to struct
	var createClusterBaseRequest banzaiTypes.CreateClusterRequest
	if err := c.BindJSON(&createClusterBaseRequest); err != nil {
		// bind failed
		banzaiUtils.LogError(banzaiConstants.TagCreateCluster, "Required field is empty: "+err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Required field is empty",
			cloud.JsonKeyError:   err,
		})
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Bind succeeded")
	}

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Searching entry with name:", createClusterBaseRequest.Name)
	var savedCluster banzaiSimpleTypes.ClusterSimple

	database.Query("SELECT * FROM "+banzaiSimpleTypes.ClusterSimple.TableName(savedCluster)+" WHERE name = ?;",
		createClusterBaseRequest.Name,
		&savedCluster)

	if savedCluster.ID != 0 {
		// duplicated entry
		msg := "Duplicate entry '" + savedCluster.Name + "' for key 'name'"
		banzaiUtils.LogError(banzaiConstants.TagCreateCluster, msg)
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: msg,
		})
		return
	}

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "No entity with this name exists. The creation is possible.")

	cloudType := createClusterBaseRequest.Cloud
	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Cloud type is ", cloudType)

	switch cloudType {
	case banzaiConstants.Amazon:
		// validate and create Amazon cluster
		awsData := createClusterBaseRequest.Properties.CreateClusterAmazon
		if isValid, err := awsData.Validate(); isValid && len(err) == 0 {
			banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Validation is OK")
			if isOk, createdCluster := cloud.CreateClusterAmazon(&createClusterBaseRequest, c); isOk {
				// update prometheus config..
				go updatePrometheusWithRetryConf(createdCluster)
			}
		} else {
			// not valid request
			cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
				cloud.JsonKeyStatus:  http.StatusBadRequest,
				cloud.JsonKeyMessage: err,
			})
		}
	case banzaiConstants.Azure:
		// validate and create Azure cluster
		aksData := createClusterBaseRequest.Properties.CreateClusterAzure
		if isValid, err := aksData.Validate(); isValid && len(err) == 0 {
			if cloud.CreateClusterAzure(&createClusterBaseRequest, c) {
				// update prometheus config..
				updatePrometheus()
			}
		} else {
			// not valid request
			cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
				cloud.JsonKeyStatus:  http.StatusBadRequest,
				cloud.JsonKeyMessage: err,
			})
		}
	default:
		// wrong cloud type
		cloud.SendNotSupportedCloudResponse(c, banzaiConstants.TagCreateCluster)
	}

}

// DeleteCluster deletes a K8S cluster from the cloud
func DeleteCluster(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Delete cluster start")

	cl, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	}

	if cloud.DeleteCluster(cl, c) {
		// cluster delete success, delete from db
		if cloud.DeleteFromDb(cl, c) {
			updatePrometheus()
		}
	}

}

func updatePrometheusWithRetryConf(createdCluster *cluster.Cluster) {
	cloud.RetryGetConfig(createdCluster, "")
	updatePrometheus()
}

func updatePrometheus() {
	err := monitor.UpdatePrometheusConfig()
	if err != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagPrometheus, "Could not update prometheus configmap: %v", err)
	}
}

// FetchClusters fetches all the K8S clusters from the cloud
func FetchClusters(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagListClusters, "Start listing clusters")

	var clusters []banzaiSimpleTypes.ClusterSimple
	var response []*cloud.ClusterRepresentation
	database.Find(&clusters)

	if len(clusters) <= 0 {
		banzaiUtils.LogInfo(banzaiConstants.TagListClusters, "No clusters found")
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "No clusters found!",
		})
		return
	}

	for _, cl := range clusters {
		clust := cloud.GetClusterRepresentation(&cl)
		if clust != nil {
			banzaiUtils.LogInfo(banzaiConstants.TagListClusters, fmt.Sprintf("Append %#v cluster representation to response", clust))
			response = append(response, clust)
		}

	}
	cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
		cloud.JsonKeyStatus: http.StatusOK,
		cloud.JsonKeyData:   response,
	})
}

// FetchCluster fetch a K8S cluster in the cloud
func FetchCluster(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Start getting cluster info")
	cl, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	}

	cloud.FetchClusterInfo(cl, c)

}

// UpdateCluster updates a K8S cluster in the cloud (e.g. autoscale)
func UpdateCluster(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Bind json into UpdateClusterRequest struct")

	// bind request body to UpdateClusterRequest struct
	var updateRequest banzaiTypes.UpdateClusterRequest
	if err := c.BindJSON(&updateRequest); err != nil {
		// bind failed, required field(s) empty
		banzaiUtils.LogWarn(banzaiConstants.TagGetClusterInfo, "Bind failed.", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Required field is empty",
			cloud.JsonKeyError:   err,
		})
		return
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Load cluster from database")

	// load cluster from db
	cl, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Start updating cluster:", cl.Name)

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Update request: ", updateRequest)
	cloudType := cl.Cloud

	switch cloudType {
	case banzaiConstants.Amazon:
		// read amazon props from amazon_cluster_properties table
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Load amazon props from db")
		database.SelectFirstWhere(&cl.Amazon, banzaiSimpleTypes.AmazonClusterSimple{ClusterSimpleId: cl.ID})
	case banzaiConstants.Azure:
		// read azure props from azure_cluster_properties table
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Load azure props from db")
		database.SelectFirstWhere(&cl.Azure, banzaiSimpleTypes.AzureClusterSimple{ClusterSimpleId: cl.ID})
	default:
		// not supported cloud type
		banzaiUtils.LogWarn(banzaiConstants.TagGetClusterInfo, "Not supported cloud type")
		cloud.SendNotSupportedCloudResponse(c, banzaiConstants.TagUpdateCluster)
		return
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Cluster to modify: ", cl)

	if isValid, err := updateRequest.Validate(*cl); isValid && len(err) == 0 {
		// validation OK
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Validate is OK")
		if cloud.UpdateClusterInCloud(c, &updateRequest, *cl) {
			// cluster updated successfully in cloud
			// update prometheus config..
			updatePrometheus()
		}
	} else {
		// validation failed
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Validation failed")
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: err,
		})
	}

}

// FetchClusterConfig fetches a cluster config
func FetchClusterConfig(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Start fetching cluster config")

	// --- [ Get cluster ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Get cluster from database")

	cl, err := cloud.GetClusterFromDB(c)
	if err != nil {
		return
	}

	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Get cluster from database succeeded")
	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Cluster type", cl.Cloud)

	switch cl.Cloud {
	case banzaiConstants.Amazon:
		cloud.GetAmazonK8SConfig(cl, c)
	case banzaiConstants.Azure:
		cloud.GetAzureK8SConfig(cl, c)
	default:
		cloud.SendNotSupportedCloudResponse(c, banzaiConstants.TagFetchClusterConfig)
	}
}

// GetClusterStatus retrieves the cluster status
func GetClusterStatus(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Start getting cluster status")

	// --- [ Get cluster ] --- //
	cloudCluster, err := cloud.GetClusterSimple(c)
	if err != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagGetClusterStatus, "Error during get cluster", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: err.Error(),
		})
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Getting cluster status succeeded")
	}

	cloudType := cloudCluster.Cloud
	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Cloud type is", cloudType)

	switch cloudType {
	case banzaiConstants.Amazon:
		cloud.GetAmazonClusterStatus(cloudCluster, c)
	case banzaiConstants.Azure:
		cloud.GetAzureClusterStatus(cloudCluster, c)
	default:
		cloud.SendNotSupportedCloudResponse(c, banzaiConstants.TagGetClusterStatus)
		return
	}
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

// FetchDeploymentStatus check the status of the Helm deployment
func FetchDeploymentStatus(c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "Start fetching deployment status")

	name := c.Param("name")

	// --- [ Get cluster ]  --- //
	banzaiUtils.LogInfo(banzaiConstants.TagFetchDeploymentStatus, "Get cluster with name:", name)
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

// Auth0Test authN check
func Auth0Test(c *gin.Context) {
	cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
		cloud.JsonKeyAuth0: "authn and authz successful",
	})
}

//Status
func Status(c *gin.Context) {
	var clusters []banzaiSimpleTypes.ClusterSimple

	banzaiUtils.LogInfo(banzaiConstants.TagStatus, "Cluster running, subsystems initialized")
	database.Find(&clusters)

	if len(clusters) == 0 {
		c.JSON(http.StatusOK, gin.H{"No running clusters found.": http.StatusOK})
	} else {
		var clusterStatuses []pods.ClusterStatusResponse
		for _, cl := range clusters {
			clust, err := cloud.GetKubicornCluster(&cl)
			banzaiUtils.LogInfo(utils.TagStatus, "Start listing pods / cluster")
			if err != nil {
				banzaiUtils.LogInfo(utils.TagStatus, err.Error())
			} else {
				var clusterStatusResponse pods.ClusterStatusResponse
				clusterStatusResponse, err = pods.ListPodsForCluster(clust)
				if err == nil {
					clusterStatuses = append(clusterStatuses, clusterStatusResponse)
				} else {
					banzaiUtils.LogError(utils.TagStatus, err)
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"clusterStatuses": clusterStatuses})
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

	resp := helm.Install(&helmInstall)
	cloud.SetResponseBodyJson(c, resp.StatusCode, resp)

}
