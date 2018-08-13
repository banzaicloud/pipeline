package oracle

import (
	"fmt"
	"strings"

	"github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	objectstoreModel "github.com/banzaicloud/pipeline/pkg/providers/oracle/model/objectstore"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Migrate executes the table migrations for the provider.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&objectstoreModel.ObjectStoreBucket{},
		&model.Cluster{},
		&model.NodePool{},
		&model.NodePoolSubnet{},
		&model.NodePoolLabel{},
		&model.Profile{},
		&model.ProfileNodePool{},
		&model.ProfileNodePoolLabel{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"provider":    Provider,
		"table_names": strings.TrimLeft(tableNames, " "),
	}).Info("migrating provider tables")

	return db.AutoMigrate(tables...).Error
}
