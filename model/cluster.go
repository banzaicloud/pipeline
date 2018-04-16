package model

import (
	"bytes"
	"fmt"

	"encoding/json"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

const unknown = "unknown"

//ClusterModel describes the common cluster model
type ClusterModel struct {
	gorm.Model
	Name             string `gorm:"unique"`
	Location         string
	NodeInstanceType string
	Cloud            string
	OrganizationId   uint
	SecretId         string
	Status           string
	Amazon           AmazonClusterModel
	Azure            AzureClusterModel
	Google           GoogleClusterModel
	Dummy            DummyClusterModel
	Kubernetes       KubernetesClusterModel
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
	KubernetesVersion string
	NodePools         []*AzureNodePoolModel `gorm:"foreignkey:ClusterModelId"`
}

type AzureNodePoolModel struct {
	ID               uint   `gorm:"primary_key"`
	ClusterModelId   uint   `gorm:"unique_index:idx_modelid_name"`
	Name             string `gorm:"unique_index:idx_modelid_name"`
	Count            int
	NodeInstanceType string
}

//GoogleNodePoolModel describes google node pools model of a cluster
type GoogleNodePoolModel struct {
	ID               uint   `gorm:"primary_key"`
	ClusterModelId   uint   `gorm:"unique_index:idx_modelid_name"`
	Name             string `gorm:"unique_index:idx_modelid_name"`
	NodeCount        int
	NodeInstanceType string
	ServiceAccount   string
	Delete           bool `gorm:"-"`
}

//GoogleClusterModel describes the google cluster model
type GoogleClusterModel struct {
	ClusterModelId uint `gorm:"primary_key"`
	Project        string
	MasterVersion  string
	NodeVersion    string
	NodePools      []*GoogleNodePoolModel `gorm:"foreignkey:ClusterModelId"`
}

type DummyClusterModel struct {
	ClusterModelId    uint `gorm:"primary_key"`
	KubernetesVersion string
	NodeCount         int
}

//KubernetesClusterModel describes the build your own cluster model
type KubernetesClusterModel struct {
	ClusterModelId uint              `gorm:"primary_key"`
	Metadata       map[string]string `gorm:"-"`
	MetadataRaw    []byte            `gorm:"meta_data"`
}

func (gn GoogleNodePoolModel) String() string {
	return fmt.Sprintf("(Name: %s, Instance type: %s, Node count: %d, Service account: %s)",
		gn.Name, gn.NodeInstanceType, gn.NodeCount, gn.ServiceAccount)
}

func (gc GoogleClusterModel) String() string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("Master version: %s, Node version: %s, Node pools: %s",
		gc.MasterVersion,
		gc.NodeVersion,
		gc.NodePools))

	return buffer.String()
}

// BeforeSave converts the metadata into a json string in case of Kubernetes
func (cs *ClusterModel) BeforeSave() error {
	log := logger.WithFields(logrus.Fields{"tag": "BeforeSave"})
	log.Info("Before save convert meta data")

	if cs.Cloud == constants.Kubernetes && cs.Kubernetes.MetadataRaw != nil && len(cs.Kubernetes.MetadataRaw) != 0 {
		if out, err := json.Marshal(cs.Kubernetes.Metadata); err != nil {
			log.Errorf("Error during convert map to json: %s", err.Error())
			return err
		} else {
			cs.Kubernetes.MetadataRaw = out
		}
	}

	return nil
}

// AfterFind converts metadata json string into map in case of Kubernetes and sets NodeInstanceType and/or Location field(s)
// to unknown if they are empty
func (cs *ClusterModel) AfterFind() error {

	log := logger.WithFields(logrus.Fields{"tag": "AfterFind"})
	log.Info("After find convert metadata")

	if len(cs.Location) == 0 {
		cs.Location = unknown
	}

	if len(cs.NodeInstanceType) == 0 {
		cs.NodeInstanceType = unknown
	}

	if cs.Cloud == constants.Kubernetes && cs.Kubernetes.MetadataRaw != nil && len(cs.Kubernetes.MetadataRaw) != 0 {
		if out, err := utils.ConvertJson2Map(cs.Kubernetes.MetadataRaw); err != nil {
			log.Errorf("Error during convert json to map: %s", err.Error())
			return err
		} else {
			cs.Kubernetes.Metadata = out
		}
	}

	return nil
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
		buffer.WriteString(fmt.Sprintf("NodePools: %v, Kubernetes version: %s",
			cs.Azure.NodePools,
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
		buffer.WriteString(fmt.Sprint(cs.Google))
	} else if cs.Cloud == constants.Dummy {
		buffer.WriteString(fmt.Sprintf("Node count: %d, kubernetes version: %s",
			cs.Dummy.NodeCount,
			cs.Dummy.KubernetesVersion))
	} else if cs.Cloud == constants.Kubernetes {
		buffer.WriteString(fmt.Sprintf("Metadata: %#v", cs.Kubernetes.Metadata))
	}

	return buffer.String()
}

// TableName sets AmazonClusterModel's table name
func (AmazonClusterModel) TableName() string {
	return constants.TableNameAmazonProperties
}

// TableName sets AzureClusterModel's table name
func (AzureClusterModel) TableName() string {
	return constants.TableNameAzureProperties
}

// TableName sets AzureNodePoolModel's table name
func (AzureNodePoolModel) TableName() string {
	return constants.TableNameAzureNodePools
}

// QueryCluster get's the clusters from the DB
func QueryCluster(filter map[string]interface{}) ([]ClusterModel, error) {
	var cluster []ClusterModel
	err := db.Where(filter).Find(&cluster).Error
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

//GetSimpleClusterWithId returns a simple cluster model
func GetSimpleClusterWithId(id uint) ClusterModel {
	return ClusterModel{Model: gorm.Model{ID: id}}
}

//TableName sets the GoogleClusterModel's table name
func (GoogleClusterModel) TableName() string {
	return constants.TableNameGoogleProperties
}

//TableName sets the GoogleNodePoolModel's table name
func (GoogleNodePoolModel) TableName() string {
	return constants.TableNameGoogleNodePools
}

//TableName sets the DummyClusterModel's table name
func (DummyClusterModel) TableName() string {
	return constants.TableNameDummyProperties
}

//TableName sets the KubernetesClusterModel's table name
func (KubernetesClusterModel) TableName() string {
	return constants.TableNameKubeProperties
}

func (googleClusterModel *GoogleClusterModel) AfterUpdate(scope *gorm.Scope) error {
	log := logger.WithFields(logrus.Fields{"tag": "AfterUpdate"})
	log.Info("Remove node pools marked for deletion")

	for _, nodePoolModel := range googleClusterModel.NodePools {
		if nodePoolModel.Delete {
			err := scope.DB().Delete(nodePoolModel).Error

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (cs *ClusterModel) UpdateStatus(status string) error {
	cs.Status = status
	return cs.Save()
}
