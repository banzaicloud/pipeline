package cloud

import (
	"github.com/jinzhu/gorm"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	azureCluster "github.com/banzaicloud/azure-aks-client/cluster"
	azureClient "github.com/banzaicloud/azure-aks-client/client"
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

func (cluster CreateClusterSimple) DeleteFromDb(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {

	log.Info("Delete from db")

	if err := db.Delete(&cluster).Error; err != nil {
		log.Warning("Can't delete cluster from database!", err)
		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't delete cluster!",
			JsonKeyResourceId: cluster.ID,
			JsonKeyError:      err,
		})
		return false
	}
	return true
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
		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyMessage: "Could not launch cluster!",
			JsonKeyName:    cluster2Db.Name,
			JsonKeyError:   err,
		})
	} else {
		log.Info("Cluster created successfully!")
		SetResponseBodyJson(c, http.StatusCreated, gin.H{
			JsonKeyStatus:     http.StatusCreated,
			JsonKeyMessage:    "Cluster created successfully!",
			JsonKeyResourceId: cluster2Db.ID,
			JsonKeyName:       cluster2Db.Name,
			JsonKeyIp:         createdCluster.KubernetesAPI.Endpoint,
		})
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

	r := azureCluster.CreateClusterRequest{
		Name:              cluster2Db.Name,
		Location:          cluster2Db.Location,
		VMSize:            cluster2Db.NodeInstanceType,
		ResourceGroup:     cluster2Db.Azure.ResourceGroup,
		AgentCount:        cluster2Db.Azure.AgentCount,
		AgentName:         cluster2Db.Azure.AgentName,
		KubernetesVersion: cluster2Db.Azure.KubernetesVersion,
	}

	res, err := azureClient.CreateCluster(r)
	if err != nil {
		SetResponseBodyJson(c, err.StatusCode, gin.H{
			JsonKeyStatus:  err.StatusCode,
			JsonKeyMessage: err.Message,
		})
		return false
	} else {
		if err := db.Save(&cluster2Db).Error; err != nil {
			DbSaveFailed(c, log, err, cluster2Db.Name)
			return false
		}

		SetResponseBodyJson(c, res.StatusCode, res.Value)
		return true
	}

}

func DbSaveFailed(c *gin.Context, log *logrus.Logger, err error, clusterName string) {
	log.Warning("Can't persist cluster into the database!", err)

	SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
		JsonKeyStatus:  http.StatusBadRequest,
		JsonKeyMessage: "Can't persist cluster into the database!",
		JsonKeyName:    clusterName,
		JsonKeyError:   err,
	})
}
