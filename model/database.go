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
	dbName := viper.GetString("database.dbname")
	db = ConnectDB(dbName)
}

func GetDataSource(dbName string) string {
	log := logger.WithFields(logrus.Fields{"action": "GetDataSource"})
	host := viper.GetString("database.host")
	port := viper.GetString("database.port")
	role := viper.GetString("database.role")
	user := viper.GetString("database.user")
	password := viper.GetString("database.password")
	dataSource := "@tcp(" + host + ":" + port + ")/" + dbName
	if role != "" {
		var err error
		dataSource, err = database.DynamicSecretDataSource("mysql", role+dataSource)
		if err != nil {
			log.Error("Database dynamic secret acquisition failed")
			panic(err.Error())
		}
	} else {
		dataSource = user + ":" + password + dataSource
	}
	return dataSource
}

func ConnectDB(dbName string) *gorm.DB {
	log := logger.WithFields(logrus.Fields{"action": "ConnectDB"})
	dataSource := GetDataSource(dbName)
	dataSource += "?charset=utf8&parseTime=True&loc=Local"
	database, err := gorm.Open("mysql", dataSource)
	if err != nil {
		log.Error("Database connection failed")
		panic(err.Error())
	}
	database.LogMode(true)
	return database
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
