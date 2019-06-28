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
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

// TableName constants
const (
	clusterFeatureTableName = "cluster_feature"
)

type featureSpec map[string]interface{}

func (fs featureSpec) Scan(src interface{}) error {
	value, err := cast.ToStringE(src)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(value), &fs)
}

func (fs featureSpec) Value() (driver.Value, error) {
	v, err := json.Marshal(fs)
	if err != nil {
		return "", err
	}
	return v, nil
}

// clusterFeatureModel describes the cluster group model.
type clusterFeatureModel struct {
	// injecting timestamp fields
	gorm.Model

	Name      string
	Status    string
	ClusterID uint
	Spec      featureSpec `gorm:"type:text"`
	CreatedBy uint
}

// TableName changes the default table name.
func (cfm clusterFeatureModel) TableName() string {
	return clusterFeatureTableName
}

// String method prints formatted cluster fields.
func (cfm clusterFeatureModel) String() string {
	return fmt.Sprintf("Id: %d, Creation date: %s, Name: %s", cfm.ID, cfm.CreatedAt, cfm.Name)
}

// Migrate executes the table migrations for the cluster module.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&clusterFeatureModel{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.Info("migrating model tables")

	return db.AutoMigrate(tables...).Error
}
