package database

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // blank import is used here for simplicity
)

func Connect(c Config) (*gorm.DB, error) {
	// Custom parameters
	c.Params = map[string]string{
		"charset":   "utf8",
		"parseTime": "True",
		"loc":       "Local",
	}

	dsn, err := GetDSN(c)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.LogMode(c.EnableLog)

	return db, nil
}
