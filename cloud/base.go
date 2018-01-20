package cloud

import (
	"errors"
	"fmt"
	banzaiTypes "github.com/banzaicloud/banzai-types/components"
	banzaiSimpleTypes "github.com/banzaicloud/banzai-types/components/database"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/database"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"encoding/base64"
)

// ClusterRepresentation combines EC2 and AKS
type ClusterRepresentation struct {
	Id        uint        `json:"id"`
	Name      string      `json:"name"`
	CloudType string      `json:"cloud"`
	*AmazonRepresentation `json:"amazon,omitempty"`
	*AzureRepresentation  `json:"azure,omitempty"`
	*GoogleRepresentation `json:"google,omitempty"`
}

type AmazonRepresentation struct {
	Ip string `json:"ip"`
}

// DeleteFromDb deletes cluster from database
func DeleteFromDb(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) bool {

	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Delete from database")

	if err := database.Delete(&cs).Error; err != nil {
		// delete failed
		banzaiUtils.LogWarn(banzaiConstants.TagDeleteCluster, "Can't delete cluster from database!", err)
		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't delete cluster!",
			JsonKeyResourceId: cs.ID,
			JsonKeyError:      err,
		})
		return false
	}
	return true
}

func updateClusterInDb(c *gin.Context, cluster banzaiSimpleTypes.ClusterSimple) bool {
	banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Update cluster in database")
	if err := database.Model(&banzaiSimpleTypes.ClusterSimple{}).Update(&cluster).Error; err != nil {
		DbSaveFailed(c, err, cluster.Name)
		return false
	}
	return true
}

// UpdateClusterInCloud updates cluster in cloud
// The request's cloud field decided which type of cloud will be called
func UpdateClusterInCloud(c *gin.Context, r *banzaiTypes.UpdateClusterRequest, preCluster banzaiSimpleTypes.ClusterSimple) bool {

	switch r.Cloud {
	case banzaiConstants.Amazon:
		return UpdateClusterAmazonInCloud(r, c, preCluster)
	case banzaiConstants.Azure:
		return UpdateClusterAzureInCloud(r, c, preCluster)
	default:
		return false
	}

}

// DbSaveFailed sends DB operation failed message back
func DbSaveFailed(c *gin.Context, err error, clusterName string) {
	banzaiUtils.LogWarn(banzaiConstants.TagDatabase, "Can't persist cluster into the database!", err)

	SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
		JsonKeyStatus:  http.StatusBadRequest,
		JsonKeyMessage: "Can't persist cluster into the database!",
		JsonKeyName:    clusterName,
		JsonKeyError:   err,
	})
}

// GetClusterFromDB from database
// If no field param was specified automatically use value as ID
// Else it will use field as query column name
func GetClusterFromDB(c *gin.Context) (*banzaiSimpleTypes.ClusterSimple, error) {

	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Get cluster from database")

	var cluster banzaiSimpleTypes.ClusterSimple
	value := c.Param("id")
	field := c.DefaultQuery("field", "")
	if field == "" {
		field = "id"
	}
	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Cluster ID:", value)
	query := fmt.Sprintf("%s = ?", field)
	database.SelectFirstWhere(&cluster, query, value)
	if cluster.ID == 0 {
		errorMsg := fmt.Sprintf("cluster not found: [%s]: %s", field, value)
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, errorMsg)
		SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			JsonKeyStatus:  http.StatusNotFound,
			JsonKeyMessage: errorMsg,
		})
		return nil, errors.New(errorMsg)
	}
	return &cluster, nil

}

//GetClusterSimple legacy EC2
func GetClusterSimple(c *gin.Context) (*banzaiSimpleTypes.ClusterSimple, error) {
	cl, err := GetClusterFromDB(c)
	if err != nil {
		return nil, err
	}
	return cl, nil
}

//DeleteCluster legacy EC2
func DeleteCluster(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) bool {

	clusterType := cs.Cloud
	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Cluster type is ", clusterType)

	switch clusterType {
	case banzaiConstants.Amazon:
		// create amazon cs
		return DeleteAmazonCluster(cs, c)
	case banzaiConstants.Azure:
		// delete azure cs
		return DeleteAzureCluster(cs, c)
	default:
		SendNotSupportedCloudResponse(c, banzaiConstants.TagDeleteCluster)
		return false
	}

}

// SendNotSupportedCloudResponse sends Not-supported-cloud-type error message back
func SendNotSupportedCloudResponse(c *gin.Context, tag string) {
	msg := "Not supported cloud type. Please use one of the following: " + banzaiConstants.Amazon + ", " + banzaiConstants.Azure + ", " + banzaiConstants.Google + "."
	banzaiUtils.LogInfo(tag, msg)
	SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
		JsonKeyStatus:  http.StatusBadRequest,
		JsonKeyMessage: msg,
	})
}

//GetClusterRepresentation legacy EC2
func GetClusterRepresentation(cs *banzaiSimpleTypes.ClusterSimple) *ClusterRepresentation {

	cloudType := cs.Cloud
	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Cloud type is ", cloudType)

	switch cloudType {
	case banzaiConstants.Amazon:
		return ReadClusterAmazon(cs)
	case banzaiConstants.Azure:
		database.SelectFirstWhere(&cs.Azure, banzaiSimpleTypes.AzureClusterSimple{ClusterSimpleId: cs.ID})
		return ReadClusterAzure(cs)
	case banzaiConstants.Google:
		database.SelectFirstWhere(&cs.Google, banzaiSimpleTypes.GoogleClusterSimple{ClusterSimpleId: cs.ID})
		svc, err := GetGoogleServiceClient()
		if err == nil {
			return ReadClusterGoogle(cs, svc)
		}
	default:
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Not supported cloud type")
	}
	return nil
}

//FetchClusterInfo legacy EC2
func FetchClusterInfo(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) {

	cloudType := cs.Cloud
	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Cloud type is ", cloudType)

	switch cloudType {
	case banzaiConstants.Amazon:
		GetClusterInfoAmazon(cs, c)
	case banzaiConstants.Azure:
		// set azure props
		database.SelectFirstWhere(&cs.Azure, banzaiSimpleTypes.AzureClusterSimple{ClusterSimpleId: cs.ID})
		GetClusterInfoAzure(cs, c)
	default:
		// wrong cloud type
		SendNotSupportedCloudResponse(c, banzaiConstants.TagGetCluster)
	}
}

func GetK8SEndpoint(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) (string, error) {

	const LOGTAG = "GetK8SEndpoint"

	cloudType := cs.Cloud
	banzaiUtils.LogInfo(LOGTAG, "Cloud type is ", cloudType)

	switch cloudType {
	case banzaiConstants.Amazon:
		endpoint, err := getAmazonK8SEndpoint(cs, c)
		if err != nil {
			return "", err
		}
		return endpoint, nil
	case banzaiConstants.Azure:
		return getAzureK8SEndpoint(cs)
	default:
		SendNotSupportedCloudResponse(c, LOGTAG)
		return "", errors.New("could not retrieve K8S endpoint")
	}
}

func GetK8SConfig(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) ([]byte, error) {

	const LOGTAG = "GetK8sConfig"

	if cs == nil {
		banzaiUtils.LogInfo(LOGTAG, "<nil> cluster")
		return nil, errors.New("<nil> cluster")
	}
	clusterType := cs.Cloud
	banzaiUtils.LogInfo(LOGTAG, "Cluster type is ", clusterType)
	switch clusterType {
	case banzaiConstants.Amazon:
		banzaiUtils.LogInfo(LOGTAG, "Trying to get AmazonKubernetesConfig")
		cloudCluster, err := GetClusterWithDbCluster(cs, c)
		if err != nil {
			banzaiUtils.LogInfo(LOGTAG, "Error during getting aws cluster")
			return nil, errors.New("error happened during getting aws cluster")
		} else {
			banzaiUtils.LogInfo(LOGTAG, "Get aws cluster succeeded")
		}
		config, err := GetAmazonKubernetesConfig(cloudCluster)
		if err != nil {
			SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
				JsonKeyStatus:  http.StatusInternalServerError,
				JsonKeyMessage: err,
			})
			return nil, err
		}
		return config, nil

	case banzaiConstants.Azure:
		banzaiUtils.LogInfo(LOGTAG, "Trying to get AzureKubernetesConfig")
		b64config, err := getAzureKubernetesConfig(cs)
		if err != nil {
			// something went wrong
			banzaiUtils.LogWarn(LOGTAG, "Error during getting Azure K8S config")
			SetResponseBodyJson(c, err.StatusCode, gin.H{
				JsonKeyStatus: err.StatusCode,
				JsonKeyData:   err.Message,
			})
			return nil, errors.New("error happened during getting K8S config")
		} else {
			banzaiUtils.LogInfo(LOGTAG, "Kubernetes Config retrieve succeeded!")
			config, _ := base64.StdEncoding.DecodeString(b64config.Properties.KubeConfig)
			return config, nil
		}
	default:
		SendNotSupportedCloudResponse(c, LOGTAG)
		return nil, errors.New("error happened during getting K8S config")
	}
}
