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

package spotguide

import (
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
)

// TODO remove this in the next release

// Migrate executes the table migrations for the spotguide module.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&SpotguideRepo{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating spotguide tables")

	return db.AutoMigrate(tables...).Error
}

const SpotguideRepoTableName = "spotguide_repos"

// Question is an opaque struct from Pipeline's point of view
type Question map[string]interface{}

type SpotguideYAML struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description,omitempty"`
	Tags        []string                    `json:"tags,omitempty"`
	Resources   pipeline.RequestedResources `json:"resources"`
	Questions   []Question                  `json:"questions"`
}

// nolint: govet
type SpotguideRepo struct {
	ID               uint      `json:"id" gorm:"primary_key"`
	OrganizationID   uint      `json:"organizationId" gorm:"unique_index:idx_spotguide_name_and_version"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	Name             string    `json:"name" gorm:"unique_index:idx_spotguide_name_and_version"`
	DisplayName      string    `json:"displayName" gorm:"-"`
	Icon             []byte    `json:"-" gorm:"size:65536"`
	Readme           string    `json:"readme" gorm:"type:text"`
	Version          string    `json:"version" gorm:"unique_index:idx_spotguide_name_and_version"`
	SpotguideYAMLRaw []byte    `json:"-" gorm:"type:text"`
	SpotguideYAML    `gorm:"-"`
}

func (SpotguideRepo) TableName() string {
	return SpotguideRepoTableName
}
