package model

import (
	"sync"

	"github.com/banzaicloud/bank-vaults/database"
	"github.com/banzaicloud/pipeline/config"
	"github.com/jinzhu/gorm"
	// blank import is used here for simplicity
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var dbOnce sync.Once
var db *gorm.DB
var logger *logrus.Logger

// Simple init for logging
func init() {
	logger = config.Logger()
}

func initDatabase() {
	log := logger.WithFields(logrus.Fields{"action": "ConnectDB"})
	host := viper.GetString("database.host")
	port := viper.GetString("database.port")
	role := viper.GetString("database.role")
	dbName := viper.GetString("database.dbname")
	dataSource, err := database.DynamicSecretDataSource("mysql", role+"@tcp("+host+":"+port+")/"+dbName+"?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		log.Error("Database dyanimc secret acquisition failed")
		panic(err.Error())
	}
	database, err := gorm.Open("mysql", dataSource)
	if err != nil {
		log.Error("Database connection failed")
		panic(err.Error())
	}
	database.LogMode(true)
	db = database
}

//GetDB returns an initialized DB
func GetDB() *gorm.DB {
	dbOnce.Do(initDatabase)
	return db
}

//IsErrorGormNotFound returns gorm.ErrRecordNotFound
func IsErrorGormNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
