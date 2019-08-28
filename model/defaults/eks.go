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

// EKSProfile describes an Amazon EKS cluster profile
type EKSProfile struct {
	DefaultModel
	Region     string                `gorm:"default:'us-west-2'"`
	Version    string                `gorm:"default:'1.10'"`
	NodePools  []*EKSNodePoolProfile `gorm:"foreignkey:Name"`
	TtlMinutes uint                  `gorm:"not null;default:0"`
}

// EKSNodePoolProfile describes an EKS cluster profile's nodepools
type EKSNodePoolProfile struct {
	AmazonNodePoolProfileBaseFields
	Image  string                      `gorm:"default:'ami-0a54c984b9f908c81'"`
	Labels []*EKSNodePoolLabelsProfile `gorm:"foreignkey:NodePoolProfileID"`
}

// EKSNodePoolLabelsProfile describe the labels of a nodepool
// of an EKS cluster profile
type EKSNodePoolLabelsProfile struct {
	ID                uint   `gorm:"primary_key"`
	Name              string `gorm:"unique_index:idx_eks_profile_node_pool_labels_id_name"`
	Value             string
	NodePoolProfileID uint `gorm:"unique_index:idx_eks_profile_node_pool_labels_id_name"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName overrides EKSProfile's table name
func (EKSProfile) TableName() string {
	return DefaultEKSProfileTableName
}

// TableName overrides EKSNodePoolProfile's table name
func (EKSNodePoolProfile) TableName() string {
	return DefaultEKSNodePoolProfileTableName
}

// TableName override the EKSNodePoolLabelsProfile's table name
func (EKSNodePoolLabelsProfile) TableName() string {
	return DefaultEKSNodePoolLabelsProfileTableName
}
