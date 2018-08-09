package main

import (
	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/banzaicloud/pipeline/pkg/providers/google"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Migrate runs migrations for the application.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	if err := amazon.Migrate(db, logger); err != nil {
		return err
	}

	if err := azure.Migrate(db, logger); err != nil {
		return err
	}

	if err := google.Migrate(db, logger); err != nil {
		return err
	}

	return nil
}
