package cloud

import (
	"github.com/jinzhu/gorm"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

type CreateClusterTypeBase struct {
	gorm.Model
	Name             string `json:"name" binding:"required" gorm:"unique"`
	Location         string `json:"location" binding:"required"`
	Cloud            string `json:"cloud" binding:"required"`
	NodeInstanceType string `json:"nodeInstanceType" binding:"required"`
	Properties struct {
		gorm.Model
		CreateClusterAmazon *CreateClusterAmazon `json:"amazon"`
		CreateClusterAzure  *CreateClusterAzure  `json:"azure"`
	} `json:"properties" binding:"required"`
}

type UpdatePrometheus interface {
	UpdatePrometheusConfig() error
}

func (cluster CreateClusterTypeBase) CreateClusterAmazon(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {
	if err := db.Save(&cluster).Error; err != nil {
		log.Warning("Can't persist cluster into the database!", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Can't persist cluster into the database!", "name": cluster.Name, "error": err})
		return false
	}

	if createdCluster, err := CreateCluster(cluster); err != nil {
		log.Info("Cluster creation failed!", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "Could not launch cluster!", "name": cluster.Name, "error": err})
	} else {
		log.Info("Cluster created successfully!")
		c.JSON(http.StatusCreated, gin.H{"status": http.StatusCreated, "message": "Cluster created successfully!", "resourceId": cluster.ID, "name": cluster.Name, "Ip": createdCluster.KubernetesAPI.Endpoint})
		go RetryGetConfig(createdCluster, "")
	}
	return true
}
