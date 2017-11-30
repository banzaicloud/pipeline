package cloud

import (
	"github.com/jinzhu/gorm"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	tableNameClusters         = "clusters"
	tableNameAmazonProperties = "amazon_cluster_properties"
	tableNameAzureProperties  = "azure_cluster_properties"
)

type CreateClusterRequest struct {
	Name             string `json:"name" binding:"required"`
	Location         string `json:"location" binding:"required"`
	Cloud            string `json:"cloud" binding:"required"`
	NodeInstanceType string `json:"nodeInstanceType" binding:"required"`
	Properties struct {
		CreateClusterAmazon *CreateClusterAmazon `json:"amazon"`
		CreateClusterAzure  *CreateClusterAzure  `json:"azure"`
	} `json:"properties" binding:"required"`
}

type CreateClusterSimple struct {
	gorm.Model
	Name             string `gorm:"unique"`
	Location         string
	NodeInstanceType string
	Cloud            string
	Amazon           CreateAmazonClusterSimple
	Azure            CreateAzureSimple
}

func (CreateClusterSimple) TableName() string {
	return tableNameClusters
}

func (request CreateClusterRequest) CreateClusterAmazon(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {

	cluster2Db := CreateClusterSimple{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		Amazon: CreateAmazonClusterSimple{
			NodeSpotPrice:      request.Properties.CreateClusterAmazon.Node.SpotPrice,
			NodeMinCount:       request.Properties.CreateClusterAmazon.Node.MinCount,
			NodeMaxCount:       request.Properties.CreateClusterAmazon.Node.MaxCount,
			NodeImage:          request.Properties.CreateClusterAmazon.Node.Image,
			MasterInstanceType: request.Properties.CreateClusterAmazon.Master.InstanceType,
			MasterImage:        request.Properties.CreateClusterAmazon.Master.Image,
		},
	}

	if err := db.Save(&cluster2Db).Error; err != nil {
		DbSaveFailed(c, log, err, cluster2Db.Name)
		return false
	}

	if createdCluster, err := CreateCluster(cluster2Db); err != nil {
		log.Info("Cluster creation failed!", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Could not launch cluster!", "name": cluster2Db.Name, "error": err})
	} else {
		log.Info("Cluster created successfully!")
		c.JSON(http.StatusCreated, gin.H{"status": http.StatusCreated, "message": "Cluster created successfully!", "resourceId": cluster2Db.ID, "name": cluster2Db.Name, "Ip": createdCluster.KubernetesAPI.Endpoint})
		go RetryGetConfig(createdCluster, "")
	}
	return true
}

func (request CreateClusterRequest) CreateClusterAzure(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {
	cluster2Db := CreateClusterSimple{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		Azure: CreateAzureSimple{
			ResourceGroup:     request.Properties.CreateClusterAzure.Node.ResourceGroup,
			AgentCount:        request.Properties.CreateClusterAzure.Node.AgentCount,
			AgentName:         request.Properties.CreateClusterAzure.Node.AgentName,
			KubernetesVersion: request.Properties.CreateClusterAzure.Node.KubernetesVersion,
		},
	}

	if err := db.Save(&cluster2Db).Error; err != nil {
		DbSaveFailed(c, log, err, cluster2Db.Name)
		return false
	}

	// todo call creation

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "ok"})

	return true
}

func DbSaveFailed(c *gin.Context, log *logrus.Logger, err error, clusterName string) {
	log.Warning("Can't persist cluster into the database!", err)
	c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Can't persist cluster into the database!", "name": clusterName, "error": err})
}
