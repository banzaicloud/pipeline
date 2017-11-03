package conf

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/spf13/viper"
)

func Database() *gorm.DB {
	//open a db connection
	host := viper.GetString("dev.host")
	port := viper.GetString("dev.port")
	user := viper.GetString("dev.user")
	password := viper.GetString("dev.password")
	dbname := viper.GetString("dev.dbname")

	db, err := gorm.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/"+dbname+"?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err.Error()) //Could not connect
	}
	db.LogMode(true)

	return db

}
