// Copyright © 2020 Banzai Cloud
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

package amazonadapter

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/internal/global"
)

// EKSClusterModel describes the EKS cluster model
type EKSClusterModel struct {
	ID      uint                      `gorm:"primary_key"`
	Cluster clustermodel.ClusterModel `gorm:"foreignkey:ClusterID"`

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

// TableName sets EKSClusterModel's table name
func (EKSClusterModel) TableName() string {
	return "amazon_eks_clusters"
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

func (cm EKSClusterModel) String() string {
	return fmt.Sprintf("%s, Master version: %s, Node pools: %s",
		cm.Cluster,
		cm.Version,
		cm.NodePools,
	)
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

// TableName sets AmazonNodePoolsModel's table name
func (AmazonNodePoolsModel) TableName() string {
	return "amazon_node_pools"
}

func (m AmazonNodePoolsModel) String() string {
	return fmt.Sprintf("NodePool Name: %s, Autoscaling: %v, InstanceType: %s, Spot price: %s, Min count: %d, Max count: %d, Count: %d, Node image: %s",
		m.Name,
		m.Autoscaling,
		m.NodeInstanceType,
		m.NodeSpotPrice,
		m.NodeMinCount,
		m.NodeMaxCount,
		m.Count,
		m.NodeImage,
	)
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

// TableName sets database table name for EKSSubnetModel
func (EKSSubnetModel) TableName() string {
	return "amazon_eks_subnets"
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
