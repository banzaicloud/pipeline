package cloud

import (
	"github.com/jinzhu/gorm"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	tableNameClusters = "clusters"
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
		log.Warning("Can't persist cluster into the database!", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Can't persist cluster into the database!", "name": cluster2Db.Name, "error": err})
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
