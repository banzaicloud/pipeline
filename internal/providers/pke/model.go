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
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
)

type Model struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	CreatedBy uint

	ClusterID uint `gorm:"foreignkey:ClusterID"`
}

// Migrate executes the table migrations for the provider.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		EC2PKEClusterModelLegacy{},
		CRI{},
		KubeADM{},
		Kubernetes{},
		Network{},
		NodePool{},
		Host{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"provider":    amazon.Provider,
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating provider tables")

	return db.AutoMigrate(tables...).Error
}

// EC2PKEClusterModelLegacy has to be deleted after dex_enabled is gone.
type EC2PKEClusterModelLegacy struct {
	ID                 uint                 `gorm:"primary_key"`
	Cluster            cluster.ClusterModel `gorm:"foreignkey:ClusterID"`
	ClusterID          uint
	MasterInstanceType string
	MasterImage        string
	CurrentWorkflowID  string
	DexEnabled         bool `gorm:"default:false;not null"`

	Network    Network    `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"network"`
	NodePools  NodePools  `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"nodepools"`
	Kubernetes Kubernetes `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"kubernetes"`
	KubeADM    KubeADM    `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"kubeadm"`
	CRI        CRI        `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID" yaml:"cri"`
}

// TableName changes the default table name.
func (EC2PKEClusterModelLegacy) TableName() string {
	return "amazon_ec2_clusters"
}
