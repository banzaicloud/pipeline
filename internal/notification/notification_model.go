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

package notification

import "time"

// TableName constants
const (
	notificationTableName = "notifications"
)

type NotificationModel struct {
	ID          uint      `gorm:"primary_key"`
	Message     string    `gorm:"not null" sql:"type:text;"`
	InitialTime time.Time `gorm:"index:idx_initial_time_end_time;default:current_timestamp;not null"`
	EndTime     time.Time `gorm:"index:idx_initial_time_end_time;default:'1970-01-01 00:00:01';not null"`
	Priority    int8      `gorm:"not null"`
}

// TableName changes the default table name.
func (NotificationModel) TableName() string {
	return notificationTableName
}
