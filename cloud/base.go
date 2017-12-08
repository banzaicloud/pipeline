package cloud

import (
	"github.com/jinzhu/gorm"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	azureCluster "github.com/banzaicloud/azure-aks-client/cluster"
	azureClient "github.com/banzaicloud/azure-aks-client/client"
	"fmt"
	"bytes"
	"reflect"
)

const (
	tableNameClusters         = "clusters"
	tableNameAmazonProperties = "amazon_cluster_properties"
	tableNameAzureProperties  = "azure_cluster_properties"
)

const (
	Amazon = "amazon"
	Azure  = "azure"
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

type UpdateClusterRequest struct {
	Cloud string     `json:"cloud" binding:"required"`
	UpdateProperties `json:"properties"`
}

type UpdateProperties struct {
	*UpdateClusterAmazon `json:"amazon"`
	*UpdateClusterAzure  `json:"azure"`
}

func (r UpdateClusterRequest) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Cloud: %s, ", r.Cloud))
	if r.Cloud == Azure && r.UpdateClusterAzure != nil {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("Agent count: %d, Agent name: %s, Kubernetes version: %s",
			r.UpdateClusterAzure.AgentCount,
			r.UpdateClusterAzure.AgentName,
			r.UpdateClusterAzure.KubernetesVersion))
	} else if r.Cloud == Amazon && r.UpdateClusterAzure != nil {
		// Write AWS Node
		if r.UpdateClusterAmazon.UpdateAmazonNode != nil {
			buffer.WriteString(fmt.Sprintf("Min count: %d, Max count: %d",
				r.UpdateClusterAmazon.MinCount,
				r.UpdateClusterAmazon.MaxCount))
		}
	}

	return buffer.String()
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

// String method prints formatted cluster fields
func (cluster CreateClusterSimple) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Id: %d, Creation date: %s, Cloud: %s, NodeInstanceType: %s, ", cluster.ID, cluster.CreatedAt, cluster.Cloud, cluster.NodeInstanceType))
	if cluster.Cloud == Azure {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("Agent count: %d, Agent name: %s, Kubernetes version: %s",
			cluster.Azure.AgentCount,
			cluster.Azure.AgentName,
			cluster.Azure.KubernetesVersion))
	} else if cluster.Cloud == Amazon {
		// Write AWS Master
		buffer.WriteString(fmt.Sprintf("Master instance type: %s, Master image: %s",
			cluster.Amazon.MasterInstanceType,
			cluster.Amazon.MasterImage))
		// Write AWS Node
		buffer.WriteString(fmt.Sprintf("Spot price: %s, Min count: %d, Max count: %d, Node image: %s",
			cluster.Amazon.NodeSpotPrice,
			cluster.Amazon.NodeMinCount,
			cluster.Amazon.NodeMaxCount,
			cluster.Amazon.NodeImage))
	}

	return buffer.String()
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

func (r UpdateClusterRequest) updateClusterAzureInCloud(c *gin.Context, db *gorm.DB, log *logrus.Logger, preCluster CreateClusterSimple) bool {

	cluster2Db := CreateClusterSimple{
		Model:            preCluster.Model,
		Name:             preCluster.Name,
		Location:         preCluster.Location,
		NodeInstanceType: preCluster.NodeInstanceType,
		Cloud:            r.Cloud,
		Azure: CreateAzureSimple{
			ResourceGroup:     preCluster.Azure.ResourceGroup,
			AgentCount:        r.UpdateClusterAzure.AgentCount,
			AgentName:         r.UpdateClusterAzure.AgentName,
			KubernetesVersion: r.UpdateClusterAzure.KubernetesVersion,
		},
	}

	ccr := azureCluster.CreateClusterRequest{
		Name:              cluster2Db.Name,
		Location:          cluster2Db.Location,
		VMSize:            cluster2Db.NodeInstanceType,
		ResourceGroup:     cluster2Db.Azure.ResourceGroup,
		AgentCount:        cluster2Db.Azure.AgentCount,
		AgentName:         cluster2Db.Azure.AgentName,
		KubernetesVersion: cluster2Db.Azure.KubernetesVersion,
	}

	res, err := azureClient.CreateUpdateCluster(ccr)
	if err != nil {
		SetResponseBodyJson(c, err.StatusCode, gin.H{
			JsonKeyStatus:  err.StatusCode,
			JsonKeyMessage: err.Message,
		})
		return false
	} else {
		// updateDb
		if err := db.Model(&CreateClusterSimple{}).Update(&cluster2Db).Error; err != nil {
			DbSaveFailed(c, log, err, cluster2Db.Name)
			return false
		}

		SetResponseBodyJson(c, res.StatusCode, res.Value)
		return true
	}

}

func (r UpdateClusterRequest) updateClusterAmazonInCloud(c *gin.Context, db *gorm.DB, log *logrus.Logger, preCluster CreateClusterSimple) bool {

	cluster2Db := CreateClusterSimple{
		Model:            preCluster.Model,
		Name:             preCluster.Name,
		Location:         preCluster.Location,
		NodeInstanceType: preCluster.NodeInstanceType,
		Cloud:            r.Cloud,
		Amazon: CreateAmazonClusterSimple{
			NodeSpotPrice:      preCluster.Amazon.NodeSpotPrice,
			NodeMinCount:       r.UpdateClusterAmazon.MinCount,
			NodeMaxCount:       r.UpdateClusterAmazon.MaxCount,
			NodeImage:          preCluster.Amazon.NodeImage,
			MasterInstanceType: preCluster.Amazon.MasterInstanceType,
			MasterImage:        preCluster.Amazon.MasterImage,
		},
	}

	if _, err := UpdateClusterAws(cluster2Db); err != nil {
		log.Warning("Can't update cluster in the cloud!", err)

		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't update cluster in the cloud!",
			JsonKeyResourceId: cluster2Db.ID,
			JsonKeyError:      err,
		})

		return false
	} else {
		log.Info("Cluster updated in the cloud!")

		SetResponseBodyJson(c, http.StatusCreated, gin.H{
			JsonKeyStatus:     http.StatusCreated,
			JsonKeyMessage:    "Cluster updated successfully!",
			JsonKeyResourceId: cluster2Db.ID,
		})

		return true
	}

}

func (r *UpdateClusterRequest) UpdateClusterInCloud(c *gin.Context, db *gorm.DB, log *logrus.Logger, preCluster CreateClusterSimple) bool {

	switch r.Cloud {
	case Amazon:
		return r.updateClusterAmazonInCloud(c, db, log, preCluster)
	case Azure:
		return r.updateClusterAzureInCloud(c, db, log, preCluster)
	default:
		return false
	}

}

// The Validate method checks the request fields
func (r *UpdateClusterRequest) Validate(log *logrus.Logger, defaultValue CreateClusterSimple) (bool, string) {

	switch r.Cloud {
	case Amazon:
		// amazon validate
		return r.ValidateAmazonRequest(log, defaultValue)
	case Azure:
		// azure validate
		return r.validateAzureRequest(log, defaultValue)
	default:
		// not supported cloud type
		return false, "Not supported cloud type."
	}

}

// ValidateAmazonRequest validates the update request (only amazon part). If any of the fields is missing, the method fills
// with stored data.
func (r *UpdateClusterRequest) ValidateAmazonRequest(log *logrus.Logger, defaultValue CreateClusterSimple) (bool, string) {

	// reset azure fields
	r.UpdateClusterAzure = nil

	defAmazonNode := &UpdateAmazonNode{
		MinCount: defaultValue.Amazon.NodeMinCount,
		MaxCount: defaultValue.Amazon.NodeMaxCount,
	}

	// ---- [ Amazon field check ] ---- //
	if r.UpdateClusterAmazon == nil {
		log.Info("'amazon' field is empty, Load it from stored data.")
		r.UpdateClusterAmazon = &UpdateClusterAmazon{
			UpdateAmazonNode: defAmazonNode,
		}
	}

	// ---- [ Node check ] ---- //
	if r.UpdateAmazonNode == nil {
		log.Info("'node' field is empty. Fill from stored data")
		r.UpdateAmazonNode = defAmazonNode
	}

	// ---- [ Node min count check ] ---- //
	if r.UpdateAmazonNode.MinCount == 0 {
		defMinCount := defaultValue.Amazon.NodeMinCount
		log.Info("Node minCount set to default value: ", defMinCount)
		r.UpdateAmazonNode.MinCount = defMinCount
	}

	// ---- [ Node max count check ] ---- //
	if r.UpdateAmazonNode.MaxCount == 0 {
		defMaxCount := defaultValue.Amazon.NodeMaxCount
		log.Info("Node maxCount set to default value: ", defMaxCount)
		r.UpdateAmazonNode.MaxCount = defMaxCount
	}

	// ---- [ Node max count > min count check ] ---- //
	if r.UpdateAmazonNode.MaxCount < r.UpdateAmazonNode.MinCount {
		log.Info("Node maxCount is lower than minCount")
		return false, "maxCount must be greater than mintCount"
	}

	// create update request struct with the stored data to check equality
	preCl := &UpdateClusterRequest{
		Cloud: defaultValue.Cloud,
		UpdateProperties: UpdateProperties{
			UpdateClusterAmazon: &UpdateClusterAmazon{
				UpdateAmazonNode: defAmazonNode,
			},
		},
	}

	// check equality
	return isUpdateEqualsWithStoredCluster(r, preCl, log)
}

// ValidateAzureRequest validates the update request (only azure part). If any of the fields is missing, the method fills
// with stored data.
func (r *UpdateClusterRequest) validateAzureRequest(log *logrus.Logger, defaultValue CreateClusterSimple) (bool, string) {

	// reset field amazon fields
	r.UpdateClusterAmazon = nil

	defAzureNode := &UpdateAzureNode{
		AgentCount:        defaultValue.Azure.AgentCount,
		AgentName:         defaultValue.Azure.AgentName,
		KubernetesVersion: defaultValue.Azure.KubernetesVersion,
	}

	// ---- [ Azure field check ] ---- //
	if r.UpdateClusterAzure == nil {
		log.Info("'azure' field is empty, Load it from stored data.")
		r.UpdateClusterAzure = &UpdateClusterAzure{
			UpdateAzureNode: defAzureNode,
		}
	}

	// ---- [ Node check ] ---- //
	if r.UpdateClusterAzure.UpdateAzureNode == nil {
		log.Info("'node' field is empty. Load it from stored data.")
		r.UpdateClusterAzure.UpdateAzureNode = defAzureNode
	}

	// ---- [ Node - Agent count check] ---- //
	if r.UpdateClusterAzure.AgentCount == 0 {
		def := defaultValue.Azure.AgentCount
		log.Info("Node agentCount set to default value: ", def)
		r.UpdateClusterAzure.AgentCount = def
	}

	// ---- [ Node - Agent name check] ---- //
	if len(r.UpdateClusterAzure.AgentName) == 0 {
		def := defaultValue.Azure.AgentName
		log.Info("Node agentName set to default value: ", def)
		r.UpdateClusterAzure.AgentName = def
	}

	// ---- [ Node - Kubernetes version check] ---- //
	if len(r.UpdateClusterAzure.KubernetesVersion) == 0 {
		def := defaultValue.Azure.KubernetesVersion
		log.Info("Node kubernetesVersion set to default value: ", def)
		r.UpdateClusterAzure.KubernetesVersion = def
	}

	// create update request struct with the stored data to check equality
	preCl := &UpdateClusterRequest{
		Cloud: defaultValue.Cloud,
		UpdateProperties: UpdateProperties{
			UpdateClusterAzure: &UpdateClusterAzure{
				UpdateAzureNode: defAzureNode,
			},
		},
	}

	// check equality
	return isUpdateEqualsWithStoredCluster(r, preCl, log)
}

func isUpdateEqualsWithStoredCluster(x interface{}, y interface{}, log *logrus.Logger) (bool, string) {
	if reflect.DeepEqual(x, y) {
		msg := "There is no change in data"
		log.Info(msg)
		return false, msg
	}
	return true, ""
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

	res, err := azureClient.CreateUpdateCluster(r)
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
