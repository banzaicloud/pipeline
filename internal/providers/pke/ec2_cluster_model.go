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

package pke

import (
	"emperror.dev/emperror"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

type EC2PKEClusterModel struct {
	ID                 uint                 `gorm:"primary_key"`
	Cluster            cluster.ClusterModel `gorm:"foreignkey:ClusterID"`
	ClusterID          uint
	MasterInstanceType string
	MasterImage        string
	CurrentWorkflowID  string

	Network    Network    `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"network"`
	NodePools  NodePools  `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"nodepools"`
	Kubernetes Kubernetes `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"kubernetes"`
	KubeADM    KubeADM    `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"kubeadm"`
	CRI        CRI        `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"cri"`
}

// TableName changes the default table name.
func (EC2PKEClusterModel) TableName() string {
	return "amazon_ec2_clusters"
}

// BeforeDelete callback / hook to delete related entries from the database
func (m *EC2PKEClusterModel) BeforeDelete(tx *gorm.DB) error {
	var e error

	if e = tx.Delete(m.Network).Error; e != nil {
		return emperror.WrapWith(e, "failed to delete network", "network", m.Network.ID)
	}

	if e = tx.Delete(m.CRI).Error; e != nil {
		return emperror.WrapWith(e, "failed to delete cri", "cri", m.CRI.ID)
	}

	for _, np := range m.NodePools {
		if e = tx.Where(Host{NodePoolID: np.NodePoolID}).Delete(np.Hosts).Error; e != nil {
			return emperror.WrapWith(e, "failed to delete nodepool hosts", "nodepool", np.Name)
		}
	}

	if e = tx.Where(NodePool{ClusterID: m.ClusterID}).Delete(m.NodePools).Error; e != nil {
		return emperror.WrapWith(e, "failed to delete nodepools", "nodepools", m.NodePools)
	}

	if e = tx.Delete(m.KubeADM).Error; e != nil {
		return emperror.WrapWith(e, "failed to delete KubeADM", "KubeADM", m.KubeADM.ID)
	}

	if e = tx.Delete(m.Kubernetes).Error; e != nil {
		return emperror.WrapWith(e, "failed to delete Kubernetes", "network", m.Kubernetes.ID)
	}

	if e = tx.Delete(m.Cluster).Error; e != nil {
		return emperror.WrapWith(e, "failed to delete cluster", "cluster", m.Cluster.ID)
	}

	return e
}
