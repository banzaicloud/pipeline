package database

import (
	"github.com/jinzhu/gorm"
	"github.com/banzaicloud/banzai-types/utils"
	"github.com/banzaicloud/banzai-types/constants"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var db *gorm.DB

func Init(host, port, user, password, dbName string) {
	db = initDatabase(host, port, user, password, dbName)
}

func DB() *gorm.DB {
	return db
}

func initDatabase(host, port, user, password, dbName string) *gorm.DB {

	//open a db connection
	db, err := gorm.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/"+dbName+"?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		utils.LogError(constants.TagDatabase, "database connection failed")
		panic(err.Error()) //Could not connect
	}
	db.LogMode(true)

	return db

}

func CreateTables(values ...interface{}) {
	db.AutoMigrate(values...)
}

func SelectFirstWhere(output interface{}, query interface{}, value ...interface{}) *gorm.DB {
	return db.Where(query, value).First(output)
}

func Query(sql string, values interface{}, output interface{}) *gorm.DB {
	return db.Raw(sql, values).Scan(output)
}

func First(where string, output interface{}) *gorm.DB {
	return db.First(output, where)
}

func Find(output interface{}) *gorm.DB {
	return db.Find(output)
}

func Save(value interface{}) *gorm.DB {
	return db.Save(value)
}

func Delete(value interface{}) *gorm.DB {
	return db.Delete(value)
}

func Model(value interface{}) *gorm.DB {
	return db.Model(value)
}

func Update(attrs ...interface{}) *gorm.DB {
	return db.Update(attrs)
}
