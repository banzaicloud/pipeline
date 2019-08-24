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

// GKEProfile describes a Google cluster profile
type GKEProfile struct {
	DefaultModel
	Location      string                `gorm:"default:'us-central1-a'"`
	NodeVersion   string                `gorm:"default:'1.10'"`
	MasterVersion string                `gorm:"default:'1.10'"`
	NodePools     []*GKENodePoolProfile `gorm:"foreignkey:Name"`
	TtlMinutes    uint                  `gorm:"not null;default:0"`
}

// GKENodePoolProfile describes a Google cluster profile's nodepools
type GKENodePoolProfile struct {
	ID               uint                        `gorm:"primary_key"`
	Autoscaling      bool                        `gorm:"default:false"`
	MinCount         int                         `gorm:"default:1"`
	MaxCount         int                         `gorm:"default:2"`
	Count            int                         `gorm:"default:1"`
	NodeInstanceType string                      `gorm:"default:'n1-standard-1'"`
	Name             string                      `gorm:"unique_index:idx_gke_profile_node_pools_name_node_name"`
	NodeName         string                      `gorm:"unique_index:idx_gke_profile_node_pools_name_node_name"`
	Preemptible      bool                        `gorm:"default:false"`
	Labels           []*GKENodePoolLabelsProfile `gorm:"foreignkey:NodePoolProfileID"`
}

// GKENodePoolLabelsProfile stores labels for Google cluster profile's nodepools
type GKENodePoolLabelsProfile struct {
	ID                uint   `gorm:"primary_key"`
	Name              string `gorm:"unique_index:idx_gke_name_profile_node_pool_id"`
	Value             string
	NodePoolProfileID uint `gorm:"unique_index:idx_gke_name_profile_node_pool_id"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName overrides GKEProfile's table name
func (GKEProfile) TableName() string {
	return DefaultGKEProfileTableName
}

// TableName overrides GKENodePoolProfile's table name
func (GKENodePoolProfile) TableName() string {
	return DefaultGKENodePoolProfileTableName
}

// TableName overrides GKENodePoolLabelsProfile's table name
func (GKENodePoolLabelsProfile) TableName() string {
	return DefaultGKENodePoolProfileLabelsTableName
}
