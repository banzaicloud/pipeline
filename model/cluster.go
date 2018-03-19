package model

import (
	"bytes"
	"fmt"

	"github.com/banzaicloud/banzai-types/constants"
	"github.com/jinzhu/gorm"
)

//ClusterModel describes the common cluster model
type ClusterModel struct {
	gorm.Model
	Name             string `gorm:"unique"`
	Location         string
	NodeInstanceType string
	Cloud            string
	OrganizationId   uint
	SecretId         string
	Amazon           AmazonClusterModel
	Azure            AzureClusterModel
	Google           GoogleClusterModel
}

//AmazonClusterModel describes the amazon cluster model
type AmazonClusterModel struct {
	ClusterModelId     uint `gorm:"primary_key"`
	NodeSpotPrice      string
	NodeMinCount       int
	NodeMaxCount       int
	NodeImage          string
	MasterInstanceType string
	MasterImage        string
}

//AzureClusterModel describes the azure cluster model
type AzureClusterModel struct {
	ClusterModelId    uint `gorm:"primary_key"`
	ResourceGroup     string
	AgentCount        int
	AgentName         string
	KubernetesVersion string
}

//GoogleClusterModel describes the google cluster model
type GoogleClusterModel struct {
	ClusterModelId uint `gorm:"primary_key"`
	Project        string
	MasterVersion  string
	NodeVersion    string
	NodeCount      int
	ServiceAccount string
}

//Save the cluster to DB
func (cs *ClusterModel) Save() error {
	db := GetDB()
	err := db.Save(&cs).Error
	if err != nil {
		return err
	}
	return nil
}

//Delete cluster from DB
func (cs *ClusterModel) Delete() error {
	db := GetDB()
	return db.Delete(&cs).Error
}

// TableName sets ClusterModel's table name
func (ClusterModel) TableName() string {
	return constants.TableNameClusters
}

// String method prints formatted cluster fields
func (cs *ClusterModel) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Id: %d, Creation date: %s, Cloud: %s, NodeInstanceType: %s, ", cs.ID, cs.CreatedAt, cs.Cloud, cs.NodeInstanceType))
	if cs.Cloud == constants.Azure {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("Agent count: %d, Agent name: %s, Kubernetes version: %s",
			cs.Azure.AgentCount,
			cs.Azure.AgentName,
			cs.Azure.KubernetesVersion))
	} else if cs.Cloud == constants.Amazon {
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
	} else if cs.Cloud == constants.Google {
		// Write GKE Master
		buffer.WriteString(fmt.Sprintf("Master version: %s",
			cs.Google.MasterVersion))
		// Write GKE Node
		buffer.WriteString(fmt.Sprintf("Node count: %d, Node version: %s",
			cs.Google.NodeCount,
			cs.Google.NodeVersion))
	}

	return buffer.String()
}

// TableName sets AmazonClusterModel's table name
func (AmazonClusterModel) TableName() string {
	return constants.TableNameAmazonProperties
}

// TableName sets AzureSimple's table name
func (AzureClusterModel) TableName() string {
	return constants.TableNameAzureProperties
}

//QueryCluster get's the cluster from the DB
func QueryCluster(filter map[string]interface{}) (*ClusterModel, error) {
	var cluster ClusterModel
	err := db.Where(filter).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

//GetSimpleClusterWithId returns a simple cluster model
func GetSimpleClusterWithId(id uint) ClusterModel {
	return ClusterModel{Model: gorm.Model{ID: id}}
}

//TableName sets the GoogleClusterModel's table name
func (GoogleClusterModel) TableName() string {
	return constants.TableNameGoogleProperties
}
