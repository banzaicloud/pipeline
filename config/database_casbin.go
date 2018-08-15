package config

import (
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/pkg/errors"
)

// CasbinDSN returns the connection string for Casbin gorm adapter.
func CasbinDSN() (string, error) {
	config := NewDBConfig()

	err := config.Validate()
	if err != nil {
		return "", errors.Wrap(err, "invalid database config")
	}

	dsn, err := database.GetDSN(config)
	if err != nil {
		return "", errors.Wrap(err, "could not get DSN for casbin gorm adapter")
	}

	return dsn, nil
}
