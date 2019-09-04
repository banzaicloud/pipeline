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

package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Migrate executes the table migrations for the auth module.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&AuthIdentity{},
		&User{},
		&UserOrganization{},
		&OrganizationMigrationModel{}, // TODO change back to Organization once a new version is released
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating auth tables")

	return db.AutoMigrate(tables...).Error
}

// OrganizationMigrationModel represents a unit of users and resources.
type OrganizationMigrationModel struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	GithubID  *int64    `gorm:"unique" json:"githubId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name      string    `gorm:"unique;not null" json:"name"`
	Provider  string    `gorm:"not null" json:"provider"`
	Users     []User    `gorm:"many2many:user_organizations" json:"users,omitempty"`
	Role      string    `json:"-" gorm:"-"` // Used only internally
}

// TableName changes the default table name.
func (OrganizationMigrationModel) TableName() string {
	return "organizations"
}
