package main

import (
	"github.com/banzaicloud/pipeline/internal/audit"
	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Migrate runs migrations for the application.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	if err := audit.Migrate(db, logger); err != nil {
		return err
	}

	if err := cluster.Migrate(db, logger); err != nil {
		return err
	}

	if err := providers.Migrate(db, logger); err != nil {
		return err
	}

	return nil
}
