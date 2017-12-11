package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
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
	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/helm/pkg/timeconv"
	"strconv"
	azureClient "github.com/banzaicloud/azure-aks-client/client"
)

//nodeInstanceType=m3.medium -d nodeInstanceSpotPrice=0.04 -d nodeMin=1 -d nodeMax=3 -d image=ami-6d48500b

//UpdateClusterType definition to updates a cluster
type UpdateClusterType struct {
	Node struct {
		MinCount int `json:"minCount" binding:"required"`
		MaxCount int `json:"maxCount" binding:"required"`
	} `json:"node" binding:"required"`
}

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

func main() {

	conf.Init()

	log = conf.Logger()
	log.Info("Logger configured")
	db = conf.Database()
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
	name := c.Param("name")
	cloudCluster, err := GetCluster(c)
	if err != nil {
		return
	}
	err = helm.DeleteDeployment(cloudCluster, name)
	if err != nil {
		log.Warning(err.Error())
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: fmt.Sprintf("%s", err),
		})
		return
	}
	cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
		cloud.JsonKeyStatus:  http.StatusOK,
		cloud.JsonKeyMessage: "success",
	})
	return
}

// CreateDeployment creates a Helm deployment
func CreateDeployment(c *gin.Context) {
	var deployment DeploymentType
	cloudCluster, err := GetCluster(c)
	if err != nil {
		return
	}
	if err := c.BindJSON(&deployment); err != nil {
		log.Info("Required field is empty" + err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Required field is empty",
			cloud.JsonKeyError:   err,
		})
		return
	}

	log.Debugf("Creating chart %s with version %s and release name %s", deployment.Name, deployment.Version, deployment.ReleaseName)
	prefix := viper.GetString("dev.chartpath")
	chartPath := path.Join(prefix, deployment.Name)

	var values []byte = nil
	if deployment.Values != "" {
		parsedJSON, err := yaml.Marshal(deployment.Values)
		if err != nil {
			log.Error("Can't parse Values: %v", err)
		}
		values, err = yaml.JSONToYAML(parsedJSON)
		if err != nil {
			log.Error("Can't convert JSON to YAML: %v", err)
			return
		}
	}
	log.Debugf("Custom values: %s", values)
	release, err := helm.CreateDeployment(cloudCluster, chartPath, deployment.ReleaseName, values)
	if err != nil {
		log.Warning(err.Error())
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: fmt.Sprintf("%s", err),
		})
		return
	}
	releaseName := release.Release.Name
	releaseNotes := release.Release.Info.Status.Notes

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
	//First get Cluster context
	cloudCluster, err := GetCluster(c)
	if err != nil {
		return
	}
	response, err := helm.ListDeployments(cloudCluster, nil)
	if err != nil {
		log.Warning("Error getting deployments. ", err)
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
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyMessage: "There is no installed charts.",
		})
		return
	}

	cloud.SetResponseBodyJson(c, http.StatusOK, releases)
	return
}

// CreateCluster creates a K8S cluster in the cloud
func CreateCluster(c *gin.Context) {

	log.Info("Cluster creation is stared")

	// bind request body to struct
	var createClusterBaseRequest cloud.CreateClusterRequest
	if err := c.BindJSON(&createClusterBaseRequest); err != nil {
		log.Info("Required field is empty" + err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Required field is empty",
			cloud.JsonKeyError:   err,
		})
		return
	}

	cloudType := createClusterBaseRequest.Cloud
	log.Info("Cloud type is ", cloudType)

	switch cloudType {
	case cloud.Amazon:
		// validate and create Amazon cluster
		awsData := createClusterBaseRequest.Properties.CreateClusterAmazon
		if isValid, err := awsData.Validate(log); isValid && len(err) == 0 {
			if createClusterBaseRequest.CreateClusterAmazon(c, db, log) {
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
		SendNotSupportedCloudResponse(c)
		break
	}

}

// SendNotSupportedCloudResponse sends Not-supported-cloud-type error message back
func SendNotSupportedCloudResponse(c *gin.Context) {
	msg := "Not supported cloud type. Please use one of the following: " + cloud.Amazon + ", " + cloud.Azure + "."
	cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
		cloud.JsonKeyStatus:  http.StatusBadRequest,
		cloud.JsonKeyMessage: msg,
	})
}

// DeleteCluster deletes a K8S cluster from the cloud
func DeleteCluster(c *gin.Context) {

	log.Info("Delete cluster start")

	var cluster cloud.ClusterSimple
	clusterId := c.Param("id")

	db.First(&cluster, clusterId)

	log.Infof("Cluster data: %#v", cluster)

	if cluster.ID == 0 {
		// not found cluster with the given ID
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "No cluster found!",
		})
		return
	}

	clusterType := cluster.Cloud
	log.Info("Cluster type is ", clusterType)

	switch clusterType {
	case cloud.Amazon:
		// create amazon cluster
		deleteAmazonCluster(c, cluster)
		break
	case cloud.Azure:
		// delete azure cluster
		deleteAzureCluster(c, clusterId, &cluster)
		break
	default:
		SendNotSupportedCloudResponse(c)
		break
	}

}

// deleteAmazonCluster deletes cluster from amazon
func deleteAmazonCluster(c *gin.Context, cluster cloud.ClusterSimple) {
	if _, err := cluster.DeleteClusterAmazon(); err != nil {
		// delete failed
		log.Warning("Can't delete cluster from cloud!", err)

		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:     http.StatusBadRequest,
			cloud.JsonKeyMessage:    "Can't delete cluster!",
			cloud.JsonKeyResourceId: cluster.ID,
			cloud.JsonKeyError:      err,
		})
	} else {
		// delete success
		log.Info("Cluster deleted from the cloud!")
		notify.SlackNotify("Cluster deleted from the cloud!")

		cloud.SetResponseBodyJson(c, http.StatusCreated, gin.H{
			cloud.JsonKeyStatus:     http.StatusCreated,
			cloud.JsonKeyMessage:    "Cluster deleted successfully!",
			cloud.JsonKeyResourceId: cluster.ID,
		})

		// delete from db
		if cluster.DeleteFromDb(c, db, log) {
			updatePrometheus()
		}

	}
}

// deleteAzureCluster deletes cluster from azure
func deleteAzureCluster(c *gin.Context, clusterId string, cluster *cloud.ClusterSimple) {

	// set azure props
	db.Where(cloud.AzureSimple{CreateClusterSimpleId: convertString2Uint(clusterId)}).First(&cluster.Azure)
	if cluster.DeleteClusterAzure(c, cluster.Name, cluster.Azure.ResourceGroup) {
		log.Info("Delete success")
		if cluster.DeleteFromDb(c, db, log) {
			updatePrometheus()
		}
	} else {
		log.Warning("Can't delete cluster from cloud!")
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:     http.StatusBadRequest,
			cloud.JsonKeyMessage:    "Can't delete cluster!",
			cloud.JsonKeyResourceId: cluster.ID,
		})
	}
}

// convertString2Uint converts a string to uint
func convertString2Uint(s string) uint {
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		panic(err)
	}
	return uint(i)
}

func updatePrometheus() {
	err := monitor.UpdatePrometheusConfig(db)
	if err != nil {
		log.Warning("Could not update prometheus configmap: %v", err)
	}
}

// FetchClusters fetches all the K8S clusters from the cloud
func FetchClusters(c *gin.Context) {
	var clusters []cloud.ClusterSimple
	var response []*ClusterRepresentation
	db.Find(&clusters)

	if len(clusters) <= 0 {
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "No clusters found!",
		})
		return
	}

	for _, cl := range clusters {
		cloudType := cl.Cloud
		var clust *ClusterRepresentation
		switch cloudType {
		case cloud.Amazon:
			clust = ReadClusterAmazon(cl)
			break
		case cloud.Azure:
			db.Where(cloud.AzureSimple{CreateClusterSimpleId: cl.ID}).First(&cl.Azure)
			clust = ReadClusterAzure(cl)
			break
		default:
			SendNotSupportedCloudResponse(c)
			break
		}

		if clust != nil {
			log.Infof("Append %#v cluster representation to response", clust)
			response = append(response, clust)
		}

	}
	cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
		cloud.JsonKeyStatus: http.StatusOK,
		cloud.JsonKeyData:   response,
	})
}

type ClusterRepresentation struct {
	Id        uint        `json:"id"`
	Name      string      `json:"name"`
	CloudType string      `json:"cloud"`
	*AmazonRepresentation `json:"amazon"`
	*AzureRepresentation  `json:"azure"`
}

type AzureRepresentation struct {
	Value azureClient.Value `json:"value"`
}

type AmazonRepresentation struct {
	Ip string `json:"ip"`
}

// ReadClusterAzure load azure props from cloud to list clusters
func ReadClusterAzure(cl cloud.ClusterSimple) *ClusterRepresentation {
	log.Info("Read aks cluster with ", cl.Name, " id")
	response, err := azureClient.GetCluster(cl.Name, cl.Azure.ResourceGroup)
	if err != nil {
		log.Infof("Something went wrong under read: %#v", err)
		return nil
	} else {
		log.Info("Read cluster success")
		clust := ClusterRepresentation{
			Id:   cl.ID,
			Name: cl.Name,
			AzureRepresentation: &AzureRepresentation{
				Value: response.Value,
			},
		}
		return &clust
	}
}

// ReadClusterAmazon load amazon props from cloud to list clusters
func ReadClusterAmazon(cl cloud.ClusterSimple) *ClusterRepresentation {
	log.Info("Read aws cluster with ", cl.ID, " id")
	c, err := cloud.ReadCluster(cl)
	if err == nil {
		log.Info("Read aws cluster success")
		clust := ClusterRepresentation{
			Id:   cl.ID,
			Name: cl.Name,
			AmazonRepresentation: &AmazonRepresentation{
				Ip: c.KubernetesAPI.Endpoint,
			},
		}
		return &clust
	} else {
		log.Info("Something went wrong under read: ", err.Error())
	}
	return nil
}

// FetchCluster fetch a K8S cluster in the cloud
func FetchCluster(c *gin.Context) {

	id := c.Param("id")
	var cl cloud.ClusterSimple
	db.Where(cloud.ClusterSimple{Model: gorm.Model{ID: convertString2Uint(id)}}).First(&cl)

	if cl.ID == 0 {
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "Cluster not found.",
		})
		return
	}

	cloudType := cl.Cloud
	log.Info("Cloud type is ", cloudType)

	switch cloudType {
	case cloud.Amazon:
		FetchClusterAmazon(c)
		break
	case cloud.Azure:
		// set azure props
		db.Where(cloud.AzureSimple{CreateClusterSimpleId: convertString2Uint(id)}).First(&cl.Azure)
		FetchClusterAzure(c, cl)
		break
	default:
		// wrong cloud type
		cloud.SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			cloud.JsonKeyStatus:  http.StatusInternalServerError,
			cloud.JsonKeyMessage: "Not supported cloud type.",
		})
		break
	}

}

// FetchClusterAzure fetches azure cluster props with the given name and resource group
func FetchClusterAzure(c *gin.Context, cl cloud.ClusterSimple) {
	log.Info("Fetch aks cluster with name: ", cl.Name, " in ", cl.Azure.ResourceGroup, " resource group.")

	response, err := azureClient.GetCluster(cl.Name, cl.Azure.ResourceGroup)
	if err != nil {
		// fetch failed
		log.Info("Status code: ", err.StatusCode)
		log.Info("Error during get cluster details: ", err.Message)
		cloud.SetResponseBodyJson(c, err.StatusCode, err)
	} else {
		// fetch success
		log.Info("Status code: ", response.StatusCode)
		cloud.SetResponseBodyJson(c, response.StatusCode, response)
	}

}

// FetchClusterAmazon fetches amazon cluster props
func FetchClusterAmazon(c *gin.Context) {
	cluster, err := GetCluster(c)
	if err != nil {
		return
	}
	isAvailable, _ := cloud.IsKubernetesClusterAvailable(cluster)
	cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
		cloud.JsonKeyStatus:    http.StatusOK,
		cloud.JsonKeyData:      cluster,
		cloud.JsonKeyAvailable: isAvailable,
		cloud.JsonKeyIp:        cluster.KubernetesAPI.Endpoint,
	})
}

// UpdateCluster updates a K8S cluster in the cloud (e.g. autoscale)
func UpdateCluster(c *gin.Context) {

	var cl cloud.ClusterSimple
	clusterId := c.Param("id")

	// bind request body to UpdateClusterRequest struct
	var updateRequest cloud.UpdateClusterRequest
	if err := c.BindJSON(&updateRequest); err != nil {
		// bind failed, required field(s) empty
		log.Warning("Required field is empty" + err.Error())
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Required field is empty",
			cloud.JsonKeyError:   err,
		})
		return
	}

	// load cluster from db
	db.Where(cloud.ClusterSimple{
		Model: gorm.Model{ID: convertString2Uint(clusterId)},
	}).Where(cloud.ClusterSimple{
		Cloud: updateRequest.Cloud,
	}).First(&cl)

	// if ID is 0, the cluster is not found in DB
	if cl.ID == 0 {
		log.Warning("No cluster found with!")
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "No cluster found!",
		})
		return
	}

	log.Info("Update request: ", updateRequest)
	cloudType := cl.Cloud

	switch cloudType {
	case cloud.Amazon:
		// read amazon props from amazon_cluster_properties table
		log.Info("Load amazon props from db")
		db.Where(cloud.AmazonClusterSimple{CreateClusterSimpleId: convertString2Uint(clusterId)}).First(&cl.Amazon)
		break
	case cloud.Azure:
		// read azure props from azure_cluster_properties table
		log.Info("Load azure props from db")
		db.Where(cloud.AzureSimple{CreateClusterSimpleId: convertString2Uint(clusterId)}).First(&cl.Azure)
		break
	default:
		// not supported cloud type
		log.Warning("Not supported cloud type")
		SendNotSupportedCloudResponse(c)
		return
	}

	log.Info("Cluster to modify: ", cl)

	if isValid, err := updateRequest.Validate(log, cl); isValid && len(err) == 0 {
		// validation OK
		if updateRequest.UpdateClusterInCloud(c, db, log, cl) {
			// cluster updated successfully in cloud
			// update prometheus config..
			updatePrometheus()
		}
	} else {
		// validation failed
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: err,
		})
	}

}

// FetchClusterConfig fetches a cluster config
func FetchClusterConfig(c *gin.Context) {
	cloudCluster, err := GetCluster(c)
	if err != nil {
		return
	}
	configPath, err := cloud.RetryGetConfig(cloudCluster, "")
	if err != nil {
		errorMsg := fmt.Sprintf("Error read cluster config: %s", err)
		cloud.SetResponseBodyJson(c, http.StatusServiceUnavailable, gin.H{
			cloud.JsonKeyStatus:  http.StatusServiceUnavailable,
			cloud.JsonKeyMessage: errorMsg,
		})
		return
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		cloud.SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			cloud.JsonKeyStatus:  http.StatusInternalServerError,
			cloud.JsonKeyMessage: err,
		})
		return
	}
	ctype := c.NegotiateFormat(gin.MIMEPlain, gin.MIMEJSON)
	switch ctype {
	case gin.MIMEJSON:
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyStatus: http.StatusOK,
			cloud.JsonKeyData:   data,
		})
	default:
		log.Debug("Content-Type: ", ctype)
		c.String(http.StatusOK, string(data))
	}
}

// GetClusterStatus retrieves the cluster status
func GetClusterStatus(c *gin.Context) {
	cluster, err := GetClusterFromDB(c)
	if err != nil {
		return
	}
	clust, err := cloud.ReadClusterOld(*cluster)
	if err != nil {
		log.Info("Cluster read failed")
	} else {
		log.Info("Cluster read successful")
	}
	isAvailable, _ := cloud.IsKubernetesClusterAvailable(clust)
	if isAvailable {
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyStatus:  http.StatusOK,
			cloud.JsonKeyMessage: "Kubernetes cluster available",
		})
	} else {
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyStatus:  http.StatusOK,
			cloud.JsonKeyMessage: "Kubernetes cluster not ready yet",
		})
	}
	return
}

// GetTillerStatus checks if tiller ready to accept deployments
func GetTillerStatus(c *gin.Context) {
	cloudCluster, err := GetCluster(c)
	if err != nil {
		return
	}
	_, err = helm.ListDeployments(cloudCluster, nil)
	if err != nil {
		cloud.SetResponseBodyJson(c, http.StatusServiceUnavailable, gin.H{
			cloud.JsonKeyStatus:  http.StatusServiceUnavailable,
			cloud.JsonKeyMessage: "Tiller not available",
		})
	} else {
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyStatus:  http.StatusOK,
			cloud.JsonKeyMessage: "Tiller available",
		})
	}
	return
}

// FetchDeploymentStatus check the status of the Helm deployment
func FetchDeploymentStatus(c *gin.Context) {
	name := c.Param("name")
	cloudCluster, err := GetCluster(c)
	if err != nil {
		return
	}
	chart, err := helm.ListDeployments(cloudCluster, &name)
	if err != nil {
		cloud.SetResponseBodyJson(c, http.StatusServiceUnavailable, gin.H{
			cloud.JsonKeyStatus:  http.StatusServiceUnavailable,
			cloud.JsonKeyMessage: "Tiller not available",
		})
		return
	}
	if chart.Count == 0 {
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: "Deployment not found",
		})
		return
	}
	if chart.Count > 1 {
		cloud.SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			cloud.JsonKeyStatus:  http.StatusBadRequest,
			cloud.JsonKeyMessage: "Multiple deployments found",
		})
		return
	}
	foundChart := chart.Releases[0]
	if foundChart.GetInfo().Status.GetCode() == 1 {
		cloud.SetResponseBodyJson(c, http.StatusOK, gin.H{
			cloud.JsonKeyStatus:  http.StatusOK,
			cloud.JsonKeyMessage: "OK",
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

// GetCluster from database
// If no field param was specified automatically use value as ID
// Else it will use field as query column name
func GetClusterFromDB(c *gin.Context) (*cloud.ClusterType, error) {
	var cluster cloud.ClusterType
	value := c.Param("id")
	field := c.DefaultQuery("field", "")
	if field == "" {
		field = "id"
	}
	query := fmt.Sprintf("%s = ?", field)
	db.Where(query, value).First(&cluster)
	if cluster.ID == 0 {
		errorMsg := fmt.Sprintf("cluster not found: [%s]: %s", field, value)
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: errorMsg,
		})
		return nil, errors.New(errorMsg)
	}
	return &cluster, nil

}

// GetCluster based on ClusterType object
// This will read the persisted Kubicorn cluster format
func GetKubicornCluster(clusterType *cloud.ClusterType) (*cluster.Cluster, error) {
	clust, err := cloud.ReadClusterOld(*clusterType)
	if err != nil {
		return nil, err
	}
	log.Info("Cluster read successful")
	return clust, nil
}

func GetCluster(c *gin.Context) (*cluster.Cluster, error) {
	clusterType, err := GetClusterFromDB(c)
	if err != nil {
		return nil, err
	}
	cluster, err := GetKubicornCluster(clusterType)
	if err != nil {
		errorMsg := fmt.Sprintf("Error read cluster: %s", err)
		cloud.SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			cloud.JsonKeyStatus:  http.StatusNotFound,
			cloud.JsonKeyMessage: errorMsg,
		})
		return nil, err
	}
	return cluster, nil
}
