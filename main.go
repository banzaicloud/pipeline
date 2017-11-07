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
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/helm/pkg/timeconv"

	monitor "github.com/banzaicloud/pipeline/monitor"
	notify "github.com/banzaicloud/pipeline/notify"
	"github.com/ghodss/yaml"
)

type CreateClusterType struct {
	Name     string `json:"name" binding:"required"`
	Location string `json:"location" binding:"required"`
	Node     struct {
		InstanceType string `json:"instanceType" binding:"required"`
		SpotPrice    string `json:"spotPrice"`
		MinCount     int    `json:"minCount" binding:"required"`
		MaxCount     int    `json:"maxCount" binding:"required"`
		Image        string `json:"image" binding:"required"`
	} `json:"node" binding:"required"`
	Master struct {
		InstanceType string `json:"instanceType" binding:"required"`
		Image        string `json:"image" binding:"required"`
	} `json:"master" binding:"required"`
}

//nodeInstanceType=m3.medium -d nodeInstanceSpotPrice=0.04 -d nodeMin=1 -d nodeMax=3 -d image=ami-6d48500b

type UpdateClusterType struct {
	Node struct {
		MinCount int `json:"minCount" binding:"required"`
		MaxCount int `json:"maxCount" binding:"required"`
	} `json:"node" binding:"required"`
}

type DeploymentType struct {
	Name        string      `json:"name" binding:"required"`
	ReleaseName string      `json:"releaseName" binding:"required"`
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
	db.AutoMigrate(&cloud.ClusterType{})

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
		v1.HEAD("/clusters/:id", GetClusterStatusByName)
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

func UpgradeDeployment(c *gin.Context) {
	return
}

func DeleteDeployment(c *gin.Context) {
	var cluster cloud.ClusterType
	clusterId := c.Param("id")
	name := c.Param("name")

	db.First(&cluster, clusterId)

	if cluster.ID == 0 {
		log.Warning("No cluster found with!")
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No cluster found!"})
		return
	}

	cloudCluster, err := cloud.ReadCluster(cluster)
	if err != nil {
		log.Warning(err.Error())
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No cluster persistent config found!"})
		return
	}
	err = helm.DeleteDeployment(cloudCluster, name)
	if err != nil {
		log.Warning(err.Error())
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": fmt.Sprintf("%s", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "success"})
	return
}

func CreateDeployment(c *gin.Context) {
	var cluster cloud.ClusterType
	clusterId := c.Param("id")
	var deployment DeploymentType

	db.First(&cluster, clusterId)

	if cluster.ID == 0 {
		log.Warning("No cluster found with!")
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No cluster found!"})
		return
	}

	if err := c.BindJSON(&deployment); err != nil {
		log.Info("Required field is empty" + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Required field is empty", "error": err})
		return
	}

	cloudCluster, err := cloud.ReadCluster(cluster)
	if err != nil {
		log.Warning(err.Error())
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No cluster persistent config found!"})
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
	releaseName := release.Release.Name

	releaseNotes := release.Release.Info.Status.Notes
	if err != nil {
		log.Warning(err.Error())
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": fmt.Sprintf("%s", err)})
		return
	}
	//Get ingress with deployment prefix TODO
	//Get local ingress address?
	deploymentUrl := fmt.Sprintf("http://%s:30080/zeppelin/", cloudCluster.KubernetesAPI.Endpoint)
	notify.SlackNotify(fmt.Sprintf("Deployment Created: %s", deploymentUrl))
	c.JSON(http.StatusOK, gin.H{"release_name": releaseName, "url": deploymentUrl, "notes": releaseNotes})
	return
}

func ListDeployments(c *gin.Context) {
	//First get Cluster context
	var cluster cloud.ClusterType
	clusterId := c.Param("id")

	db.First(&cluster, clusterId)

	if cluster.ID == 0 {
		log.Warning("No cluster found with!")
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No cluster found!"})
		return
	}
	cloudCluster, err := cloud.ReadCluster(cluster)
	if err != nil {
		log.Warning(err.Error())
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No cluster persistent config found!"})
		return
	}
	response, err := helm.ListDeployments(cloudCluster, nil)
	if err != nil {
		log.Warning("Error getting deployments. ", err)
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": fmt.Sprintf("%s", err)})
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
		c.JSON(http.StatusOK, gin.H{"message": "There is no installed charts."})
		return
	}

	c.JSON(http.StatusOK, releases)
	return
}

func CreateCluster(c *gin.Context) {
	var createClusterrequest CreateClusterType
	if err := c.BindJSON(&createClusterrequest); err != nil {
		log.Info("Required field is empty" + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Required field is empty", "error": err})
		return
	}

	cluster := cloud.ClusterType{
		Name:                  createClusterrequest.Name,
		Location:              createClusterrequest.Location,
		MasterImage:           createClusterrequest.Master.Image,
		NodeImage:             createClusterrequest.Node.Image,
		MasterInstanceType:    createClusterrequest.Master.InstanceType,
		NodeInstanceType:      createClusterrequest.Node.InstanceType,
		NodeInstanceSpotPrice: createClusterrequest.Node.SpotPrice,
		NodeMin:               createClusterrequest.Node.MinCount,
		NodeMax:               createClusterrequest.Node.MaxCount,
	}

	if err := db.Save(&cluster).Error; err != nil {
		log.Warning("Can't persist cluster into the database!", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Can't persist cluster into the database!", "name": cluster.Name, "error": err})
		return
	}

	if createdCluster, err := cloud.CreateCluster(cluster); err != nil {
		log.Info("Cluster creation failed!", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Could not launch cluster!", "name": cluster.Name, "error": err})
	} else {
		log.Info("Cluster created successfully!")
		c.JSON(http.StatusCreated, gin.H{"status": http.StatusCreated, "message": "Cluster created successfully!", "resourceId": cluster.ID, "name": cluster.Name, "Ip": createdCluster.KubernetesAPI.Endpoint})
		go cloud.RetryGetConfig(createdCluster, "")
	}
	err := monitor.UpdatePrometheusConfig(db)
	if err != nil {
		log.Warning("Could not update prometheus configmap: %v", err)
	}
	return
}

func DeleteCluster(c *gin.Context) {

	var cluster cloud.ClusterType
	clusterId := c.Param("id")

	db.First(&cluster, clusterId)

	if cluster.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No cluster found!"})
		return
	}
	if _, err := cloud.DeleteCluster(cluster); err != nil {
		log.Warning("Can't delete cluster from cloud!", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Can't delete cluster!", "resourceId": cluster.ID, "error": err})
		return
	} else {
		log.Info("Cluster deleted from the cloud!")
		notify.SlackNotify("Cluster deleted from the cloud!")
		c.JSON(http.StatusCreated, gin.H{"status": http.StatusCreated, "message": "Cluster deleted successfully!", "resourceId": cluster.ID})
	}
	if err := db.Delete(&cluster).Error; err != nil {
		log.Warning("Can't delete cluster from database!", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Can't delete cluster!", "resourceId": cluster.ID, "error": err})
		return
	}
	err := monitor.UpdatePrometheusConfig(db)
	if err != nil {
		log.Warning("Could not update prometheus configmap: %v", err)
	}
	return
}

func FetchClusters(c *gin.Context) {
	var clusters []cloud.ClusterType
	type ClusterRepresentation struct {
		Id   uint
		Name string
		Ip   string
	}
	var response []ClusterRepresentation
	db.Find(&clusters)

	if len(clusters) <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No clusters found!"})
		return
	}

	for _, clusterType := range clusters {
		c, err := cloud.ReadCluster(clusterType)
		if err == nil {
			clust := ClusterRepresentation{
				Id:   clusterType.ID,
				Name: clusterType.Name,
				Ip:   c.KubernetesAPI.Endpoint,
			}
			response = append(response, clust)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": response})
}

func FetchCluster(c *gin.Context) {
	var cluster cloud.ClusterType
	clusterId := c.Param("id")

	db.First(&cluster, clusterId)

	if cluster.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No clusters found!"})
		return
	}
	clust, err := cloud.ReadCluster(cluster)
	log.Info(clust.Name)
	if err != nil {
		log.Info("Cluster read failed")
	} else {
		log.Info("Cluster read successful")
		cloud.GetKubeConfig(clust)
	}
	isAvailable, _ := cloud.IsKubernetesClusterAvailable(clust)
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": cluster, "available": isAvailable, "Ip": clust.KubernetesAPI.Endpoint})

}

func UpdateCluster(c *gin.Context) {

	var cluster cloud.ClusterType
	clusterId := c.Param("id")

	db.First(&cluster, clusterId)

	var updateClusterType UpdateClusterType
	if err := c.BindJSON(&updateClusterType); err != nil {
		log.Info("Required field is empty" + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Required field is empty", "error": err})
		return
	}

	if cluster.ID == 0 {
		log.Warning("No cluster found with!")
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No cluster found!"})
		return
	}

	if err := db.Model(&cluster).UpdateColumns(cloud.ClusterType{NodeMin: updateClusterType.Node.MinCount, NodeMax: updateClusterType.Node.MaxCount}).Error; err != nil {
		log.Warning("Can't update cluster in the database!", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Can't update cluster in the database!", "name": cluster.Name, "error": err})
		return
	}

	if _, err := cloud.UpdateCluster(cluster); err != nil {
		log.Warning("Can't update cluster in the cloud!", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Can't update cluster in the cloud!", "resourceId": cluster.ID, "error": err})
	} else {
		log.Info("Cluster updated in the cloud!")
		c.JSON(http.StatusOK, gin.H{"status": http.StatusCreated, "message": "Cluster updated successfully!", "resourceId": cluster.ID})
	}
}

func FetchClusterConfig(c *gin.Context) {
	var cluster cloud.ClusterType
	clusterId := c.Param("id")

	db.First(&cluster, clusterId)

	if cluster.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No clusters found!"})
		return
	}
	clust, err := cloud.ReadCluster(cluster)
	log.Info(clust.Name)
	if err != nil {
		log.Info("Cluster read failed")
	} else {
		log.Info("Cluster read successful")
		cloud.GetKubeConfig(clust)
	}
	path, err := cloud.RetryGetConfig(clust, "")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": http.StatusServiceUnavailable, "message": "Failed to get kubeconf. Cluster not ready yet.", "error": err})
		return
	}
	data, err := ioutil.ReadFile(path)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "Failed to read kubeconf.", "error": err})
		return
	}
	ctype := c.NegotiateFormat(gin.MIMEPlain, gin.MIMEJSON)
	switch ctype {
	case gin.MIMEJSON:
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": data})
	default:
		log.Debug("Content-Type: ", ctype)
		c.String(http.StatusOK, string(data))
	}
}

func GetClusterStatus(c *gin.Context) {
	var cluster cloud.ClusterType
	clusterId := c.Param("id")

	db.First(&cluster, clusterId)

	if cluster.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No clusters found!"})
		return
	}
	clust, err := cloud.ReadCluster(cluster)
	if err != nil {
		log.Info("Cluster read failed")
	} else {
		log.Info("Cluster read successful")
	}
	isAvailable, _ := cloud.IsKubernetesClusterAvailable(clust)
	if isAvailable {
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "Kubernetes cluster available"})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": http.StatusServiceUnavailable, "message": "Kubernetes cluster not ready yet"})
	}
	return
}

func GetClusterStatusByName(c *gin.Context) {
	var cluster cloud.ClusterType
	clusterId := c.Param("name")

	db.First(&cluster, clusterId)

	if cluster.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No clusters found!"})
		return
	}
	clust, err := cloud.ReadCluster(cluster)
	if err != nil {
		log.Info("Cluster read failed")
	} else {
		log.Info("Cluster read successful")
	}
	isAvailable, _ := cloud.IsKubernetesClusterAvailable(clust)
	if isAvailable {
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "Kubernetes cluster available"})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": http.StatusServiceUnavailable, "message": "Kubernetes cluster not ready yet"})
	}
	return
}

func GetTillerStatus(c *gin.Context) {
	var cluster cloud.ClusterType
	clusterId := c.Param("id")

	db.First(&cluster, clusterId)

	if cluster.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No clusters found!"})
		return
	}
	clust, err := cloud.ReadCluster(cluster)
	if err != nil {
		log.Info("Cluster read failed")
	} else {
		log.Info("Cluster read successful")
	}

	_, err = helm.ListDeployments(clust, nil)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": http.StatusServiceUnavailable, "message": "Tiller not available"})
	} else {
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "Tiller available"})
	}
	return
}

func FetchDeploymentStatus(c *gin.Context) {
	var cluster cloud.ClusterType
	clusterId := c.Param("id")
	name := c.Param("name")
	db.First(&cluster, clusterId)

	if cluster.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "No clusters found!"})
		return
	}
	clust, err := cloud.ReadCluster(cluster)
	if err != nil {
		log.Info("Cluster read failed")
	} else {
		log.Info("Cluster read successful")
	}

	chart, err := helm.ListDeployments(clust, &name)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": http.StatusServiceUnavailable, "message": "Tiller not available"})
	}
	if chart.Count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "Deployment not found"})
		return
	}
	if chart.Count > 1 {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Multiple deployments found"})
		return
	}
	foundChart := chart.Releases[0]
	if foundChart.GetInfo().Status.GetCode() == 1 {
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "OK"})
		return
	}
	return
}

func Auth0Test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"Auth0": "authn and authz successfull"})
}
