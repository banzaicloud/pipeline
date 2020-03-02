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

package model

import (
	"time"

	"github.com/jinzhu/gorm"
)

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

// TableName sets ACKClusterModel's table name
func (ACKClusterModel) TableName() string {
	return "alibaba_acsk_clusters"
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

// TableName sets ACKNodePoolModel's table name
func (ACKNodePoolModel) TableName() string {
	return "alibaba_acsk_node_pools"
}
