// Copyright © 2020 Banzai Cloud
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

package helmadapter

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
)

// Migrate executes the table migrations for the helm module.
func Migrate(db *gorm.DB, logger Logger) error {
	tables := []interface{}{
		repositoryModel{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.Info("migrating model tables", map[string]interface{}{"table_names": strings.TrimSpace(tableNames)})

	return db.AutoMigrate(tables...).Error
}
