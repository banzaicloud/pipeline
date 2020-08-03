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
	"testing"
	"time"

	"github.com/jinzhu/gorm"

	//  SQLite driver used for integration test
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/platform/gin/auditlog"
)

func setUpDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	err = Migrate(db, common.NoopLogger{})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func TestDatabaseDriver(t *testing.T) {
	entry := auditlog.Entry{
		Time:          time.Date(1984, time.April, 4, 0, 0, 0, 0, time.UTC),
		CorrelationID: "cid",
		UserID:        1,
		HTTP: auditlog.HTTPEntry{
			ClientIP:     "127.0.0.1",
			UserAgent:    "go-test",
			Method:       "POST",
			Path:         "/",
			RequestBody:  "{}",
			StatusCode:   200,
			ResponseTime: 1000,
			ResponseSize: 10,
			Errors:       nil,
		},
	}

	db := setUpDatabase(t)

	driver := NewDatabaseDriver(db)

	err := driver.Store(entry)
	require.NoError(t, err)

	model := EntryModel{
		ID:            1,
		Time:          entry.Time,
		CorrelationID: entry.CorrelationID,
		ClientIP:      entry.HTTP.ClientIP,
		UserAgent:     entry.HTTP.UserAgent,
		Path:          entry.HTTP.Path,
		Method:        entry.HTTP.Method,
		UserID:        entry.UserID,
		StatusCode:    entry.HTTP.StatusCode,
		Body:          &entry.HTTP.RequestBody,
		Headers:       "{}",
		ResponseTime:  entry.HTTP.ResponseTime,
		ResponseSize:  entry.HTTP.ResponseSize,
		Errors:        nil,
	}

	var expectedModel EntryModel

	err = db.
		Where(EntryModel{Path: "/"}).
		First(&expectedModel).
		Error
	require.NoError(t, err)

	assert.Equal(t, model, expectedModel)
}
