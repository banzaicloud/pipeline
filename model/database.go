package model

import (
	"github.com/banzaicloud/pipeline/config"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var db *gorm.DB
var logger *logrus.Logger

// Simple init for logging
func init() {
	logger = config.Logger()
}

func initDatabase() *gorm.DB {
	log := logger.WithFields(logrus.Fields{"action": "ConnectDB"})
	host := viper.GetString("database.host")
	port := viper.GetString("database.port")
	user := viper.GetString("database.user")
	password := viper.GetString("database.password")
	dbName := viper.GetString("database.dbname")
	//TODO this shouldn't be shared
	db, err := gorm.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/"+dbName+"?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		log.Error("Database connection failed")
		panic(err.Error()) //Could not connect
	}
	db.LogMode(true)
	return db
}

func GetDB() *gorm.DB {
	if db == nil {
		db = initDatabase()
	}
	return db
}
