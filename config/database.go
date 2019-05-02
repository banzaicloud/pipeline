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

	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
)

// nolint: gochecknoglobals
var dbOnce sync.Once

// nolint: gochecknoglobals
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
		Dialect:   viper.GetString("database.dialect"),
		Host:      viper.GetString("database.host"),
		Port:      viper.GetInt("database.port"),
		TLS:       viper.GetString("database.tls"),
		Role:      viper.GetString("database.role"),
		User:      viper.GetString("database.user"),
		Pass:      viper.GetString("database.password"),
		Name:      viper.GetString("database.dbname"),
		EnableLog: viper.GetBool("database.logging"),
	}
}
