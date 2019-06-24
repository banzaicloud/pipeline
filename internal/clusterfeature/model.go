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

package clusterfeature

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// TableName constants
const (
	clusterFeatureTableName = "clusterfeature"
)

// ClusterFeatureModel describes the cluster group model.
type ClusterFeatureModel struct {
	ID        int `gorm:"primary_key"`
	Name      string
	Status    string
	ClusterID uint
	Spec      []byte
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
	CreatedBy uint
}

// TableName changes the default table name.
func (cfm ClusterFeatureModel) TableName() string {
	return clusterFeatureTableName
}

// String method prints formatted cluster fields.
func (cfm ClusterFeatureModel) String() string {
	return fmt.Sprintf("Id: %d, Creation date: %s, Name: %s", cfm.ID, cfm.CreatedAt, cfm.Name)
}

// Migrate executes the table migrations for the cluster module.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&ClusterFeatureModel{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.Info("migrating model tables")

	return db.AutoMigrate(tables...).Error
}
