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

package audit

import (
	"time"

	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
)

// TableName constants
const (
	auditEventTableName = "audit_events"
)

// AuditEvent holds all information related to a user interaction.
type AuditEvent struct {
	ID            uint      `gorm:"primary_key"`
	Time          time.Time `gorm:"index"`
	CorrelationID string    `gorm:"size:36"`
	ClientIP      string    `gorm:"size:45"`
	UserAgent     string
	Path          string `gorm:"size:8000"`
	Method        string `gorm:"size:7"`
	UserID        pkgAuth.UserID
	StatusCode    int
	Body          *string `gorm:"type:json"`
	Headers       string  `gorm:"type:json"`
	ResponseTime  int
	ResponseSize  int
	Errors        *string `gorm:"type:json"`
}

// TableName specifies a database table name for the model.
func (AuditEvent) TableName() string {
	return auditEventTableName
}
