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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/global"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	modelOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
)

const unknown = "unknown"

// TableName constants
const (
	tableNameClusters             = "clusters"
	tableNameAlibabaProperties    = "alibaba_acsk_clusters"
	tableNameAlibabaNodePools     = "alibaba_acsk_node_pools"
	tableNameAmazonNodePools      = "amazon_node_pools"
	tableNameAmazonEksProperties  = "amazon_eks_clusters"
	tableNameAzureProperties      = "azure_aks_clusters"
	tableNameAzureNodePools       = "azure_aks_node_pools"
	tableNameDummyProperties      = "dummy_clusters"
	tableNameKubernetesProperties = "kubernetes_clusters"
	tableNameEKSSubnets           = "amazon_eks_subnets"
)

// ClusterModel describes the common cluster model
// Note: this model is being moved to github.com/banzaicloud/pipeline/pkg/model.ClusterModel
type ClusterModel struct {
	ID             uint   `gorm:"primary_key"`
	UID            string `gorm:"unique_index:idx_clusters_uid"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time `gorm:"unique_index:idx_clusters_unique_id" sql:"index"`
	StartedAt      *time.Time
	Name           string `gorm:"unique_index:idx_clusters_unique_id"`
	Location       string
	Cloud          string
	Distribution   string
	OrganizationId uint `gorm:"unique_index:idx_clusters_unique_id"`
	SecretId       string
	ConfigSecretId string
	SshSecretId    string
	Status         string
	RbacEnabled    bool
	ScaleOptions   ScaleOptions           `gorm:"foreignkey:ClusterID"`
	StatusMessage  string                 `sql:"type:text;"`
	ACK            ACKClusterModel        `gorm:"foreignkey:ID"`
	AKS            AKSClusterModel        `gorm:"foreignkey:ID"`
	EKS            EKSClusterModel        `gorm:"foreignkey:ClusterID"`
	Dummy          DummyClusterModel      `gorm:"foreignkey:ID"`
	Kubernetes     KubernetesClusterModel `gorm:"foreignkey:ID"`
	OKE            modelOracle.Cluster
	CreatedBy      uint
}

// TableName sets ClusterModel's table name
func (ClusterModel) TableName() string {
	return tableNameClusters
}

// ScaleOptions describes scale options
type ScaleOptions struct {
	ID                  uint `gorm:"primary_key"`
	ClusterID           uint `gorm:"unique_index:idx_scale_options_cluster_id"`
	Enabled             bool
	DesiredCpu          float64
	DesiredMem          float64
	DesiredGpu          int
	OnDemandPct         int
	Excludes            string `sql:"type:text;"`
	KeepDesiredCapacity bool
}

// ACKNodePoolModel describes Alibaba Cloud CS node groups model of a cluster
type ACKNodePoolModel struct {
	ID                           uint `gorm:"primary_key"`
	CreatedAt                    time.Time
	CreatedBy                    uint
	ClusterID                    uint   `gorm:"unique_index:idx_ack_node_pools_cluster_id_name"`
	Name                         string `gorm:"unique_index:idx_ack_node_pools_cluster_id_name"`
	InstanceType                 string
	DeprecatedSystemDiskCategory string `gorm:"column:system_disk_category"`
	DeprecatedSystemDiskSize     int    `gorm:"column:system_disk_size"`
	DeprecatedImage              string `gorm:"column:image"`
	Count                        int
	MinCount                     int
	MaxCount                     int
	AsgID                        string
	ScalingConfigID              string
	Labels                       map[string]string `gorm:"-"`
	Delete                       bool              `gorm:"-"`
}

// ACKClusterModel describes the Alibaba Cloud CS cluster model
type ACKClusterModel struct {
	ID                       uint `gorm:"primary_key"`
	ProviderClusterID        string
	RegionID                 string
	ZoneID                   string
	MasterInstanceType       string
	MasterSystemDiskCategory string
	MasterSystemDiskSize     int
	SNATEntry                bool
	SSHFlags                 bool
	NodePools                []*ACKNodePoolModel `gorm:"foreignkey:ClusterID"`
	KubernetesVersion        string
	VSwitchID                string
}

// AmazonNodePoolsModel describes Amazon node groups model of a cluster
type AmazonNodePoolsModel struct {
	ID               uint `gorm:"primary_key"`
	CreatedAt        time.Time
	CreatedBy        uint
	ClusterID        uint   `gorm:"unique_index:idx_amazon_node_pools_cluster_id_name"`
	Name             string `gorm:"unique_index:idx_amazon_node_pools_cluster_id_name"`
	NodeSpotPrice    string
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeImage        string
	NodeInstanceType string
	Labels           map[string]string `gorm:"-"`
	Delete           bool              `gorm:"-"`
}

// EKSSubnetModel describes the model of subnets used for creating an EKS cluster
type EKSSubnetModel struct {
	ID               uint `gorm:"primary_key"`
	CreatedAt        time.Time
	EKSCluster       EKSClusterModel
	ClusterID        uint    `gorm:"index:idx_eks_subnets_cluster_id"`
	SubnetId         *string `gorm:"size:32"`
	Cidr             *string `gorm:"size:18"`
	AvailabilityZone *string `gorm:"size:25"`
}

// EKSClusterModel describes the EKS cluster model
type EKSClusterModel struct {
	ID      uint `gorm:"primary_key"`
	Version string

	ClusterID    uint                    `gorm:"unique_index:idx_eks_clusters_cluster_id"`
	NodePools    []*AmazonNodePoolsModel `gorm:"foreignkey:ClusterID"`
	VpcId        *string                 `gorm:"size:32"`
	VpcCidr      *string                 `gorm:"size:18"`
	RouteTableId *string                 `gorm:"size:32"`
	Subnets      []*EKSSubnetModel       `gorm:"foreignkey:ClusterID"`

	// IAM settings
	DefaultUser        bool
	ClusterRoleId      string
	NodeInstanceRoleId string

	LogTypes EKSLogTypes `sql:"type:json"`

	APIServerAccessPoints EKSAPIServerAccessPoints `sql:"type:json"`

	CurrentWorkflowID string

	SSHGenerated bool `gorm:"default:true"`
}

type EKSLogTypes = JSONStringArray

type EKSAPIServerAccessPoints = JSONStringArray

// JSONStringArray is a special type, that represents a JSON array of strings in SQL databases
type JSONStringArray []string

// Value implements the driver.Valuer interface
func (elt JSONStringArray) Value() (driver.Value, error) {
	return json.Marshal(elt)
}

// Scan implements the sql.Scanner interface
func (elt *JSONStringArray) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), elt)
}

// AKSClusterModel describes the aks cluster model
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
	ClusterID        uint   `gorm:"unique_index:idx_aks_node_pools_cluster_id_name"`
	Name             string `gorm:"unique_index:idx_aks_node_pools_cluster_id_name"`
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeInstanceType string
	VNetSubnetID     string
	Labels           map[string]string `gorm:"-"`
}

// DummyClusterModel describes the dummy cluster model
type DummyClusterModel struct {
	ID                uint `gorm:"primary_key"`
	KubernetesVersion string
	NodeCount         int
}

// KubernetesClusterModel describes the build your own cluster model
type KubernetesClusterModel struct {
	ID          uint              `gorm:"primary_key"`
	Metadata    map[string]string `gorm:"-"`
	MetadataRaw []byte            `gorm:"meta_data"`
}

// BeforeSave converts the metadata into a json string in case of Kubernetes
func (cs *KubernetesClusterModel) BeforeSave() (err error) {
	cs.MetadataRaw, err = json.Marshal(cs.Metadata)
	return
}

// AfterFind converts the metadata json string to a map in case of Kubernetes
func (cs *KubernetesClusterModel) AfterFind() error {
	if len(cs.MetadataRaw) != 0 {
		return json.Unmarshal(cs.MetadataRaw, &cs.Metadata)
	}
	return nil
}

func (cs *ClusterModel) BeforeCreate() (err error) {
	if cs.UID == "" {
		cs.UID = uuid.Must(uuid.NewV4()).String()
	}

	return
}

// AfterFind converts metadata json string into map in case of Kubernetes and sets NodeInstanceType and/or Location field(s)
// to unknown if they are empty
func (cs *ClusterModel) AfterFind() error {

	if len(cs.Location) == 0 {
		cs.Location = unknown
	}

	if cs.Distribution == "acsk" {
		// we renamed acsk distribution to ack
		cs.Distribution = pkgCluster.ACK
	}

	return nil
}

// Save the cluster to DB
func (cs *ClusterModel) Save() error {
	db := global.DB()
	err := db.Save(&cs).Error
	if err != nil {
		return err
	}
	return nil
}

// Delete cluster from DB
func (cs *ClusterModel) Delete() error {
	db := global.DB()
	return db.Delete(&cs).Error
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

// TableName sets ACKClusterModel's table name
func (ACKClusterModel) TableName() string {
	return tableNameAlibabaProperties
}

// TableName sets ACKNodePoolModel's table name
func (ACKNodePoolModel) TableName() string {
	return tableNameAlibabaNodePools
}

// TableName sets AmazonNodePoolsModel's table name
func (AmazonNodePoolsModel) TableName() string {
	return tableNameAmazonNodePools
}

// TableName sets EKSClusterModel's table name
func (EKSClusterModel) TableName() string {
	return tableNameAmazonEksProperties
}

// SetCurrentWorkflowID sets currentWorkflowID
func (cm *EKSClusterModel) SetCurrentWorkflowID(workflowID string) error {
	cm.CurrentWorkflowID = workflowID
	fields := map[string]interface{}{
		"currentWorkflowID": cm.CurrentWorkflowID,
	}

	db := global.DB()
	err := db.Model(&cm).Updates(fields).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to update currentWorkflowID for EKS cluster", "cluster_id", cm.ClusterID)
	}
	return nil
}

func (cm *EKSClusterModel) PersistSSHGenerate(sshGenerated bool) error {
	cm.SSHGenerated = sshGenerated
	fields := map[string]interface{}{
		"sshGenerated": cm.SSHGenerated,
	}

	db := global.DB()
	err := db.Model(&cm).Updates(fields).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to update sshGenerated field for EKS cluster", "cluster_id", cm.ClusterID)
	}
	return nil
}

func (cm EKSClusterModel) IsSSHGenerated() bool {
	return cm.SSHGenerated
}

// TableName sets database table name for EKSSubnetModel
func (EKSSubnetModel) TableName() string {
	return tableNameEKSSubnets
}

// TableName sets AzureClusterModel's table name
func (AKSClusterModel) TableName() string {
	return tableNameAzureProperties
}

// TableName sets AzureNodePoolModel's table name
func (AKSNodePoolModel) TableName() string {
	return tableNameAzureNodePools
}

// TableName sets the DummyClusterModel's table name
func (DummyClusterModel) TableName() string {
	return tableNameDummyProperties
}

// TableName sets the KubernetesClusterModel's table name
func (KubernetesClusterModel) TableName() string {
	return tableNameKubernetesProperties
}

// AfterUpdate removes marked node pool(s)
func (cm *EKSClusterModel) AfterUpdate(tx *gorm.DB) error {
	log.WithField("clusterId", cm.ClusterID).Debug("remove node pools marked for deletion")

	for _, nodePoolModel := range cm.NodePools {
		if nodePoolModel.Delete {
			err := tx.Model(cm).Association("NodePools").Delete(nodePoolModel).Error
			if err != nil {
				return err
			}

			err = tx.Delete(nodePoolModel).Error
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// AfterUpdate removes marked node pool(s)
func (a *ACKClusterModel) AfterUpdate(scope *gorm.Scope) error {
	log.Debug("Remove node pools marked for deletion")

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
	if cs.Status == status && cs.StatusMessage == statusMessage {
		return nil
	}

	if cs.ID != 0 {
		// Record status change to history before modifying the actual status.
		// If setting/saving the actual status doesn't succeed somehow, at least we can reconstruct it from history (i.e. event sourcing).
		statusHistory := StatusHistoryModel{
			ClusterID:   cs.ID,
			ClusterName: cs.Name,

			FromStatus:        cs.Status,
			FromStatusMessage: cs.StatusMessage,
			ToStatus:          status,
			ToStatusMessage:   statusMessage,
		}

		if err := global.DB().Save(&statusHistory).Error; err != nil {
			return errors.Wrap(err, "failed to record cluster status change to history")
		}
	}

	if cs.Status == pkgCluster.Creating && (cs.Status == pkgCluster.Running || cs.Status == pkgCluster.Warning) {
		now := time.Now()
		cs.StartedAt = &now
	}
	cs.Status = status
	cs.StatusMessage = statusMessage

	if err := cs.Save(); err != nil {
		return errors.Wrap(err, "failed to update cluster status")
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
