package model

import (
	"bytes"
	"fmt"

	"encoding/json"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"time"
)

const unknown = "unknown"

//ClusterModel describes the common cluster model
type ClusterModel struct {
	ID             uint `gorm:"primary_key"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time `gorm:"unique_index:idx_unique_id" sql:"index"`
	Name           string     `gorm:"unique_index:idx_unique_id"`
	Location       string
	Cloud          string
	OrganizationId uint `gorm:"unique_index:idx_unique_id"`
	SecretId       string
	Status         string
	StatusMessage  string
	Amazon         AmazonClusterModel
	Azure          AzureClusterModel
	Google         GoogleClusterModel
	Dummy          DummyClusterModel
	Kubernetes     KubernetesClusterModel
}

//AmazonClusterModel describes the amazon cluster model
type AmazonClusterModel struct {
	ClusterModelId     uint `gorm:"primary_key"`
	MasterInstanceType string
	MasterImage        string
	NodePools          []*AmazonNodePoolsModel `gorm:"foreignkey:ClusterModelId"`
	SshSecretID        string
}

//AmazonNodePoolsModel describes Amazon node groups model of a cluster
type AmazonNodePoolsModel struct {
	ID               uint   `gorm:"primary_key"`
	ClusterModelId   uint   `gorm:"unique_index:idx_modelid_name"`
	Name             string `gorm:"unique_index:idx_modelid_name"`
	NodeSpotPrice    string
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeImage        string
	NodeInstanceType string
	Delete           bool `gorm:"-"`
}

//AzureClusterModel describes the azure cluster model
type AzureClusterModel struct {
	ClusterModelId    uint `gorm:"primary_key"`
	ResourceGroup     string
	KubernetesVersion string
	NodePools         []*AzureNodePoolModel `gorm:"foreignkey:ClusterModelId"`
}

// AzureNodePoolModel describes azure node pools model of a cluster
type AzureNodePoolModel struct {
	ID               uint   `gorm:"primary_key"`
	ClusterModelId   uint   `gorm:"unique_index:idx_modelid_name"`
	Name             string `gorm:"unique_index:idx_modelid_name"`
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeInstanceType string
}

//GoogleNodePoolModel describes google node pools model of a cluster
type GoogleNodePoolModel struct {
	ID               uint   `gorm:"primary_key"`
	ClusterModelId   uint   `gorm:"unique_index:idx_modelid_name"`
	Name             string `gorm:"unique_index:idx_modelid_name"`
	Autoscaling      bool   `gorm:"default:false"`
	NodeMinCount     int
	NodeMaxCount     int
	NodeCount        int
	NodeInstanceType string
	ServiceAccount   string
	Delete           bool `gorm:"-"`
}

//GoogleClusterModel describes the google cluster model
type GoogleClusterModel struct {
	ClusterModelId uint `gorm:"primary_key"`
	MasterVersion  string
	NodeVersion    string
	NodePools      []*GoogleNodePoolModel `gorm:"foreignkey:ClusterModelId"`
}

// DummyClusterModel describes the dummy cluster model
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
		out, err := json.Marshal(cs.Kubernetes.Metadata)
		if err != nil {
			log.Errorf("Error during convert map to json: %s", err.Error())
			return err
		}
		cs.Kubernetes.MetadataRaw = out
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

	if cs.Cloud == constants.Kubernetes && cs.Kubernetes.MetadataRaw != nil && len(cs.Kubernetes.MetadataRaw) != 0 {
		out, err := utils.ConvertJson2Map(cs.Kubernetes.MetadataRaw)
		if err != nil {
			log.Errorf("Error during convert json to map: %s", err.Error())
			return err
		}
		cs.Kubernetes.Metadata = out
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
	buffer.WriteString(fmt.Sprintf("Id: %d, Creation date: %s, Cloud: %s, ", cs.ID, cs.CreatedAt, cs.Cloud))
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
		for _, nodePool := range cs.Amazon.NodePools {
			buffer.WriteString(fmt.Sprintf("NodePool Name: %s, InstanceType: %s, Spot price: %s, Min count: %d, Max count: %d, Node image: %s",
				nodePool.Name,
				nodePool.NodeInstanceType,
				nodePool.NodeSpotPrice,
				nodePool.NodeMinCount,
				nodePool.NodeMaxCount,
				nodePool.NodeImage))
		}

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

// TableName sets AmazonNodePoolsModel's table name
func (AmazonNodePoolsModel) TableName() string {
	return constants.TableNameAmazonNodePools
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
	return ClusterModel{ID: id}
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
	return constants.TableNameKubernetesProperties
}

// AfterUpdate removes marked node pool(s)
func (gc *GoogleClusterModel) AfterUpdate(scope *gorm.Scope) error {
	log := logger.WithFields(logrus.Fields{"tag": "AfterUpdate"})
	log.Info("Remove node pools marked for deletion")

	for _, nodePoolModel := range gc.NodePools {
		if nodePoolModel.Delete {
			err := scope.DB().Delete(nodePoolModel).Error

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// AfterUpdate removes marked node pool(s)
func (a *AmazonClusterModel) AfterUpdate(scope *gorm.Scope) error {
	log := logger.WithFields(logrus.Fields{"tag": "AfterUpdate"})
	log.Info("Remove node pools marked for deletion")

	for _, nodePoolModel := range a.NodePools {
		if nodePoolModel.Delete {
			err := scope.DB().Delete(nodePoolModel).Error

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// UpdateStatus updates the model's status and status message in database
func (cs *ClusterModel) UpdateStatus(status, statusMessage string) error {
	cs.Status = status
	cs.StatusMessage = statusMessage
	return cs.Save()
}
