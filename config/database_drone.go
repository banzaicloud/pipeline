package config

import (
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

// DroneDB returns an initialized DB instance for Drone.
func DroneDB() (*gorm.DB, error) {
	config := NewDBConfig()

	config.Name = "drone"

	err := config.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid database config")
	}

	db, err := database.Connect(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize drone db")
	}

	return db, nil
}
