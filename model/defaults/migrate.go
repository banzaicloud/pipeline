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

package defaults

import (
	"fmt"
	"strings"

	"github.com/banzaicloud/pipeline/model"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Migrate executes the table migrations for the defaults module.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&EKSProfile{},
		&EKSNodePoolProfile{},
		&EKSNodePoolLabelsProfile{},
		&AKSProfile{},
		&AKSNodePoolProfile{},
		&AKSNodePoolLabelsProfile{},
		&GKEProfile{},
		&GKENodePoolProfile{},
		&GKENodePoolLabelsProfile{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating defaults tables")

	if err := db.AutoMigrate(tables...).Error; err != nil {
		return err
	}

	if err := model.AddForeignKey(db, logger, &EKSProfile{}, &EKSNodePoolProfile{}, "Name"); err != nil {
		return err
	}

	if err := model.AddForeignKey(db, logger, &EKSNodePoolProfile{}, &EKSNodePoolLabelsProfile{}, "NodePoolProfileID"); err != nil {
		return err
	}

	if err := model.AddForeignKey(db, logger, &AKSProfile{}, &AKSNodePoolProfile{}, "Name"); err != nil {
		return err
	}

	if err := model.AddForeignKey(db, logger, &AKSNodePoolProfile{}, &AKSNodePoolLabelsProfile{}, "NodePoolProfileID"); err != nil {
		return err
	}

	if err := model.AddForeignKey(db, logger, &GKEProfile{}, &GKENodePoolProfile{}, "Name"); err != nil {
		return err
	}

	if err := model.AddForeignKey(db, logger, &GKENodePoolProfile{}, &GKENodePoolLabelsProfile{}, "NodePoolProfileID"); err != nil {
		return err
	}

	return nil
}
