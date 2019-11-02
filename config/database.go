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

package config

import (
	"sync"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/platform/database"
)

// nolint: gochecknoglobals
var dbOnce sync.Once

// nolint: gochecknoglobals
var db *gorm.DB

func initDatabase() {
	config := NewDBConfig()

	err := config.Validate()
	if err != nil {
		emperror.Panic(errors.WrapIf(err, "invalid database config"))
	}

	db, err = database.Connect(config)
	if err != nil {
		emperror.Panic(errors.WrapIf(err, "failed to initialize db"))
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
		Dialect:  viper.GetString("database.dialect"),
		Host:     viper.GetString("database.host"),
		Port:     viper.GetInt("database.port"),
		TLS:      viper.GetString("database.tls"),
		Role:     viper.GetString("database.role"),
		User:     viper.GetString("database.user"),
		Password: viper.GetString("database.password"),
		Name:     viper.GetString("database.dbname"),
		QueryLog: viper.GetBool("database.logging"),
	}
}
