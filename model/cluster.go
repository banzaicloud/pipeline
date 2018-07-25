package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/banzaicloud/pipeline/database"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	modelOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/jinzhu/gorm"
)

const unknown = "unknown"

//TableName constants
const (
	TableNameClusters             = "clusters"
	TableNameAmazonProperties     = "amazon_cluster_properties"
	TableNameAmazonNodePools      = "amazon_node_pools"
	TableNameAmazonEksProperties  = "amazon_eks_cluster_properties"
	TableNameAzureProperties      = "azure_cluster_properties"
	TableNameAzureNodePools       = "azure_node_pools"
	TableNameGoogleProperties     = "google_cluster_properties"
	TableNameGoogleNodePools      = "google_node_pools"
	TableNameDummyProperties      = "dummy_cluster_properties"
	TableNameKubernetesProperties = "kubernetes_cluster_properties"
)

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
	ConfigSecretId string
	SshSecretId    string
	Status         string
	Monitoring     bool
	Logging        bool
	StatusMessage  string `sql:"type:text;"`
	Amazon         AmazonClusterModel
	Azure          AzureClusterModel
	Eks            AmazonEksClusterModel
	Google         GoogleClusterModel
	Dummy          DummyClusterModel
	Kubernetes     KubernetesClusterModel
	Oracle         modelOracle.Cluster
	Applications   []Application `gorm:"foreignkey:ClusterID"`
	CreatedBy      uint
}

//AmazonClusterModel describes the amazon cluster model
type AmazonClusterModel struct {
	ClusterModelId     uint `gorm:"primary_key"`
	MasterInstanceType string
	MasterImage        string
	NodePools          []*AmazonNodePoolsModel `gorm:"foreignkey:ClusterModelId"`
}

//AmazonNodePoolsModel describes Amazon node groups model of a cluster
type AmazonNodePoolsModel struct {
	ID               uint `gorm:"primary_key"`
	CreatedAt        time.Time
	CreatedBy        uint
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

//AmazonEksClusterModel describes the amazon cluster model
type AmazonEksClusterModel struct {
	ClusterModelId uint                    `gorm:"primary_key"`
	Version        string                  //kubernetes "1.10"
	NodePools      []*AmazonNodePoolsModel `gorm:"foreignkey:ClusterModelId"`
	AccessKeyID    string
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
	ID               uint `gorm:"primary_key"`
	CreatedAt        time.Time
	CreatedBy        uint
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
	ID               uint `gorm:"primary_key"`
	CreatedAt        time.Time
	CreatedBy        uint
	ClusterModelId   uint   `gorm:"unique_index:idx_modelid_name"`
	Name             string `gorm:"unique_index:idx_modelid_name"`
	Autoscaling      bool   `gorm:"default:false"`
	NodeMinCount     int
	NodeMaxCount     int
	NodeCount        int
	NodeInstanceType string
	Delete           bool `gorm:"-"`
}

//GoogleClusterModel describes the google cluster model
type GoogleClusterModel struct {
	ClusterModelId uint `gorm:"primary_key"`
	MasterVersion  string
	NodeVersion    string
	Region         string
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
	return fmt.Sprintf("ID: %d, createdAt: %v, createdBy: %d, Name: %s, Autoscaling: %v, NodeMinCount: %d, NodeMaxCount: %d, NodeCount: %d",
		gn.ID, gn.CreatedAt, gn.CreatedBy, gn.Name, gn.Autoscaling, gn.NodeMinCount, gn.NodeMaxCount, gn.NodeCount)
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
	log.Info("Before save convert meta data")

	if cs.Cloud == pkgCluster.Kubernetes && cs.Kubernetes.MetadataRaw != nil && len(cs.Kubernetes.MetadataRaw) != 0 {
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

	log.Info("After find convert metadata")

	if len(cs.Location) == 0 {
		cs.Location = unknown
	}

	if cs.Cloud == pkgCluster.Kubernetes && cs.Kubernetes.MetadataRaw != nil && len(cs.Kubernetes.MetadataRaw) != 0 {
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
	db := database.GetDB()
	err := db.Save(&cs).Error
	if err != nil {
		return err
	}
	return nil
}

func (cs *ClusterModel) preDelete() {

	log.Info("Delete config secret")
	if err := secret.Store.Delete(cs.OrganizationId, cs.ConfigSecretId); err != nil {
		log.Warnf("Error during deleting config secret: %s", err.Error())
	}

	log.Info("Delete SSH secret")
	if err := secret.Store.Delete(cs.OrganizationId, cs.SshSecretId); err != nil {
		log.Warnf("Error during deleting config secret: %s", err.Error())
	}

}

//Delete cluster from DB
func (cs *ClusterModel) Delete() error {

	log.Info("Delete config secret")
	cs.preDelete()

	db := database.GetDB()
	return db.Delete(&cs).Error
}

// TableName sets ClusterModel's table name
func (ClusterModel) TableName() string {
	return TableNameClusters
}

// String method prints formatted cluster fields
func (cs *ClusterModel) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Id: %d, Creation date: %s, Cloud: %s, ", cs.ID, cs.CreatedAt, cs.Cloud))
	if cs.Cloud == pkgCluster.Azure {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("NodePools: %v, Kubernetes version: %s",
			cs.Azure.NodePools,
			cs.Azure.KubernetesVersion))
	} else if cs.Cloud == pkgCluster.Amazon {
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

	} else if cs.Cloud == pkgCluster.Google {
		buffer.WriteString(fmt.Sprint(cs.Google))
	} else if cs.Cloud == pkgCluster.Dummy {
		buffer.WriteString(fmt.Sprintf("Node count: %d, kubernetes version: %s",
			cs.Dummy.NodeCount,
			cs.Dummy.KubernetesVersion))
	} else if cs.Cloud == pkgCluster.Kubernetes {
		buffer.WriteString(fmt.Sprintf("Metadata: %#v", cs.Kubernetes.Metadata))
	}

	return buffer.String()
}

// TableName sets AmazonClusterModel's table name
func (AmazonClusterModel) TableName() string {
	return TableNameAmazonProperties
}

// TableName sets AmazonNodePoolsModel's table name
func (AmazonNodePoolsModel) TableName() string {
	return TableNameAmazonNodePools
}

// TableName sets AmazonEksClusterModel's table name
func (AmazonEksClusterModel) TableName() string {
	return TableNameAmazonEksProperties
}

// TableName sets AzureClusterModel's table name
func (AzureClusterModel) TableName() string {
	return TableNameAzureProperties
}

// TableName sets AzureNodePoolModel's table name
func (AzureNodePoolModel) TableName() string {
	return TableNameAzureNodePools
}

// QueryCluster get's the clusters from the DB
func QueryCluster(filter map[string]interface{}) ([]ClusterModel, error) {
	var cluster []ClusterModel
	err := database.GetDB().Where(filter).Find(&cluster).Error
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

//TableName sets the GoogleClusterModel's table name
func (GoogleClusterModel) TableName() string {
	return TableNameGoogleProperties
}

//TableName sets the GoogleNodePoolModel's table name
func (GoogleNodePoolModel) TableName() string {
	return TableNameGoogleNodePools
}

//TableName sets the DummyClusterModel's table name
func (DummyClusterModel) TableName() string {
	return TableNameDummyProperties
}

//TableName sets the KubernetesClusterModel's table name
func (KubernetesClusterModel) TableName() string {
	return TableNameKubernetesProperties
}

// AfterUpdate removes marked node pool(s)
func (gc *GoogleClusterModel) AfterUpdate(scope *gorm.Scope) error {
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

// UpdateConfigSecret updates the model's config secret id in database
func (cs *ClusterModel) UpdateConfigSecret(configSecretId string) error {
	cs.ConfigSecretId = configSecretId
	return cs.Save()
}

// UpdateSshSecret updates the model's ssh secret id in database
func (cs *ClusterModel) UpdateSshSecret(sshSecretId string) error {
	cs.SshSecretId = sshSecretId
	return cs.Save()
}

// AddSshKey generates and associates an SSH key with the cluster
func (cs *ClusterModel) AddSshKey() (string, error) {
	log.Info("Generate and store SSH key ")

	sshKey, err := secret.GenerateSSHKeyPair()
	if err != nil {
		log.Errorf("KeyGenerator failed reason: %s", err.Error())
		return "", err
	}

	secretId, err := secret.StoreSSHKeyPair(sshKey, cs.OrganizationId, cs.ID, cs.Name)
	if err != nil {
		log.Errorf("KeyStore failed reason: %s", err.Error())
		return "", err
	}

	return secretId, nil
}

// ReloadFromDatabase load cluster from DB
func (cs *ClusterModel) ReloadFromDatabase() error {
	return database.GetDB().Where(ClusterModel{
		ID:             cs.ID,
		OrganizationId: cs.OrganizationId,
	}, &cs).Error
}
