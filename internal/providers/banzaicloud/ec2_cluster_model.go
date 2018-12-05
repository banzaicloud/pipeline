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

package banzaicloud

import "github.com/banzaicloud/pipeline/internal/cluster"

type EC2BanzaiCloudClusterModel struct {
	ID                 uint                 `gorm:"primary_key"`
	Cluster            cluster.ClusterModel `gorm:"foreignkey:ClusterID"`
	ClusterID          uint
	MasterInstanceType string
	MasterImage        string

	Network    Network    `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"network"`
	NodePools  NodePools  `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"nodepools"`
	Kubernetes Kubernetes `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"kubernetes"`
	KubeADM    KubeADM    `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"kubeadm"`
	CRI        CRI        `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"cri"`
}

// TableName changes the default table name.
func (EC2BanzaiCloudClusterModel) TableName() string {
	return "amazon_ec2_clusters"
}
