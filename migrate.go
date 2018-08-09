package main

import (
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Migrate runs migrations for the application.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	err := azure.Migrate(db, logger)
	if err != nil {
		return err
	}

	return nil
}
