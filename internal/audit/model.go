package audit

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Migrate executes the table migrations for the provider.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&AuditEvent{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating audit tables")

	return db.AutoMigrate(tables...).Error
}
