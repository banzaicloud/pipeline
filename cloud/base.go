package cloud

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"net/http"
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
	Properties       struct {
		CreateClusterAmazon *CreateClusterAmazon `json:"amazon"`
		CreateClusterAzure  *CreateClusterAzure  `json:"azure"`
	} `json:"properties" binding:"required"`
}

type UpdateClusterRequest struct {
	Cloud            string `json:"cloud" binding:"required"`
	UpdateProperties `json:"properties"`
}

type UpdateProperties struct {
	*UpdateClusterAmazon `json:"amazon"`
	*UpdateClusterAzure  `json:"azure"`
}

type ClusterSimple struct {
	gorm.Model
	Name             string `gorm:"unique"`
	Location         string
	NodeInstanceType string
	Cloud            string
	Amazon           AmazonClusterSimple
	Azure            AzureSimple
}

type ClusterRepresentation struct {
	Id                    uint   `json:"id"`
	Name                  string `json:"name"`
	CloudType             string `json:"cloud"`
	*AmazonRepresentation `json:"amazon"`
	*AzureRepresentation  `json:"azure"`
}

type AmazonRepresentation struct {
	Ip string `json:"ip"`
}

// String method prints formatted update request fields
func (r UpdateClusterRequest) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Cloud: %s, ", r.Cloud))
	if r.Cloud == Azure && r.UpdateClusterAzure != nil {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("Agent count: %d",
			r.UpdateClusterAzure.AgentCount))
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

// DeleteFromDb deletes cluster from database
func (cs *ClusterSimple) DeleteFromDb(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {

	utils.LogInfo(log, utils.TagDeleteCluster, "Delete from database")

	if err := db.Delete(&cs).Error; err != nil {
		// delete failed
		utils.LogWarn(log, utils.TagDeleteCluster, "Can't delete cluster from database!", err)
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

// TableName sets ClusterSimple's table name
func (ClusterSimple) TableName() string {
	return tableNameClusters
}

// String method prints formatted cluster fields
func (cs *ClusterSimple) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Id: %d, Creation date: %s, Cloud: %s, NodeInstanceType: %s, ", cs.ID, cs.CreatedAt, cs.Cloud, cs.NodeInstanceType))
	if cs.Cloud == Azure {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("Agent count: %d, Agent name: %s, Kubernetes version: %s",
			cs.Azure.AgentCount,
			cs.Azure.AgentName,
			cs.Azure.KubernetesVersion))
	} else if cs.Cloud == Amazon {
		// Write AWS Master
		buffer.WriteString(fmt.Sprintf("Master instance type: %s, Master image: %s",
			cs.Amazon.MasterInstanceType,
			cs.Amazon.MasterImage))
		// Write AWS Node
		buffer.WriteString(fmt.Sprintf("Spot price: %s, Min count: %d, Max count: %d, Node image: %s",
			cs.Amazon.NodeSpotPrice,
			cs.Amazon.NodeMinCount,
			cs.Amazon.NodeMaxCount,
			cs.Amazon.NodeImage))
	}

	return buffer.String()
}

func updateClusterInDb(c *gin.Context, db *gorm.DB, log *logrus.Logger, cluster ClusterSimple) bool {
	utils.LogInfo(log, utils.TagUpdateCluster, "Update cluster in database")
	if err := db.Model(&ClusterSimple{}).Update(&cluster).Error; err != nil {
		DbSaveFailed(c, log, err, cluster.Name)
		return false
	}
	return true
}

// UpdateClusterInCloud updates cluster in cloud
// The request's cloud field decided which type of cloud will be called
func (r *UpdateClusterRequest) UpdateClusterInCloud(c *gin.Context, db *gorm.DB, log *logrus.Logger, preCluster ClusterSimple) bool {

	switch r.Cloud {
	case Amazon:
		return r.UpdateClusterAmazonInCloud(c, db, log, preCluster)
	case Azure:
		return r.UpdateClusterAzureInCloud(c, db, log, preCluster)
	default:
		return false
	}

}

// The Validate method checks the request fields
func (r *UpdateClusterRequest) Validate(log *logrus.Logger, defaultValue ClusterSimple) (bool, string) {

	switch r.Cloud {
	case Amazon:
		// amazon validate
		return r.ValidateAmazonRequest(log, defaultValue)
	case Azure:
		// azure validate
		return r.ValidateAzureRequest(log, defaultValue)
	default:
		// not supported cloud type
		return false, "Not supported cloud type."
	}

}

// isUpdateEqualsWithStoredCluster compares x and y interfaces with deep equal
func isUpdateEqualsWithStoredCluster(x interface{}, y interface{}, log *logrus.Logger) (bool, string) {
	if reflect.DeepEqual(x, y) {
		msg := "There is no change in data"
		utils.LogInfo(log, utils.TagValidateUpdateCluster, msg)
		return false, msg
	}
	utils.LogInfo(log, utils.TagValidateUpdateCluster, "Different interfaces")
	return true, ""
}

// DbSaveFailed sends DB operation failed message back
func DbSaveFailed(c *gin.Context, log *logrus.Logger, err error, clusterName string) {
	log.Warning("Can't persist cluster into the database!", err)

	SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
		JsonKeyStatus:  http.StatusBadRequest,
		JsonKeyMessage: "Can't persist cluster into the database!",
		JsonKeyName:    clusterName,
		JsonKeyError:   err,
	})
}

// GetCluster from database
// If no field param was specified automatically use value as ID
// Else it will use field as query column name
func GetClusterFromDB(c *gin.Context, db *gorm.DB, log *logrus.Logger) (*ClusterSimple, error) {

	utils.LogInfo(log, utils.TagGetCluster, "Get cluster from database")

	var cluster ClusterSimple
	value := c.Param("id")
	field := c.DefaultQuery("field", "")
	if field == "" {
		field = "id"
	}
	utils.LogInfo(log, utils.TagGetCluster, "Cluster ID:", value)
	query := fmt.Sprintf("%s = ?", field)
	db.Where(query, value).First(&cluster)
	if cluster.ID == 0 {
		errorMsg := fmt.Sprintf("cluster not found: [%s]: %s", field, value)
		utils.LogInfo(log, utils.TagGetCluster, errorMsg)
		SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			JsonKeyStatus:  http.StatusNotFound,
			JsonKeyMessage: errorMsg,
		})
		return nil, errors.New(errorMsg)
	}
	return &cluster, nil

}

func GetClusterSimple(c *gin.Context, db *gorm.DB, log *logrus.Logger) (*ClusterSimple, error) {
	cl, err := GetClusterFromDB(c, db, log)
	if err != nil {
		return nil, err
	}
	return cl, nil
}

func (cs *ClusterSimple) DeleteCluster(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {

	clusterType := cs.Cloud
	utils.LogInfo(log, utils.TagDeleteCluster, "Cluster type is ", clusterType)

	switch clusterType {
	case Amazon:
		// create amazon cs
		return cs.DeleteAmazonCluster(c, db, log)
	case Azure:
		// delete azure cs
		return cs.DeleteAzureCluster(c, db, log)
	default:
		SendNotSupportedCloudResponse(c, log, utils.TagDeleteCluster)
		return false
	}

}

// SendNotSupportedCloudResponse sends Not-supported-cloud-type error message back
func SendNotSupportedCloudResponse(c *gin.Context, log *logrus.Logger, tag string) {
	msg := "Not supported cloud type. Please use one of the following: " + Amazon + ", " + Azure + "."
	utils.LogInfo(log, tag, msg)
	SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
		JsonKeyStatus:  http.StatusBadRequest,
		JsonKeyMessage: msg,
	})
}

func (cs *ClusterSimple) GetClusterRepresentation(db *gorm.DB, log *logrus.Logger) *ClusterRepresentation {

	cloudType := cs.Cloud
	utils.LogInfo(log, utils.TagGetCluster, "Cloud type is ", cloudType)

	switch cloudType {
	case Amazon:
		return cs.ReadClusterAmazon(log)
		break
	case Azure:
		db.Where(AzureSimple{ClusterSimpleId: cs.ID}).First(&cs.Azure)
		return cs.ReadClusterAzure(log)
		break
	default:
		utils.LogInfo(log, utils.TagGetCluster, "Not supported cloud type")
		break
	}
	return nil
}

func (cs *ClusterSimple) FetchClusterInfo(c *gin.Context, db *gorm.DB, log *logrus.Logger) {

	cloudType := cs.Cloud
	utils.LogInfo(log, utils.TagGetClusterInfo, "Cloud type is ", cloudType)

	switch cloudType {
	case Amazon:
		cs.GetClusterInfoAmazon(c, log)
		break
	case Azure:
		// set azure props
		db.Where(AzureSimple{ClusterSimpleId: cs.ID}).First(&cs.Azure)
		cs.GetClusterInfoAzure(c, log)
		break
	default:
		// wrong cloud type
		SendNotSupportedCloudResponse(c, log, utils.TagGetCluster)
		break
	}
}
