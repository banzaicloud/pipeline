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

package azureadapter

import (
	"time"
)

// AKSClusterModel describes the aks cluster model
type AKSClusterModel struct {
	ID                uint `gorm:"primary_key"`
	ResourceGroup     string
	KubernetesVersion string
	NodePools         []*AKSNodePoolModel `gorm:"foreignkey:ClusterID"`
}

// TableName sets AzureClusterModel's table name
func (AKSClusterModel) TableName() string {
	return "azure_aks_clusters"
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

// TableName sets AzureNodePoolModel's table name
func (AKSNodePoolModel) TableName() string {
	return "azure_aks_node_pools"
}
