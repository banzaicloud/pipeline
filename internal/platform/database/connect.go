// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
