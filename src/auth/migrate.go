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

package auth

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Migrate executes the table migrations for the auth module.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&AuthIdentity{},
		&User{},
		&UserOrganization{},
		&Organization{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating auth tables")

	err := db.AutoMigrate(tables...).Error
	if err != nil {
		return err
	}

	// Migrate Organization normalized names
	// Unique constraints are not handled here
	switch db.Dialect().GetName() {
	case "mysql", "postgres":
		err = db.Exec("UPDATE organizations SET normalized_name = REPLACE(REPLACE(name, '.', '-'), '@', '-') WHERE normalized_name = '' OR normalized_name IS NULL").Error
		if err != nil {
			return err
		}

	case "sqlite3":
		// Noop

	default:
		return errors.New("cannot migrate organization normalized names for this dialect")
	}

	return nil
}
