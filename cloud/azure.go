package cloud

import (
	"github.com/sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
	azureCluster "github.com/banzaicloud/azure-aks-client/cluster"
	azureClient "github.com/banzaicloud/azure-aks-client/client"
)

const (
	azureDefaultAgentCount        = 1
	azureDefaultAgentName         = "agentpool1"
	azureDefaultKubernetesVersion = "1.7.7"
)

type CreateClusterAzure struct {
	Node *CreateAzureNode `json:"node"`
}

type UpdateClusterAzure struct {
	*UpdateAzureNode `json:"node"`
}

type CreateAzureNode struct {
	ResourceGroup     string `json:"resourceGroup"`
	AgentCount        int    `json:"agentCount"`
	AgentName         string `json:"agentName"`
	KubernetesVersion string `json:"kubernetesVersion"`
}

type UpdateAzureNode struct {
	AgentCount int `json:"agentCount"`
}

type AzureSimple struct {
	ClusterSimpleId   uint `gorm:"primary_key"`
	ResourceGroup     string
	AgentCount        int
	AgentName         string
	KubernetesVersion string
}

type AzureRepresentation struct {
	Value azureClient.Value `json:"value"`
}

// TableName sets AzureSimple's table name
func (AzureSimple) TableName() string {
	return tableNameAzureProperties
}

// Validate validates azure cluster create request
func (azure *CreateClusterAzure) Validate(log *logrus.Logger) (bool, string) {

	if azure == nil {
		msg := "Required field 'azure' is empty."
		log.Info(msg)
		return false, msg
	}

	// ---- [ Node check ] ---- //
	if azure.Node == nil {
		msg := "Required field 'node' is empty."
		log.Info(msg)
		return false, msg
	}

	if len(azure.Node.ResourceGroup) == 0 {
		msg := "Required field 'resourceGroup' is empty."
		log.Info(msg)
		return false, msg
	}

	if azure.Node.AgentCount == 0 {
		log.Info("Node agentCount set to default value: ", azureDefaultAgentCount)
		azure.Node.AgentCount = azureDefaultAgentCount
	}

	if len(azure.Node.AgentName) == 0 {
		log.Info("Node agentName set to default value: ", azureDefaultAgentName)
		azure.Node.AgentName = azureDefaultAgentName
	}

	if len(azure.Node.KubernetesVersion) == 0 {
		log.Info("Node kubernetesVersion set to default value: ", azureDefaultKubernetesVersion)
		azure.Node.KubernetesVersion = azureDefaultKubernetesVersion
	}

	return true, ""
}

// ValidateAzureRequest validates the update request (only azure part). If any of the fields is missing, the method fills
// with stored data.
func (r *UpdateClusterRequest) ValidateAzureRequest(log *logrus.Logger, defaultValue ClusterSimple) (bool, string) {

	// reset field amazon fields
	r.UpdateClusterAmazon = nil

	defAzureNode := &UpdateAzureNode{
		AgentCount: defaultValue.Azure.AgentCount,
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

// CreateClusterAzure creates azure cluster in the cloud
func (request *CreateClusterRequest) CreateClusterAzure(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {

	cluster2Db := ClusterSimple{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		Azure: AzureSimple{
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

	// call creation
	res, err := azureClient.CreateUpdateCluster(r)
	if err != nil {
		// creation failed
		log.Info("Cluster creation failed!", err.Message)
		SetResponseBodyJson(c, err.StatusCode, gin.H{
			JsonKeyStatus:  err.StatusCode,
			JsonKeyMessage: err.Message,
		})
		return false
	} else {
		// creation success
		log.Info("Cluster created successfully!")
		if err := db.Save(&cluster2Db).Error; err != nil {
			DbSaveFailed(c, log, err, cluster2Db.Name)
			return false
		}

		SetResponseBodyJson(c, res.StatusCode, res.Value)
		return true
	}

}

func (cs *ClusterSimple) GetAzureClusterStatus(c *gin.Context, db *gorm.DB, log *logrus.Logger) {
	// load azure props from db
	db.Where(AzureSimple{ClusterSimpleId: cs.ID}).First(&cs.Azure)
	resp, err := azureClient.GetCluster(cs.Name, cs.Azure.ResourceGroup)
	if err != nil {
		log.Info("Error during get cluster info: ", err.Message)
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: err.Message,
		})
	} else {
		log.Info("Get cluster success")
		stage := resp.Value.Properties.ProvisioningState
		var msg string
		var code int
		if stage == "Succeeded" {
			msg = "Cluster available"
			code = http.StatusOK
		} else {
			msg = "Cluster not ready yet"
			code = http.StatusNoContent
		}
		SetResponseBodyJson(c, code, gin.H{
			JsonKeyStatus:  code,
			JsonKeyMessage: msg,
		})
	}
}

// updateClusterAzureInCloud updates azure cluster in cloud
func (r *UpdateClusterRequest) UpdateClusterAzureInCloud(c *gin.Context, db *gorm.DB, log *logrus.Logger, preCluster ClusterSimple) bool {

	cluster2Db := ClusterSimple{
		Model:            preCluster.Model,
		Name:             preCluster.Name,
		Location:         preCluster.Location,
		NodeInstanceType: preCluster.NodeInstanceType,
		Cloud:            r.Cloud,
		Azure: AzureSimple{
			ResourceGroup:     preCluster.Azure.ResourceGroup,
			AgentCount:        r.UpdateClusterAzure.AgentCount,
			AgentName:         preCluster.Azure.AgentName,
			KubernetesVersion: preCluster.Azure.KubernetesVersion,
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
		log.Info("Cluster update failed!", err.Message)
		SetResponseBodyJson(c, err.StatusCode, gin.H{
			JsonKeyStatus:  err.StatusCode,
			JsonKeyMessage: err.Message,
		})
		return false
	} else {
		log.Info("Cluster update success")
		// updateDb
		if updateClusterInDb(c, db, log, cluster2Db) {
			// success update
			SetResponseBodyJson(c, res.StatusCode, res.Value)
			return true
		} else {
			return false
		}

	}

}

// ReadClusterAzure load azure props from cloud to list clusters
func (cs *ClusterSimple) ReadClusterAzure(log *logrus.Logger) *ClusterRepresentation {
	log.Info("Read aks cluster with ", cs.Name, " id")
	response, err := azureClient.GetCluster(cs.Name, cs.Azure.ResourceGroup)
	if err != nil {
		log.Infof("Something went wrong under read: %#v", err)
		return nil
	} else {
		log.Info("Read cluster success")
		clust := ClusterRepresentation{
			Id:        cs.ID,
			Name:      cs.Name,
			CloudType: Azure,
			AzureRepresentation: &AzureRepresentation{
				Value: response.Value,
			},
		}
		return &clust
	}
}

// GetClusterInfoAzure fetches azure cluster props with the given name and resource group
func (cs ClusterSimple) GetClusterInfoAzure(c *gin.Context, log *logrus.Logger) {
	log.Info("Fetch aks cluster with name: ", cs.Name, " in ", cs.Azure.ResourceGroup, " resource group.")

	response, err := azureClient.GetCluster(cs.Name, cs.Azure.ResourceGroup)
	if err != nil {
		// fetch failed
		log.Info("Status code: ", err.StatusCode)
		log.Info("Error during get cluster details: ", err.Message)
		SetResponseBodyJson(c, err.StatusCode, err)
	} else {
		// fetch success
		log.Info("Status code: ", response.StatusCode)
		SetResponseBodyJson(c, response.StatusCode, response)
	}

}

// deleteAzureCluster deletes cluster from azure
func (cs *ClusterSimple) DeleteAzureCluster(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {

	// set azure props
	db.Where(AzureSimple{ClusterSimpleId: cs.ID}).First(&cs.Azure)
	if cs.DeleteClusterAzure(c, cs.Name, cs.Azure.ResourceGroup) {
		log.Info("Delete success")
		return true
	} else {
		log.Warning("Can't delete cluster from cloud!")
		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't delete cluster!",
			JsonKeyResourceId: cs.ID,
		})
		return false
	}
}
