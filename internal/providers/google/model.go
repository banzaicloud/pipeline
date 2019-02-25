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

package google

import (
	"fmt"
	"strings"

	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/providers/google"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Migrate executes the table migrations for the provider.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&ObjectStoreBucketModel{},

		&GKEClusterModel{},
		&GKENodePoolModel{},
		&GKENodePoolLabelModel{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"provider":    google.Provider,
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating provider tables")

	err := db.AutoMigrate(tables...).Error
	if err != nil {
		return err
	}

	err = model.AddForeignKey(db, logger, &model.ClusterModel{}, &GKEClusterModel{}, "ClusterID")
	if err != nil {
		return err
	}

	err = model.AddForeignKeyAndReferencedKey(db, logger, &GKEClusterModel{}, &GKENodePoolModel{}, "ClusterID", "ClusterID")
	if err != nil {
		return err
	}

	return nil
}
