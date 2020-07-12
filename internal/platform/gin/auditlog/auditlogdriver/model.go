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

package auditlogdriver

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/common"
)

// Migrate executes the table migrations for the audit model.
func Migrate(db *gorm.DB, logger common.Logger) error {
	tables := []interface{}{
		&EntryModel{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(map[string]interface{}{
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating audit log tables")

	return db.AutoMigrate(tables...).Error
}
