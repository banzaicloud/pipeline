package objectstore

import (
	"github.com/banzaicloud/pipeline/database"
	"github.com/sirupsen/logrus"
)

// Init initializes the models
func Init(logger *logrus.Entry) error {

	logger.Infoln("Create Oracle object store table(s):",
		ManagedOracleBucket.TableName(ManagedOracleBucket{}),
	)

	return database.GetDB().AutoMigrate(
		&ManagedOracleBucket{},
	).Error
}
