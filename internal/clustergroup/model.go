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

package clustergroup

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// TableName constants
const (
	clustersTableName             = "clustergroups"
	clusterGroupFeaturesTableName = "clustergroup_features"
	clusterGroupMembersTableName  = "clustergroup_members"
)

// ClusterGroupModel describes the cluster group model.
type ClusterGroupModel struct {
	ID             uint   `gorm:"primary_key"`
	UID            string `gorm:"unique_index:idx_uid"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time `gorm:"unique_index:idx_unique_id" sql:"index"`
	CreatedBy      uint
	Name           string                     `gorm:"unique_index:idx_unique_id"`
	OrganizationID uint                       `gorm:"unique_index:idx_unique_id"`
	Members        []MemberClusterModel       `gorm:"foreignkey:ClusterGroupID"`
	FeatureParams  []ClusterGroupFeatureModel `gorm:"foreignkey:ClusterGroupID"`
}

// MemberClusterModel describes a member of a cluster group.
type MemberClusterModel struct {
	ID             uint `gorm:"primary_key"`
	ClusterGroupID uint
	ClusterID      uint
}

// ClusterGroupFeature describes a feature of a cluster group.
type ClusterGroupFeatureModel struct {
	ID                 uint `gorm:"primary_key"`
	Name               string
	ClusterGroupID     uint
	Enabled            bool
	Properties         []byte `sql:"type:json"`
	ReconcileState     string
	LastReconcileError string `sql:"type:text"`
}

// TableName changes the default table name.
func (ClusterGroupModel) TableName() string {
	return clustersTableName
}

// TableName changes the default table name.
func (ClusterGroupFeatureModel) TableName() string {
	return clusterGroupFeaturesTableName
}

// TableName changes the default table name.
func (MemberClusterModel) TableName() string {
	return clusterGroupMembersTableName
}

func (g *ClusterGroupModel) BeforeCreate() (err error) {
	if g.UID == "" {
		g.UID = uuid.Must(uuid.NewV4()).String()
	}
	return
}

// String method prints formatted cluster fields.
func (g ClusterGroupModel) String() string {
	return fmt.Sprintf("Id: %d, Creation date: %s, Name: %s", g.ID, g.CreatedAt, g.Name)
}

// Migrate executes the table migrations for the cluster module.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&ClusterGroupModel{},
		&ClusterGroupFeatureModel{},
		&MemberClusterModel{},
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
