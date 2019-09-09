// Copyright © 2019 Banzai Cloud
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

package notificationadapter

import (
	"context"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // SQLite driver used for integration test
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
)

func testGormStoreGetActiveNotifications(t *testing.T) {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	err = Migrate(db, commonadapter.NewNoopLogger())
	require.NoError(t, err)

	message := "message"
	priority := int8(1)

	model := &notificationModel{
		Message:     message,
		InitialTime: time.Now().Add(-time.Hour),
		EndTime:     time.Now().Add(time.Hour),
		Priority:    priority,
	}

	err = db.Save(model).Error
	require.NoError(t, err)

	inactiveModel := &notificationModel{
		Message:     message,
		InitialTime: time.Now().Add(-2 * time.Hour),
		EndTime:     time.Now().Add(-time.Hour),
		Priority:    priority,
	}

	err = db.Save(inactiveModel).Error
	require.NoError(t, err)

	store := NewGormStore(db)

	notifications, err := store.GetActiveNotifications(context.Background())
	require.NoError(t, err)

	assert.Equal(
		t,
		[]notification.Notification{
			{
				ID:       model.ID,
				Message:  message,
				Priority: priority,
			},
		},
		notifications,
	)
}
