package database

import (
	"github.com/jinzhu/gorm"
	"github.com/banzaicloud/banzai-types/constants"
	"bytes"
	"fmt"
	"github.com/banzaicloud/banzai-types/database"
)

type ClusterSimple struct {
	gorm.Model
	Name             string `gorm:"unique"`
	Location         string
	NodeInstanceType string
	Cloud            string
	Amazon           AmazonClusterSimple
	Azure            AzureClusterSimple
}

type AmazonClusterSimple struct {
	ClusterSimpleId    uint `gorm:"primary_key"`
	NodeSpotPrice      string
	NodeMinCount       int
	NodeMaxCount       int
	NodeImage          string
	MasterInstanceType string
	MasterImage        string
}

type AzureClusterSimple struct {
	ClusterSimpleId   uint `gorm:"primary_key"`
	ResourceGroup     string
	AgentCount        int
	AgentName         string
	KubernetesVersion string
}

// TableName sets ClusterSimple's table name
func (ClusterSimple) TableName() string {
	return constants.TableNameClusters
}

// String method prints formatted cluster fields
func (cs *ClusterSimple) String() string {
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
	}

	return buffer.String()
}

// TableName sets AmazonClusterSimple's table name
func (AmazonClusterSimple) TableName() string {
	return constants.TableNameAmazonProperties
}

// TableName sets AzureSimple's table name
func (AzureClusterSimple) TableName() string {
	return constants.TableNameAzureProperties
}

func (cs *ClusterSimple) LoadClusterFromDatabase(clusterId uint, cloud string) {
	database.SelectFirstWhere(&cs, ClusterSimple{
		Model: gorm.Model{ID: clusterId},
		Cloud: cloud,
	})
}

func GetSimpleClusterWithId(id uint) ClusterSimple {
	return ClusterSimple{Model: gorm.Model{ID: id}}
}
