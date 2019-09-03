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
	"time"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
)

// TableName constants
const (
	notificationTableName = "notifications"
)

type notificationModel struct {
	ID          uint      `gorm:"primary_key"`
	Message     string    `gorm:"not null" sql:"type:text;"`
	InitialTime time.Time `gorm:"index:idx_initial_time_end_time;default:current_timestamp;not null"`
	EndTime     time.Time `gorm:"index:idx_initial_time_end_time;default:'1970-01-01 00:00:01';not null"`
	Priority    int8      `gorm:"not null"`
}

// TableName changes the default table name.
func (notificationModel) TableName() string {
	return notificationTableName
}

// GormNotificationStore is a notification store using Gorm for data persistence.
type GormNotificationStore struct {
	db *gorm.DB
}

// NewGormNotificationStore returns a new GormNotificationStore.
func NewGormNotificationStore(db *gorm.DB) *GormNotificationStore {
	return &GormNotificationStore{
		db: db,
	}
}

func (s *GormNotificationStore) GetActiveNotifications(ctx context.Context) ([]notification.Notification, error) {
	var notifications []notificationModel

	err := s.db.Find(&notifications, "NOW() BETWEEN initial_time AND end_time").Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to find notifications")
	}
	var result []notification.Notification

	for _, n := range notifications {
		result = append(result, notification.Notification{
			ID:       n.ID,
			Message:  n.Message,
			Priority: n.Priority,
		})
	}

	return result, nil
}
