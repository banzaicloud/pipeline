// Copyright © 2020 Banzai Cloud
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

package auditlogdriver

import (
	"encoding/json"
	"time"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/platform/gin/auditlog"
)

// EntryModel holds all information related to a user interaction.
type EntryModel struct {
	ID            uint      `gorm:"primary_key"`
	Time          time.Time `gorm:"index"`
	CorrelationID string    `gorm:"size:36"`
	ClientIP      string    `gorm:"size:45"`
	UserAgent     string
	Path          string `gorm:"size:8000"`
	Method        string `gorm:"size:7"`
	UserID        uint
	StatusCode    int
	Body          *string `gorm:"type:json"`
	Headers       string  `gorm:"type:json"`
	ResponseTime  int
	ResponseSize  int
	Errors        *string `gorm:"type:json"`
}

// TableName specifies a database table name for the model.
func (EntryModel) TableName() string {
	return "audit_events"
}

// NewDatabaseDriver returns an audit log driver that records entries in the database.
func NewDatabaseDriver(db *gorm.DB) auditlog.Driver {
	return dbDriver{
		db: db,
	}
}

type dbDriver struct {
	db *gorm.DB
}

func (d dbDriver) Store(entry auditlog.Entry) error {
	model := EntryModel{
		Time:          entry.Time,
		CorrelationID: entry.CorrelationID,
		ClientIP:      entry.HTTP.ClientIP,
		UserAgent:     entry.HTTP.UserAgent,
		Path:          entry.HTTP.Path,
		Method:        entry.HTTP.Method,
		UserID:        entry.UserID,
		StatusCode:    entry.HTTP.StatusCode,
		Headers:       "{}",
		ResponseTime:  entry.HTTP.ResponseTime,
		ResponseSize:  entry.HTTP.ResponseSize,
	}

	// Saving the model fails when the body is not valid JSON (and not empty).
	// The previous implementation transparently suppressed these errors, so this should be fine for now.
	// The Body column should probably accept a raw body instead of enforcing JSON.
	if entry.HTTP.RequestBody != "" {
		model.Body = &entry.HTTP.RequestBody
	}

	if len(entry.HTTP.Errors) > 0 {
		e, err := json.Marshal(entry.HTTP.Errors)
		if err != nil {
			return errors.Wrap(err, "audit log")
		}

		errs := string(e)
		model.Errors = &errs
	}

	if err := d.db.Save(&model).Error; err != nil {
		return errors.WrapIf(err, "audit log")
	}

	return nil
}
