package config

import (
	"sync"

	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
)

var dbOnce sync.Once
var db *gorm.DB

func initDatabase() {
	config := NewDBConfig()

	err := config.Validate()
	if err != nil {
		logger.Panic("invalid database config: ", err.Error())
	}

	logger := Logger()

	db, err = database.Connect(config)
	if err != nil {
		logger.Panic("failed to initialize db: ", err.Error())
	}
}

// DB returns an initialized DB instance.
func DB() *gorm.DB {
	dbOnce.Do(initDatabase)

	return db
}

// NewDBConfig returns a new DB configuration struct.
func NewDBConfig() database.Config {
	return database.Config{
		Host:      viper.GetString("database.host"),
		Port:      viper.GetInt("database.port"),
		Role:      viper.GetString("database.role"),
		User:      viper.GetString("database.user"),
		Pass:      viper.GetString("database.password"),
		Name:      viper.GetString("database.dbname"),
		EnableLog: viper.GetBool("database.logging"),
	}
}
