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

package cluster

import (
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/model"
)

// Migrate executes the table migrations for the cluster module.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&MigrationClusterModel{},
		&StatusHistoryModel{},
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

type MigrationClusterModel struct {
	ID  uint   `gorm:"primary_key"`
	UID string `gorm:"unique_index:idx_clusters_uid"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"unique_index:idx_clusters_unique_id" sql:"index"`
	StartedAt *time.Time
	CreatedBy uint

	Name           string `gorm:"unique_index:idx_clusters_unique_id"`
	Location       string
	Cloud          string
	Distribution   string
	OrganizationID uint `gorm:"unique_index:idx_clusters_unique_id"`
	SecretID       string
	ConfigSecretID string
	SSHSecretID    string
	Status         string
	RbacEnabled    bool
	OidcEnabled    bool `gorm:"default:false;not null"`
	Monitoring     bool
	Logging        bool
	SecurityScan   bool
	StatusMessage  string             `sql:"type:text;"`
	ScaleOptions   model.ScaleOptions `gorm:"foreignkey:ClusterID"`
	TtlMinutes     uint               `gorm:"default:0"`
}

// TableName changes the default table name.
func (MigrationClusterModel) TableName() string {
	return clustersTableName
}
