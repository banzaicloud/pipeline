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

package defaults

import (
	"time"
)

// AKSProfile describes an Azure cluster profile
type AKSProfile struct {
	DefaultModel
	Location          string                `gorm:"default:'eastus'"`
	KubernetesVersion string                `gorm:"default:'1.9.2'"`
	NodePools         []*AKSNodePoolProfile `gorm:"foreignkey:Name"`
	TtlMinutes        uint                  `gorm:"not null;default:0"`
}

// AKSNodePoolProfile describes an Azure cluster profile's nodepools
type AKSNodePoolProfile struct {
	ID               uint                        `gorm:"primary_key"`
	Autoscaling      bool                        `gorm:"default:false"`
	MinCount         int                         `gorm:"default:1"`
	MaxCount         int                         `gorm:"default:2"`
	Count            int                         `gorm:"default:1"`
	NodeInstanceType string                      `gorm:"default:'Standard_D4_v2'"`
	Name             string                      `gorm:"unique_index:idx_aks_profile_node_pools_name_node_name"`
	NodeName         string                      `gorm:"unique_index:idx_aks_profile_node_pools_name_node_name"`
	Labels           []*AKSNodePoolLabelsProfile `gorm:"foreignkey:NodePoolProfileID"`
}

// AKSNodePoolLabelsProfile stores labels for Azure cluster profile's nodepools
type AKSNodePoolLabelsProfile struct {
	ID                uint   `gorm:"primary_key"`
	Name              string `gorm:"unique_index:idx_aks_profile_node_pool_labels_name_id"`
	Value             string
	NodePoolProfileID uint `gorm:"unique_index:idx_aks_profile_node_pool_labels_name_id"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName overrides AKSNodePoolProfile's table name
func (AKSNodePoolProfile) TableName() string {
	return DefaultAKSNodePoolProfileTableName
}

// TableName overrides AKSProfile's table name
func (AKSProfile) TableName() string {
	return DefaultAKSProfileTableName
}

// TableName overrides AKSProfile's table name
func (AKSNodePoolLabelsProfile) TableName() string {
	return DefaultAKSNodePoolProfileLabelsTableName
}
