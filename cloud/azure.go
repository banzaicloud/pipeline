package cloud

import (
	azureClient "github.com/banzaicloud/azure-aks-client/client"
	azureCluster "github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"net/http"
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

	utils.LogInfo(log, utils.TagValidateCreateCluster, "Start validate create request (azure)")

	if azure == nil {
		utils.LogInfo(log, utils.TagValidateCreateCluster, "Azure is <nil>")
		return false, ""
	}

	if azure == nil {
		msg := "Required field 'azure' is empty."
		utils.LogInfo(log, utils.TagValidateCreateCluster, msg)
		return false, msg
	}

	// ---- [ Node check ] ---- //
	if azure.Node == nil {
		msg := "Required field 'node' is empty."
		utils.LogInfo(log, utils.TagValidateCreateCluster, msg)
		return false, msg
	}

	if len(azure.Node.ResourceGroup) == 0 {
		msg := "Required field 'resourceGroup' is empty."
		utils.LogInfo(log, utils.TagValidateCreateCluster, msg)
		return false, msg
	}

	if azure.Node.AgentCount == 0 {
		utils.LogInfo(log, utils.TagValidateCreateCluster, "Node agentCount set to default value: ", azureDefaultAgentCount)
		azure.Node.AgentCount = azureDefaultAgentCount
	}

	if len(azure.Node.AgentName) == 0 {
		utils.LogInfo(log, utils.TagValidateCreateCluster, "Node agentName set to default value: ", azureDefaultAgentName)
		azure.Node.AgentName = azureDefaultAgentName
	}

	if len(azure.Node.KubernetesVersion) == 0 {
		utils.LogInfo(log, utils.TagValidateCreateCluster, "Node kubernetesVersion set to default value: ", azureDefaultKubernetesVersion)
		azure.Node.KubernetesVersion = azureDefaultKubernetesVersion
	}

	return true, ""
}

// ValidateAzureRequest validates the update request (only azure part). If any of the fields is missing, the method fills
// with stored data.
func (r *UpdateClusterRequest) ValidateAzureRequest(log *logrus.Logger, defaultValue ClusterSimple) (bool, string) {

	utils.LogInfo(log, utils.TagValidateCreateCluster, "Reset amazon fields")

	if r == nil {
		utils.LogInfo(log, utils.TagValidateCreateCluster, "Update request is <nil>")
		return false, ""
	}

	// reset field amazon fields
	r.UpdateClusterAmazon = nil

	utils.LogInfo(log, utils.TagValidateCreateCluster, "Start validate update request (azure)")

	defAzureNode := &UpdateAzureNode{
		AgentCount: defaultValue.Azure.AgentCount,
	}

	// ---- [ Azure field check ] ---- //
	if r.UpdateClusterAzure == nil {
		utils.LogInfo(log, utils.TagValidateCreateCluster, "'azure' field is empty, Load it from stored data.")
		r.UpdateClusterAzure = &UpdateClusterAzure{
			UpdateAzureNode: defAzureNode,
		}
	}

	// ---- [ Node check ] ---- //
	if r.UpdateClusterAzure.UpdateAzureNode == nil {
		utils.LogInfo(log, utils.TagValidateCreateCluster, "'node' field is empty. Load it from stored data.")
		r.UpdateClusterAzure.UpdateAzureNode = defAzureNode
	}

	// ---- [ Node - Agent count check] ---- //
	if r.UpdateClusterAzure.AgentCount == 0 {
		def := defaultValue.Azure.AgentCount
		utils.LogInfo(log, utils.TagValidateCreateCluster, "Node agentCount set to default value: ", def)
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

	utils.LogInfo(log, utils.TagValidateUpdateCluster, "Check stored & updated cluster equals")

	// check equality
	return isUpdateEqualsWithStoredCluster(r, preCl, log)
}

// CreateClusterAzure creates azure cluster in the cloud
func (request *CreateClusterRequest) CreateClusterAzure(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {

	utils.LogInfo(log, utils.TagCreateCluster, "Start create cluster (azure)")

	if request == nil {
		utils.LogInfo(log, utils.TagCreateCluster, "Create request is <nil>")
		return false
	}

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

	utils.LogInfo(log, utils.TagCreateCluster, "Call azure client")

	// call creation
	res, err := azureClient.CreateUpdateCluster(r)
	if err != nil {
		// creation failed
		utils.LogInfo(log, utils.TagCreateCluster, "Cluster creation failed!", err.Message)
		SetResponseBodyJson(c, err.StatusCode, gin.H{
			JsonKeyStatus:  err.StatusCode,
			JsonKeyMessage: err.Message,
		})
		return false
	} else {
		// creation success
		utils.LogInfo(log, utils.TagCreateCluster, "Cluster created successfully!")
		utils.LogInfo(log, utils.TagCreateCluster, "Save create cluster into database")
		if err := db.Save(&cluster2Db).Error; err != nil {
			DbSaveFailed(c, log, err, cluster2Db.Name)
			return false
		}

		utils.LogInfo(log, utils.TagCreateCluster, "Save create cluster into database succeeded")
		SetResponseBodyJson(c, res.StatusCode, res.Value)
		return true
	}

}

func (cs *ClusterSimple) GetAzureClusterStatus(c *gin.Context, db *gorm.DB, log *logrus.Logger) {

	utils.LogInfo(log, utils.TagGetClusterStatus, "Start get cluster status (azure)")

	if cs == nil {
		utils.LogInfo(log, utils.TagGetClusterStatus, "<nil> cluster struct")
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: "",
		})
		return
	}

	utils.LogInfo(log, utils.TagGetClusterStatus, "Load azure props from database")

	// load azure props from db
	db.Where(AzureSimple{ClusterSimpleId: cs.ID}).First(&cs.Azure)
	resp, err := azureClient.GetCluster(cs.Name, cs.Azure.ResourceGroup)
	if err != nil {
		utils.LogInfo(log, utils.TagGetClusterStatus, "Error during get cluster info: ", err.Message)
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: err.Message,
		})
	} else {
		utils.LogInfo(log, utils.TagGetClusterStatus, "Get cluster success")
		stage := resp.Value.Properties.ProvisioningState
		utils.LogInfo(log, utils.TagGetClusterStatus, "Cluster stage is", stage)
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

	utils.LogInfo(log, utils.TagUpdateCluster, "Start updating cluster (azure)")

	if r == nil {
		utils.LogInfo(log, utils.TagUpdateCluster, "<nil> update cluster")
		return false
	}

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
		utils.LogInfo(log, utils.TagUpdateCluster, "Cluster update failed!", err.Message)
		SetResponseBodyJson(c, err.StatusCode, gin.H{
			JsonKeyStatus:  err.StatusCode,
			JsonKeyMessage: err.Message,
		})
		return false
	} else {
		utils.LogInfo(log, utils.TagUpdateCluster, "Cluster update succeeded")
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
	utils.LogInfo(log, utils.TagGetCluster, "Read aks cluster with", cs.Name, "id")

	if cs == nil {
		utils.LogInfo(log, utils.TagGetCluster, "<nil> cluster")
		return nil
	}

	response, err := azureClient.GetCluster(cs.Name, cs.Azure.ResourceGroup)
	if err != nil {
		utils.LogInfo(log, utils.TagGetCluster, "Something went wrong under read:", err)
		return nil
	} else {
		utils.LogInfo(log, utils.TagGetCluster, "Read cluster success")
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
func (cs *ClusterSimple) GetClusterInfoAzure(c *gin.Context, log *logrus.Logger) {
	utils.LogInfo(log, utils.TagGetCluster, "Fetch aks cluster with name:", cs.Name, "in", cs.Azure.ResourceGroup, "resource group.")

	if cs == nil {
		utils.LogInfo(log, utils.TagGetCluster, "<nil> cluster")
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus: http.StatusInternalServerError,
		})
		return
	}

	response, err := azureClient.GetCluster(cs.Name, cs.Azure.ResourceGroup)
	if err != nil {
		// fetch failed
		utils.LogInfo(log, utils.TagGetCluster, "Status code:", err.StatusCode)
		utils.LogInfo(log, utils.TagGetCluster, "Error during get cluster details:", err.Message)
		SetResponseBodyJson(c, err.StatusCode, err)
	} else {
		// fetch success
		utils.LogInfo(log, utils.TagGetCluster, "Status code:", response.StatusCode)
		SetResponseBodyJson(c, response.StatusCode, response)
	}

}

// deleteAzureCluster deletes cluster from azure
func (cs *ClusterSimple) DeleteAzureCluster(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {

	utils.LogInfo(log, utils.TagGetCluster, "Start delete azure cluster")

	if cs == nil {
		utils.LogInfo(log, utils.TagGetCluster, "<nil> cluster")
		return false
	}

	// set azure props
	db.Where(AzureSimple{ClusterSimpleId: cs.ID}).First(&cs.Azure)
	if cs.DeleteClusterAzure(c, cs.Name, cs.Azure.ResourceGroup) {
		utils.LogInfo(log, utils.TagGetCluster, "Delete succeeded")
		return true
	} else {
		utils.LogWarn(log, utils.TagGetCluster, "Can't delete cluster from cloud!")
		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't delete cluster!",
			JsonKeyResourceId: cs.ID,
		})
		return false
	}
}
