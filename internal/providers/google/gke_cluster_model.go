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

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/jinzhu/gorm"
)

// GKEClusterModel is the schema for the DB.
type GKEClusterModel struct {
	Cluster   cluster.ClusterModel `gorm:"foreignkey:ClusterID"`
	ClusterID uint                 `gorm:"index;not null"`

	MasterVersion string
	NodeVersion   string
	Region        string
	NodePools     []*GKENodePoolModel `gorm:"foreignkey:ClusterID"`
}

// TableName changes the default table name.
func (GKEClusterModel) TableName() string {
	return "google_gke_clusters"
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

	ClusterID uint `gorm:"unique_index:idx_cluster_id_name"`

	Name             string `gorm:"unique_index:idx_cluster_id_name"`
	Autoscaling      bool   `gorm:"default:false"`
	NodeMinCount     int
	NodeMaxCount     int
	NodeCount        int
	NodeInstanceType string
	Delete           bool `gorm:"-"`
}

// TableName changes the default table name.
func (GKENodePoolModel) TableName() string {
	return "google_gke_node_pools"
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
