// Copyright © 2019 Banzai Cloud
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

package deployment

import (
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

const clusterGroupDeploymentTableName = "clustergroup_deployments"
const clusterGroupDeploymentOverridesTableName = "clustergroup_deployment_target_clusters"

// TableName changes the default table name.
func (ClusterGroupDeploymentModel) TableName() string {
	return clusterGroupDeploymentTableName
}

// TableName changes the default table name.
func (TargetCluster) TableName() string {
	return clusterGroupDeploymentOverridesTableName
}

// ClusterGroupDeploymentModel describes a cluster group deployment
type ClusterGroupDeploymentModel struct {
	ID                    uint `gorm:"primary_key"`
	ClusterGroupID        uint `gorm:"unique_index:idx_unique_cid_rname"`
	CreatedAt             time.Time
	UpdatedAt             *time.Time
	DeploymentName        string
	DeploymentVersion     string
	DeploymentPackage     []byte
	DeploymentReleaseName string `gorm:"unique_index:idx_unique_cid_rname"`
	Description           string
	ChartName             string
	Namespace             string
	OrganizationName      string
	Values                []byte           `sql:"type:text;"`
	TargetClusters        []*TargetCluster `gorm:"foreignkey:ClusterGroupDeploymentID"`
}

// TargetCluster describes cluster specific values for a cluster group deployment
type TargetCluster struct {
	ID                       uint `gorm:"primary_key"`
	ClusterGroupDeploymentID uint `gorm:"unique_index:idx_unique_dep_cl"`
	ClusterID                uint `gorm:"unique_index:idx_unique_dep_cl"`
	ClusterName              string
	CreatedAt                time.Time
	UpdatedAt                *time.Time
	Values                   []byte `sql:"type:text;"`
}

// Migrate executes the table migrations for the cluster module.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&ClusterGroupDeploymentModel{},
		&TargetCluster{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating model tables")

	return db.AutoMigrate(tables...).Error
}
