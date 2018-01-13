package cloud

import (
	azureClient "github.com/banzaicloud/azure-aks-client/client"
	azureCluster "github.com/banzaicloud/azure-aks-client/cluster"
	banzaiTypes "github.com/banzaicloud/banzai-types/components"
	banzaiAzureTypes "github.com/banzaicloud/banzai-types/components/azure"
	banzaiSimpleTypes "github.com/banzaicloud/banzai-types/components/database"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"

	"github.com/gin-gonic/gin"
	"net/http"

	"encoding/base64"
	"fmt"
	"github.com/banzaicloud/banzai-types/database"
	"github.com/go-errors/errors"
	"io/ioutil"
	"os"
)

//AzureRepresentation
type AzureRepresentation struct {
	Value banzaiAzureTypes.Value `json:"value"`
}

// CreateClusterAzure creates azure cluster in the cloud
func CreateClusterAzure(request *banzaiTypes.CreateClusterRequest, c *gin.Context) (bool, *banzaiSimpleTypes.ClusterSimple) {

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Start create cluster (azure)")

	if request == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Create request is <nil>")
		return false, nil
	}

	cluster2Db := banzaiSimpleTypes.ClusterSimple{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		Azure: banzaiSimpleTypes.AzureClusterSimple{
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

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Call azure client")

	// call creation
	_, err := azureClient.CreateUpdateCluster(r)
	if err != nil {
		// creation failed
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Cluster creation failed!", err.Message)
		SetResponseBodyJson(c, err.StatusCode, gin.H{
			JsonKeyStatus:  err.StatusCode,
			JsonKeyMessage: err.Message,
		})
		return false, nil
	} else {
		// creation success
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Cluster created successfully!")
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Save create cluster into database")

		// polling cluster
		pollingRes, err := azureClient.PollingCluster(r.Name, r.ResourceGroup)
		if err != nil {
			// polling error
			SetResponseBodyJson(c, err.StatusCode, err)
			return false, nil
		} else {
			// polling success
			if err := database.Save(&cluster2Db).Error; err != nil {
				DbSaveFailed(c, err, cluster2Db.Name)
				return false, nil
			}

			banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Save create cluster into database succeeded")
			SetResponseBodyJson(c, pollingRes.StatusCode, gin.H{
				JsonKeyStatus: pollingRes.StatusCode,
				JsonKeyResourceId: cluster2Db.ID,
				JsonKeyData: pollingRes.Value,
			})
			return true, &cluster2Db
		}
	}

}

func GetAzureClusterStatus(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Start get cluster status (azure)")

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "<nil> cluster struct")
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: "",
		})
		return
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Load azure props from database")

	// load azure props from db
	database.SelectFirstWhere(&cs.Azure, banzaiSimpleTypes.AzureClusterSimple{ClusterSimpleId: cs.ID})
	resp, err := azureClient.GetCluster(cs.Name, cs.Azure.ResourceGroup)
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Error during get cluster info: ", err.Message)
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: err.Message,
		})
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Get cluster success")
		stage := resp.Value.Properties.ProvisioningState
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Cluster stage is", stage)
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

// UpdateClusterAzureInCloud updates azure cluster in cloud
func UpdateClusterAzureInCloud(r *banzaiTypes.UpdateClusterRequest, c *gin.Context, preCluster banzaiSimpleTypes.ClusterSimple) bool {

	banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Start updating cluster (azure)")

	if r == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "<nil> update cluster")
		return false
	}

	cluster2Db := banzaiSimpleTypes.ClusterSimple{
		Model:            preCluster.Model,
		Name:             preCluster.Name,
		Location:         preCluster.Location,
		NodeInstanceType: preCluster.NodeInstanceType,
		Cloud:            r.Cloud,
		Azure: banzaiSimpleTypes.AzureClusterSimple{
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
		banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Cluster update failed!", err.Message)
		SetResponseBodyJson(c, err.StatusCode, gin.H{
			JsonKeyStatus:  err.StatusCode,
			JsonKeyMessage: err.Message,
		})
		return false
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Cluster update succeeded")
		// updateDb
		if updateClusterInDb(c, cluster2Db) {
			// success update
			SetResponseBodyJson(c, res.StatusCode, gin.H{
				JsonKeyResourceId: cluster2Db.ID,
				JsonKeyData: res.Value,
			})
			return true
		} else {
			return false
		}
	}
}

// ReadClusterAzure load azure props from cloud to list clusters
func ReadClusterAzure(cs *banzaiSimpleTypes.ClusterSimple) *ClusterRepresentation {
	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Read aks cluster with", cs.Name, "id")

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "<nil> cluster")
		return nil
	}

	response, err := azureClient.GetCluster(cs.Name, cs.Azure.ResourceGroup)
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Something went wrong under read:", err)
		return nil
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Read cluster success")
		clust := ClusterRepresentation{
			Id:        cs.ID,
			Name:      cs.Name,
			CloudType: banzaiConstants.Azure,
			AzureRepresentation: &AzureRepresentation{
				Value: response.Value,
			},
		}
		return &clust
	}
}

// GetClusterInfoAzure fetches azure cluster props with the given name and resource group
func GetClusterInfoAzure(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) {
	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Fetch aks cluster with name:", cs.Name, "in", cs.Azure.ResourceGroup, "resource group.")

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "<nil> cluster")
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus: http.StatusInternalServerError,
		})
		return
	}

	response, err := azureClient.GetCluster(cs.Name, cs.Azure.ResourceGroup)
	if err != nil {
		// fetch failed
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Status code:", err.StatusCode)
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Error during get cluster details:", err.Message)
		SetResponseBodyJson(c, err.StatusCode, err)
	} else {
		// fetch success
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Status code:", response.StatusCode)
		SetResponseBodyJson(c, response.StatusCode, gin.H{
			JsonKeyResourceId: cs.ID,
			JsonKeyData: response,
		})
	}

}

// DeleteAzureCluster deletes cluster from azure
func DeleteAzureCluster(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) bool {

	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Start delete azure cluster")

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "<nil> cluster")
		return false
	}

	// set azure props
	database.SelectFirstWhere(&cs.Azure, banzaiSimpleTypes.AzureClusterSimple{ClusterSimpleId: cs.ID})
	if DeleteClusterAzure(c, cs.Name, cs.Azure.ResourceGroup) {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Delete succeeded")
		return true
	} else {
		banzaiUtils.LogWarn(banzaiConstants.TagGetCluster, "Can't delete cluster from cloud!")
		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't delete cluster!",
			JsonKeyResourceId: cs.ID,
		})
		return false
	}
}

//GetAzureK8SConfig retrieves kubeconfig for Azure AKS
func GetAzureK8SConfig(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) {
	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Start loading azure k8s config")

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "<nil> cluster")
		return
	}

	// set azure props
	config, err := getAzureKubernetesConfig(cs)
	if err != nil {
		// something went wrong
		SetResponseBodyJson(c, err.StatusCode, gin.H{
			JsonKeyStatus: err.StatusCode,
			JsonKeyData:   err.Message,
		})
	} else {
		// get config succeeded

		writeConfig2File(fmt.Sprintf("./statestore/%s", cs.Name), config)

		banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Get k8s config succeeded")
		decodedConfig, err := base64.StdEncoding.DecodeString(config.Properties.KubeConfig)
		if err != nil {
			banzaiUtils.LogError(banzaiConstants.TagFetchClusterConfig, "Error decoding config failed:", config)
			SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
				JsonKeyStatus:  http.StatusInternalServerError,
				JsonKeyMessage: err.Error(),
				JsonKeyData:    config,
			})
			return
		}
		SetResponseBodyJson(c, http.StatusOK, string(decodedConfig))
	}

}

func getAzureK8SEndpoint(cs *banzaiSimpleTypes.ClusterSimple) (string, error) {
	resp := ReadClusterAzure(cs)
	if resp == nil {
		return "", errors.New("Could not retrieve K8S endpoint")
	}
	return resp.Value.Properties.Fqdn, nil
}

func getAzureKubernetesConfig(cs *banzaiSimpleTypes.ClusterSimple) (*banzaiAzureTypes.Config, *banzaiTypes.BanzaiResponse) {
	database.SelectFirstWhere(&cs.Azure, banzaiSimpleTypes.AzureClusterSimple{ClusterSimpleId: cs.ID})
	config, err := azureClient.GetClusterConfig(cs.Name, cs.Azure.ResourceGroup, "clusterUser")

	return config, err
}

func writeConfig2File(path string, config *banzaiAzureTypes.Config) {

	if config == nil {
		banzaiUtils.LogWarn(banzaiConstants.TagFetchClusterConfig, "config is nil")
		return
	}

	decodedConfig, _ := base64.StdEncoding.DecodeString(config.Properties.KubeConfig)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0777); err != nil {
			banzaiUtils.LogWarn(banzaiConstants.TagFetchClusterConfig, "error during write to file", err)
			return
		}
	}

	if err := ioutil.WriteFile(fmt.Sprintf("%s/config", path), decodedConfig, 0777); err != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagFetchClusterConfig, "error during write to file", err)
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "write config file succeeded")
	}

}
