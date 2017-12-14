package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cloud"
	"github.com/banzaicloud/pipeline/conf"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/monitor"
	"github.com/banzaicloud/pipeline/notify"
	"github.com/ghodss/yaml"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/helm/pkg/timeconv"
	"github.com/banzaicloud/pipeline/utils"
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

var log *logrus.Logger
var db *gorm.DB
var Version string
var GitRev string

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

	log = conf.Logger()
	utils.LogInfo(log, utils.TagInit, "Logger configured")
	db = conf.Database()
	utils.LogInfo(log, utils.TagInit, "Create table(s):",
		cloud.ClusterSimple.TableName(cloud.ClusterSimple{}),
		cloud.AmazonClusterSimple.TableName(cloud.AmazonClusterSimple{}),
		cloud.AzureSimple.TableName(cloud.AzureSimple{}))
	db.AutoMigrate(&cloud.ClusterSimple{}, &cloud.AmazonClusterSimple{}, &cloud.AzureSimple{})

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

	utils.LogInfo(log, utils.TagDeleteDeployment, "Start delete deployment")

	name := c.Param("name")

	// --- [ Get cluster ] --- //
	utils.LogInfo(log, utils.TagDeleteDeployment, "Get cluster")
	cloudCluster, err := cloud.GetCluster(c, db, log)
	if err != nil {
		return
	}

	// --- [Delete deployment] --- //
	utils.LogInfo(log, utils.TagDeleteDeployment, "Delete deployment")
	err = helm.DeleteDeployment(cloudCluster, name)
	if err != nil {
		// error during delete deployment
		utils.LogWarn(log, utils.TagDeleteDeployment, err.Error())
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: fmt.Sprintf("%s", err),
		})
		return
	} else {
		// delete succeeded
		utils.LogInfo(log, utils.TagDeleteDeployment, "Delete deployment succeeded")
	}
	cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
		cloud.JsonKeyStatus:  http.StatusOK,
		cloud.JsonKeyMessage: "success",
	})
	return
}

// CreateDeployment creates a Helm deployment
func CreateDeployment(c *gin.Context) {

	utils.LogInfo(log, utils.TagCreateDeployment, "Start create deployment")

	// --- [ Get cluster ] --- //
	utils.LogInfo(log, utils.TagCreateDeployment, "Get cluster")
	cloudCluster, err := cloud.GetCluster(c, db, log)
	if err != nil {
		msg := "Error during get cluster cluster. " + err.Error()
		utils.LogInfo(log, utils.TagCreateDeployment, msg)
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: msg,
		})
		return
	}

	utils.LogInfo(log, utils.TagCreateDeployment, "Get cluster succeeded")

	utils.LogInfo(log, utils.TagCreateDeployment, "Bind json into DeploymentType struct")
	var deployment DeploymentType
	if err := c.BindJSON(&deployment); err != nil {
		utils.LogInfo(log, utils.TagCreateDeployment, "Bind failed")
		utils.LogInfo(log, utils.TagCreateDeployment, "Required field is empty."+err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Required field is empty",
			cloud.JsonKeyError:   err,
		})
		return
	}

	utils.LogDebug(log, utils.TagCreateDeployment, fmt.Sprintf("Creating chart %s with version %s and release name %s", deployment.Name, deployment.Version, deployment.ReleaseName))
	prefix := viper.GetString("dev.chartpath")
	chartPath := path.Join(prefix, deployment.Name)

	var values []byte = nil
	if deployment.Values != "" {
		parsedJSON, err := yaml.Marshal(deployment.Values)
		if err != nil {
			utils.LogError(log, utils.TagCreateDeployment, "Can't parse Values:", err)
		}
		values, err = yaml.JSONToYAML(parsedJSON)
		if err != nil {
			utils.LogError(log, utils.TagCreateDeployment, "Can't convert JSON to YAML:", err)
			return
		}
	}
	utils.LogDebug(log, utils.TagCreateDeployment, "Custom values:", string(values))
	utils.LogInfo(log, utils.TagCreateDeployment, "Create deployment")
	release, err := helm.CreateDeployment(cloudCluster, chartPath, deployment.ReleaseName, values)
	if err != nil {
		utils.LogWarn(log, utils.TagCreateDeployment, "Error during create deployment.", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: fmt.Sprintf("%s", err),
		})
		return
	} else {
		utils.LogInfo(log, utils.TagCreateDeployment, "Create deployment succeeded")
	}

	releaseName := release.Release.Name
	releaseNotes := release.Release.Info.Status.Notes

	utils.LogDebug(log, utils.TagCreateDeployment, "Release name:", releaseName)
	utils.LogDebug(log, utils.TagCreateDeployment, "Release notes:", releaseNotes)

	//Get ingress with deployment prefix TODO
	//Get local ingress address?
	deploymentUrl := fmt.Sprintf("http://%s:30080/zeppelin/", cloudCluster.KubernetesAPI.Endpoint)
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

	utils.LogInfo(log, utils.TagListDeployments, "Start listing deployments")

	// --- [ Get cluster ] ---- //
	utils.LogInfo(log, utils.TagListDeployments, "Get cluster")
	cloudCluster, err := cloud.GetCluster(c, db, log)
	if err != nil {
		msg := "Error during getting cluster"
		utils.LogWarn(log, utils.TagListDeployments, "Error during getting cluster:", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: msg,
		})
		return
	} else {
		utils.LogInfo(log, utils.TagListDeployments, "Getting cluster succeeded")
	}

	// --- [ Get deployments ] --- //
	utils.LogInfo(log, utils.TagListDeployments, "Get deployments")
	response, err := helm.ListDeployments(cloudCluster, nil)
	if err != nil {
		utils.LogWarn(log, utils.TagListDeployments, "Error getting deployments. ", err)
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
		utils.LogInfo(log, utils.TagListDeployments, msg)
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

	utils.LogInfo(log, utils.TagCreateCluster, "Cluster creation is stared")
	utils.LogInfo(log, utils.TagCreateCluster, "Bind json into CreateClusterRequest struct")

	// bind request body to struct
	var createClusterBaseRequest cloud.CreateClusterRequest
	if err := c.BindJSON(&createClusterBaseRequest); err != nil {
		// bind failed
		utils.LogError(log, utils.TagCreateCluster, "Required field is empty: "+err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Required field is empty",
			cloud.JsonKeyError:   err,
		})
		return
	} else {
		utils.LogInfo(log, utils.TagCreateCluster, "Bind succeeded")
	}

	utils.LogInfo(log, utils.TagCreateCluster, "Searching entry with name:", createClusterBaseRequest.Name)
	var savedCluster cloud.ClusterSimple
	db.Raw("SELECT * FROM "+cloud.ClusterSimple.TableName(savedCluster)+" WHERE name = ?;", createClusterBaseRequest.Name).Scan(&savedCluster)

	if savedCluster.ID != 0 {
		// duplicated entry
		msg := "Duplicate entry '" + savedCluster.Name + "' for key 'name'"
		utils.LogError(log, utils.TagCreateCluster, msg)
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: msg,
		})
		return
	}

	utils.LogInfo(log, utils.TagCreateCluster, "No entity with this name exists. The creation is possible.")

	cloudType := createClusterBaseRequest.Cloud
	utils.LogInfo(log, utils.TagCreateCluster, "Cloud type is ", cloudType)

	switch cloudType {
	case cloud.Amazon:
		// validate and create Amazon cluster
		awsData := createClusterBaseRequest.Properties.CreateClusterAmazon
		if isValid, err := awsData.Validate(log); isValid && len(err) == 0 {
			utils.LogInfo(log, utils.TagCreateCluster, "Validation is OK")
			if isOk, createdCluster := createClusterBaseRequest.CreateClusterAmazon(c, db, log); isOk {
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
		break
	case cloud.Azure:
		// validate and create Azure cluster
		aksData := createClusterBaseRequest.Properties.CreateClusterAzure
		if isValid, err := aksData.Validate(log); isValid && len(err) == 0 {
			if createClusterBaseRequest.CreateClusterAzure(c, db, log) {
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
		break
	default:
		// wrong cloud type
		cloud.SendNotSupportedCloudResponse(c, log, utils.TagCreateCluster)
		break
	}

}

// DeleteCluster deletes a K8S cluster from the cloud
func DeleteCluster(c *gin.Context) {

	utils.LogInfo(log, utils.TagDeleteCluster, "Delete cluster start")

	var cluster cloud.ClusterSimple
	clusterId := c.Param("id")

	db.First(&cluster, clusterId)

	utils.LogInfo(log, utils.TagDeleteCluster, "Cluster data:", cluster)

	if cluster.ID == 0 {
		// not found cluster with the given ID
		utils.LogInfo(log, utils.TagDeleteCluster, "Clouster not found")
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "No cluster found!",
		})
		return
	}

	if cluster.DeleteCluster(c, db, log) {
		// cluster delete success, delete from db
		if cluster.DeleteFromDb(c, db, log) {
			updatePrometheus()
		}
	}

}

func updatePrometheusWithRetryConf(createdCluster *cluster.Cluster) {
	cloud.RetryGetConfig(createdCluster, "")
	updatePrometheus()
}

func updatePrometheus() {
	err := monitor.UpdatePrometheusConfig(db)
	if err != nil {
		utils.LogWarn(log, utils.TagUpdatePrometheus, "Could not update prometheus configmap: %v", err)
	}
}

// FetchClusters fetches all the K8S clusters from the cloud
func FetchClusters(c *gin.Context) {

	utils.LogInfo(log, utils.TagListClusters, "Start listing clusters")

	var clusters []cloud.ClusterSimple
	var response []*cloud.ClusterRepresentation
	db.Find(&clusters)

	if len(clusters) <= 0 {
		utils.LogInfo(log, utils.TagListClusters, "No clusters found")
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "No clusters found!",
		})
		return
	}

	for _, cl := range clusters {
		clust := cl.GetClusterRepresentation(db, log)
		if clust != nil {
			utils.LogInfo(log, utils.TagListClusters, fmt.Sprintf("Append %#v cluster representation to response", clust))
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

	id := c.Param("id")

	utils.LogInfo(log, utils.TagGetClusterInfo, "Start getting cluster info with", id, "id")

	var cl cloud.ClusterSimple
	db.Where(cloud.ClusterSimple{Model: gorm.Model{ID: utils.ConvertString2Uint(id)}}).First(&cl)

	if cl.ID == 0 {
		msg := "Cluster not found."
		utils.LogWarn(log, utils.TagGetClusterInfo, msg)
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: msg,
		})
		return
	}

	cl.FetchClusterInfo(c, db, log)

}

// UpdateCluster updates a K8S cluster in the cloud (e.g. autoscale)
func UpdateCluster(c *gin.Context) {

	var cl cloud.ClusterSimple
	clusterId := c.Param("id")

	utils.LogInfo(log, utils.TagGetClusterInfo, "Start updating cluster with", clusterId, "id")
	utils.LogInfo(log, utils.TagGetClusterInfo, "Bind json into UpdateClusterRequest struct")

	// bind request body to UpdateClusterRequest struct
	var updateRequest cloud.UpdateClusterRequest
	if err := c.BindJSON(&updateRequest); err != nil {
		// bind failed, required field(s) empty
		utils.LogWarn(log, utils.TagGetClusterInfo, "Bind failed.", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Required field is empty",
			cloud.JsonKeyError:   err,
		})
		return
	}

	utils.LogInfo(log, utils.TagGetClusterInfo, "Load cluster from database")

	// load cluster from db
	db.Where(cloud.ClusterSimple{
		Model: gorm.Model{ID: utils.ConvertString2Uint(clusterId)},
	}).Where(cloud.ClusterSimple{
		Cloud: updateRequest.Cloud,
	}).First(&cl)

	// if ID is 0, the cluster is not found in DB
	if cl.ID == 0 {
		utils.LogInfo(log, utils.TagGetClusterInfo, "No cluster found with!")
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "No cluster found!",
		})
		return
	}

	utils.LogInfo(log, utils.TagGetClusterInfo, "Update request: ", updateRequest)
	cloudType := cl.Cloud

	switch cloudType {
	case cloud.Amazon:
		// read amazon props from amazon_cluster_properties table
		utils.LogInfo(log, utils.TagGetClusterInfo, "Load amazon props from db")
		db.Where(cloud.AmazonClusterSimple{ClusterSimpleId: utils.ConvertString2Uint(clusterId)}).First(&cl.Amazon)
		break
	case cloud.Azure:
		// read azure props from azure_cluster_properties table
		utils.LogInfo(log, utils.TagGetClusterInfo, "Load azure props from db")
		db.Where(cloud.AzureSimple{ClusterSimpleId: utils.ConvertString2Uint(clusterId)}).First(&cl.Azure)
		break
	default:
		// not supported cloud type
		utils.LogWarn(log, utils.TagGetClusterInfo, "Not supported cloud type")
		cloud.SendNotSupportedCloudResponse(c, log, utils.TagUpdateCluster)
		return
	}

	utils.LogInfo(log, utils.TagGetClusterInfo, "Cluster to modify: ", cl)

	if isValid, err := updateRequest.Validate(log, cl); isValid && len(err) == 0 {
		// validation OK
		utils.LogInfo(log, utils.TagGetClusterInfo, "Validate is OK")
		if updateRequest.UpdateClusterInCloud(c, db, log, cl) {
			// cluster updated successfully in cloud
			// update prometheus config..
			updatePrometheus()
		}
	} else {
		// validation failed
		utils.LogInfo(log, utils.TagGetClusterInfo, "Validation failed")
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: err,
		})
	}

}

// FetchClusterConfig fetches a cluster config
func FetchClusterConfig(c *gin.Context) {

	utils.LogInfo(log, utils.TagFetchClusterConfig, "Start fetching cluster config")

	// --- [ Get cluster ] --- //
	utils.LogInfo(log, utils.TagFetchClusterConfig, "Get cluster")
	cloudCluster, err := cloud.GetCluster(c, db, log)
	if err != nil {
		utils.LogInfo(log, utils.TagFetchClusterConfig, "Error during getting cluster")
		return
	} else {
		utils.LogInfo(log, utils.TagFetchClusterConfig, "Get cluster succeeded")
	}

	// --- [ Get config ] --- //
	utils.LogInfo(log, utils.TagFetchClusterConfig, "Get config")
	configPath, err := cloud.RetryGetConfig(cloudCluster, "")
	if err != nil {
		errorMsg := fmt.Sprintf("Error read cluster config: %s", err)
		utils.LogWarn(log, utils.TagFetchClusterConfig, errorMsg)
		cloud.SetResponseBodyJson(c, http.StatusServiceUnavailable, gin.H{
			cloud.JsonKeyStatus:  http.StatusServiceUnavailable,
			cloud.JsonKeyMessage: errorMsg,
		})
		return
	}

	// --- [ Read file ] --- //
	utils.LogInfo(log, utils.TagFetchClusterConfig, "Read file")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		utils.LogInfo(log, utils.TagFetchClusterConfig, "Error during read file:", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			cloud.JsonKeyStatus:  http.StatusInternalServerError,
			cloud.JsonKeyMessage: err,
		})
		return
	} else {
		utils.LogDebug(log, utils.TagFetchClusterConfig, "Read file succeeded:", data)
	}

	ctype := c.NegotiateFormat(gin.MIMEPlain, gin.MIMEJSON)
	switch ctype {
	case gin.MIMEJSON:
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyStatus: http.StatusOK,
			cloud.JsonKeyData:   data,
		})
	default:
		utils.LogDebug(log, utils.TagFetchClusterConfig, "Content-Type: ", ctype)
		c.String(http.StatusOK, string(data))
	}
}

// GetClusterStatus retrieves the cluster status
func GetClusterStatus(c *gin.Context) {

	utils.LogInfo(log, utils.TagGetClusterStatus, "Start getting cluster status")

	// --- [ Get cluster ] --- //
	cloudCluster, err := cloud.GetClusterSimple(c, db, log)
	if err != nil {
		utils.LogWarn(log, utils.TagGetClusterStatus, "Error during get cluster", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: err.Error(),
		})
		return
	} else {
		utils.LogInfo(log, utils.TagGetClusterStatus, "Getting cluster status succeeded")
	}

	cloudType := cloudCluster.Cloud
	utils.LogInfo(log, utils.TagGetClusterStatus, "Cloud type is", cloudType)

	switch cloudType {
	case cloud.Amazon:
		cloudCluster.GetAmazonClusterStatus(c, log)
		break
	case cloud.Azure:
		cloudCluster.GetAzureClusterStatus(c, db, log)
		break
	default:
		cloud.SendNotSupportedCloudResponse(c, log, utils.TagGetClusterStatus)
		return
	}
}

// GetTillerStatus checks if tiller ready to accept deployments
func GetTillerStatus(c *gin.Context) {

	utils.LogInfo(log, utils.TagGetTillerStatus, "Start getting tiller status")

	// --- [ Get cluster ] --- //
	utils.LogInfo(log, utils.TagGetTillerStatus, "Get cluster")
	cloudCluster, err := cloud.GetCluster(c, db, log)
	if err != nil {
		utils.LogWarn(log, utils.TagGetTillerStatus, "Error during getting cluster.", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: err.Error(),
		})
		return
	}

	// --- [ List deployments ] ---- //
	_, err = helm.ListDeployments(cloudCluster, nil)
	if err != nil {
		utils.LogWarn(log, utils.TagGetTillerStatus, "Error during getting deployments.", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusServiceUnavailable, gin.H{
			cloud.JsonKeyStatus:  http.StatusServiceUnavailable,
			cloud.JsonKeyMessage: "Tiller not available",
		})
	} else {
		utils.LogInfo(log, utils.TagGetTillerStatus, "Tiller available")
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyStatus:  http.StatusOK,
			cloud.JsonKeyMessage: "Tiller available",
		})
	}
	return
}

// FetchDeploymentStatus check the status of the Helm deployment
func FetchDeploymentStatus(c *gin.Context) {

	utils.LogInfo(log, utils.TagFetchDeploymentStatus, "Start fetching deployment status")

	name := c.Param("name")

	// --- [ Get cluster ]  --- //
	utils.LogInfo(log, utils.TagFetchDeploymentStatus, "Get cluster with name:", name)
	cloudCluster, err := cloud.GetCluster(c, db, log)
	if err != nil {
		utils.LogWarn(log, utils.TagFetchDeploymentStatus, "Error during get cluster.", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "Cluster not found",
		})
		return
	} else {
		utils.LogInfo(log, utils.TagFetchDeploymentStatus, "Get cluster succeeded:", cloudCluster)
	}

	// --- [ List deployments ] --- //
	utils.LogInfo(log, utils.TagFetchDeploymentStatus, "List deployments")
	chart, err := helm.ListDeployments(cloudCluster, &name)
	if err != nil {
		utils.LogWarn(log, utils.TagFetchDeploymentStatus, "Error during listing deployments:", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusServiceUnavailable, gin.H{
			cloud.JsonKeyStatus:  http.StatusServiceUnavailable,
			cloud.JsonKeyMessage: "Tiller not available",
		})
		return
	} else {
		utils.LogInfo(log, utils.TagFetchDeploymentStatus, "List deployments succeeded")
	}

	if chart.Count == 0 {
		msg := "Deployment not found"
		utils.LogInfo(log, utils.TagFetchDeploymentStatus, msg)
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: msg,
		})
		return
	}

	if chart.Count > 1 {
		msg := "Multiple deployments found"
		utils.LogInfo(log, utils.TagFetchDeploymentStatus, msg)
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: msg,
		})
		return
	}
	// TODO simplify the flow
	// --- [Check deployment state ] --- //
	utils.LogInfo(log, utils.TagFetchDeploymentStatus, "Check deployment state")
	status, err := helm.CheckDeploymentState(cloudCluster, name)
	if err != nil {
		utils.LogWarn(log, utils.TagFetchDeploymentStatus, "Error during check deployment state:", err.Error())
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "Error happened fetching status",
		})
		return
	} else {
		utils.LogInfo(log, utils.TagFetchDeploymentStatus, "Check deployment state")
	}

	msg := fmt.Sprintf("Deployment state is: %s", status)
	utils.LogInfo(log, utils.TagFetchDeploymentStatus, msg)

	if status == "Running" {
		utils.LogInfo(log, utils.TagFetchDeploymentStatus, "Deployment status is: %s", status)
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

//FetchClusters fetches all the K8S clusters from the cloud
func Status(c *gin.Context) {
	var clusters []cloud.ClusterSimple

	utils.LogInfo(log, utils.TagStatus, "Cluster running, subsystems initialized")
	db.Find(&clusters)

	//TODO:add more complex status checks
	//no error on viper, log, db init
	c.JSON(http.StatusOK, gin.H{"Cluster running, subsystems initialized": http.StatusOK})
}
