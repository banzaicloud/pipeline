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

package google

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

// TableName constants
const (
	gkeClusterModelTableName  = "google_gke_clusters"
	gkeNodePoolModelTableName = "google_gke_node_pools"
	gkeNodePoolLabelTableName = "google_gke_node_pool_labels"
)

// GKEClusterModel is the schema for the DB.
type GKEClusterModel struct {
	ID        uint                 `gorm:"primary_key"`
	Cluster   cluster.ClusterModel `gorm:"foreignkey:ClusterID"`
	ClusterID uint                 `gorm:"unique_index:idx_gke_cluster_id"`

	MasterVersion string
	NodeVersion   string
	Region        string
	NodePools     []*GKENodePoolModel `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID"`
	ProjectId     string
	Vpc           string `gorm:"size:64"`
	Subnet        string `gorm:"size:64"`
}

// TableName changes the default table name.
func (GKEClusterModel) TableName() string {
	return gkeClusterModelTableName
}

// BeforeCreate sets some initial values for the cluster.
func (m *GKEClusterModel) BeforeCreate() error {
	m.Cluster.Cloud = Provider
	m.Cluster.Distribution = ClusterDistributionGKE

	return nil
}

// AfterUpdate removes node pool(s) marked for deletion.
func (m *GKEClusterModel) AfterUpdate(scope *gorm.Scope) error {
	for _, nodePoolModel := range m.NodePools {
		if nodePoolModel.Delete {
			// TODO: use transaction?
			err := scope.DB().Delete(nodePoolModel).Error

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// BeforeDelete deletes all nodepools that belongs to GKEClusterModel
func (m *GKEClusterModel) BeforeDelete(tx *gorm.DB) error {
	for _, nodePool := range m.NodePools {
		if err := tx.Delete(nodePool).Error; err != nil {
			return err
		}
	}

	return nil
}

// AfterDelete deletes Cluster that belongs to GKEClusterModel
func (m *GKEClusterModel) AfterDelete(tx *gorm.DB) error {
	if err := tx.Delete(m.Cluster).Error; err != nil {
		return err
	}

	return nil
}

func (m GKEClusterModel) String() string {
	return fmt.Sprintf("%s, Master version: %s, Node version: %s, Node pools: %s",
		m.Cluster,
		m.MasterVersion,
		m.NodeVersion,
		m.NodePools,
	)
}

// GKENodePoolModel is the schema for the DB.
type GKENodePoolModel struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	CreatedBy uint

	ClusterID uint   `gorm:"unique_index:idx_gke_np_cluster_id_name"`
	Name      string `gorm:"unique_index:idx_gke_np_cluster_id_name"`

	Autoscaling      bool `gorm:"default:false"`
	Preemptible      bool `gorm:"default:false"`
	NodeMinCount     int
	NodeMaxCount     int
	NodeCount        int
	NodeInstanceType string
	Labels           map[string]string `gorm:"-"`
	Delete           bool              `gorm:"-"`
}

// TableName changes the default table name.
func (GKENodePoolModel) TableName() string {
	return gkeNodePoolModelTableName
}

// BeforeDelete deletes all nodepoollabels that belongs to GKENodePoolModel
func (m *GKENodePoolModel) BeforeDelete(tx *gorm.DB) error {
	for _, label := range m.Labels {
		if err := tx.Delete(label).Error; err != nil {
			return err
		}
	}

	return nil
}

func (m GKENodePoolModel) String() string {
	return fmt.Sprintf(
		"ID: %d, createdAt: %v, createdBy: %d, Name: %s, Autoscaling: %v, NodeMinCount: %d, NodeMaxCount: %d, NodeCount: %d",
		m.ID,
		m.CreatedAt,
		m.CreatedBy,
		m.Name,
		m.Autoscaling,
		m.NodeMinCount,
		m.NodeMaxCount,
		m.NodeCount,
	)
}

// GKENodePoolLabelModel stores labels for node pools
type GKENodePoolLabelModel struct {
	ID         uint   `gorm:"primary_key"`
	Name       string `gorm:"unique_index:idx_gke_node_pool_labels_id_name"`
	Value      string
	NodePoolID uint `gorm:"unique_index:idx_gke_node_pool_labels_id_name"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TableName changes the default table name.
func (GKENodePoolLabelModel) TableName() string {
	return gkeNodePoolLabelTableName
}

func (m GKENodePoolLabelModel) String() string {
	return fmt.Sprintf(
		"ID: %d, Name: %s, Value: %s, NodePoolID: %d, createdAt: %v, UpdatedAt: %v",
		m.ID,
		m.Name,
		m.Value,
		m.NodePoolID,
		m.CreatedAt,
		m.UpdatedAt,
	)
}
