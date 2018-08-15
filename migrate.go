package main

import (
	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/banzaicloud/pipeline/pkg/model"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Migrate runs migrations for the application.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	if err := model.Migrate(db, logger); err != nil {
		return err
	}

	if err := providers.Migrate(db, logger); err != nil {
		return err
	}

	return nil
}
