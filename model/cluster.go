// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/banzaicloud/pipeline/config"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	modelOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const unknown = "unknown"

//TableName constants
const (
	TableNameClusters             = "clusters"
	TableNameAlibabaProperties    = "alibaba_acsk_clusters"
	TableNameAlibabaNodePools     = "alibaba_acsk_node_pools"
	TableNameAmazonNodePools      = "amazon_node_pools"
	TableNameAmazonEksProperties  = "amazon_eks_clusters"
	TableNameAzureProperties      = "azure_aks_clusters"
	TableNameAzureNodePools       = "azure_aks_node_pools"
	TableNameDummyProperties      = "dummy_clusters"
	TableNameKubernetesProperties = "kubernetes_clusters"
	TableNameEKSSubnets           = "amazon_eks_subnets"
)

//ClusterModel describes the common cluster model
// Note: this model is being moved to github.com/banzaicloud/pipeline/pkg/model.ClusterModel
type ClusterModel struct {
	ID             uint   `gorm:"primary_key"`
	UID            string `gorm:"unique_index:idx_uid"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time `gorm:"unique_index:idx_unique_id" sql:"index"`
	Name           string     `gorm:"unique_index:idx_unique_id"`
	Location       string
	Cloud          string
	Distribution   string
	OrganizationId uint `gorm:"unique_index:idx_unique_id"`
	SecretId       string
	ConfigSecretId string
	SshSecretId    string
	Status         string
	RbacEnabled    bool
	Monitoring     bool
	Logging        bool
	ServiceMesh    bool
	ScaleOptions   ScaleOptions `gorm:"foreignkey:ClusterID"`
	SecurityScan   bool
	StatusMessage  string                 `sql:"type:text;"`
	ACSK           ACSKClusterModel       `gorm:"foreignkey:ID"`
	AKS            AKSClusterModel        `gorm:"foreignkey:ID"`
	EKS            EKSClusterModel        `gorm:"foreignkey:ClusterID"`
	Dummy          DummyClusterModel      `gorm:"foreignkey:ID"`
	Kubernetes     KubernetesClusterModel `gorm:"foreignkey:ID"`
	OKE            modelOracle.Cluster
	CreatedBy      uint
}

// ScaleOptions describes scale options
type ScaleOptions struct {
	ID                  uint `gorm:"primary_key"`
	ClusterID           uint `gorm:"unique_index:ux_cluster_id"`
	Enabled             bool
	DesiredCpu          float64
	DesiredMem          float64
	DesiredGpu          int
	OnDemandPct         int
	Excludes            string `sql:"type:text;"`
	KeepDesiredCapacity bool
}

// ACSKNodePoolModel describes Alibaba Cloud CS node groups model of a cluster
type ACSKNodePoolModel struct {
	ID                           uint `gorm:"primary_key"`
	CreatedAt                    time.Time
	CreatedBy                    uint
	ClusterID                    uint   `gorm:"unique_index:idx_cluster_id_name"`
	Name                         string `gorm:"unique_index:idx_cluster_id_name"`
	InstanceType                 string
	DeprecatedSystemDiskCategory string `gorm:"column:system_disk_category"`
	DeprecatedSystemDiskSize     int    `gorm:"column:system_disk_size"`
	DeprecatedImage              string `gorm:"column:image"`
	Count                        int
	MinCount                     int
	MaxCount                     int
	AsgID                        string
	ScalingConfigID              string
	Delete                       bool `gorm:"-"`
}

// ACSKClusterModel describes the Alibaba Cloud CS cluster model
type ACSKClusterModel struct {
	ID                       uint `gorm:"primary_key"`
	ProviderClusterID        string
	RegionID                 string
	ZoneID                   string
	MasterInstanceType       string
	MasterSystemDiskCategory string
	MasterSystemDiskSize     int
	SNATEntry                bool
	SSHFlags                 bool
	NodePools                []*ACSKNodePoolModel `gorm:"foreignkey:ClusterID"`
	KubernetesVersion        string
}

//AmazonNodePoolsModel describes Amazon node groups model of a cluster
type AmazonNodePoolsModel struct {
	ID               uint `gorm:"primary_key"`
	CreatedAt        time.Time
	CreatedBy        uint
	ClusterID        uint   `gorm:"unique_index:idx_cluster_id_name"`
	Name             string `gorm:"unique_index:idx_cluster_id_name"`
	NodeSpotPrice    string
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeImage        string
	NodeInstanceType string
	Delete           bool `gorm:"-"`
}

// EKSSubnetModel describes the model of subnets used for creating an EKS cluster
type EKSSubnetModel struct {
	ID         uint `gorm:"primary_key"`
	CreatedAt  time.Time
	EKSCluster EKSClusterModel
	ClusterID  uint    `gorm:"index:idx_cluster_id"`
	SubnetId   *string `gorm:"size:32"`
	Cidr       *string `gorm:"size:18"`
}

//EKSClusterModel describes the EKS cluster model
type EKSClusterModel struct {
	ID        uint `gorm:"primary_key"`
	ClusterID uint `gorm:"unique_index:ux_cluster_id"`

	Version      string
	NodePools    []*AmazonNodePoolsModel `gorm:"foreignkey:ClusterID"`
	VpcId        *string                 `gorm:"size:32"`
	VpcCidr      *string                 `gorm:"size:18"`
	RouteTableId *string                 `gorm:"size:32"`
	Subnets      []*EKSSubnetModel       `gorm:"foreignkey:ClusterID"`
}

//AKSClusterModel describes the aks cluster model
type AKSClusterModel struct {
	ID                uint `gorm:"primary_key"`
	ResourceGroup     string
	KubernetesVersion string
	NodePools         []*AKSNodePoolModel `gorm:"foreignkey:ClusterID"`
}

// AKSNodePoolModel describes AKS node pools model of a cluster
type AKSNodePoolModel struct {
	ID               uint `gorm:"primary_key"`
	CreatedAt        time.Time
	CreatedBy        uint
	ClusterID        uint   `gorm:"unique_index:idx_cluster_id_name"`
	Name             string `gorm:"unique_index:idx_cluster_id_name"`
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeInstanceType string
}

// DummyClusterModel describes the dummy cluster model
type DummyClusterModel struct {
	ID                uint `gorm:"primary_key"`
	KubernetesVersion string
	NodeCount         int
}

//KubernetesClusterModel describes the build your own cluster model
type KubernetesClusterModel struct {
	ID          uint              `gorm:"primary_key"`
	Metadata    map[string]string `gorm:"-"`
	MetadataRaw []byte            `gorm:"meta_data"`
}

func (cs *ClusterModel) BeforeCreate() (err error) {
	if cs.UID == "" {
		cs.UID = uuid.NewV4().String()
	}

	return
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
	db := config.DB()
	err := db.Save(&cs).Error
	if err != nil {
		return err
	}
	return nil
}

//Delete cluster from DB
func (cs *ClusterModel) Delete() error {
	db := config.DB()
	return db.Delete(&cs).Error
}

// TableName sets ClusterModel's table name
func (ClusterModel) TableName() string {
	return TableNameClusters
}

// String method prints formatted cluster fields
func (cs *ClusterModel) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Id: %d, Creation date: %s, Cloud: %s, Distribution: %s, ", cs.ID, cs.CreatedAt, cs.Cloud, cs.Distribution))

	switch cs.Distribution {
	case pkgCluster.EKS:
		// Write EKS Master
		buffer.WriteString(fmt.Sprintf("Master version: %s",
			cs.EKS.Version))

		// Write EKS Node
		for _, nodePool := range cs.EKS.NodePools {
			buffer.WriteString(fmt.Sprintf("NodePool Name: %s, Autoscaling: %v, InstanceType: %s, Spot price: %s, Min count: %d, Max count: %d, Count: %d, Node image: %s",
				nodePool.Name,
				nodePool.Autoscaling,
				nodePool.NodeInstanceType,
				nodePool.NodeSpotPrice,
				nodePool.NodeMinCount,
				nodePool.NodeMaxCount,
				nodePool.Count,
				nodePool.NodeImage))
		}
	case pkgCluster.AKS:
		// Write AKS
		buffer.WriteString(fmt.Sprintf("NodePools: %v, Kubernetes version: %s",
			cs.AKS.NodePools,
			cs.AKS.KubernetesVersion))
	case pkgCluster.Dummy:
		buffer.WriteString(fmt.Sprintf("Node count: %d, kubernetes version: %s",
			cs.Dummy.NodeCount,
			cs.Dummy.KubernetesVersion))
	case pkgCluster.Kubernetes:
		buffer.WriteString(fmt.Sprintf("Metadata: %#v", cs.Kubernetes.Metadata))
	}

	return buffer.String()
}

// TableName sets ACSKClusterModel's table name
func (ACSKClusterModel) TableName() string {
	return TableNameAlibabaProperties
}

// TableName sets ACSKNodePoolModel's table name
func (ACSKNodePoolModel) TableName() string {
	return TableNameAlibabaNodePools
}

// TableName sets AmazonNodePoolsModel's table name
func (AmazonNodePoolsModel) TableName() string {
	return TableNameAmazonNodePools
}

// TableName sets EKSClusterModel's table name
func (EKSClusterModel) TableName() string {
	return TableNameAmazonEksProperties
}

// TableName sets database table name for EKSSubnetModel
func (EKSSubnetModel) TableName() string {
	return TableNameEKSSubnets
}

// TableName sets AzureClusterModel's table name
func (AKSClusterModel) TableName() string {
	return TableNameAzureProperties
}

// TableName sets AzureNodePoolModel's table name
func (AKSNodePoolModel) TableName() string {
	return TableNameAzureNodePools
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
func (a *EKSClusterModel) AfterUpdate(scope *gorm.Scope) error {
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

// AfterUpdate removes marked node pool(s)
func (a *ACSKClusterModel) AfterUpdate(scope *gorm.Scope) error {
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
	originalStatus := cs.Status
	originalStatusMessage := cs.StatusMessage

	cs.Status = status
	cs.StatusMessage = statusMessage
	err := cs.Save()
	if err != nil {
		return errors.Wrap(err, "failed to update cluster status")
	}

	if cs.Status != status {
		statusHistory := &StatusHistoryModel{
			ClusterID:   cs.ID,
			ClusterName: cs.Name,

			FromStatus:        originalStatus,
			FromStatusMessage: originalStatusMessage,
			ToStatus:          status,
			ToStatusMessage:   statusMessage,
		}

		db := config.DB()

		err := db.Save(&statusHistory).Error
		if err != nil {
			return errors.Wrap(err, "failed to update cluster status history")
		}
	}

	return nil
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
