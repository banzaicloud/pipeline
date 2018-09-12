package audit

import "time"

// TableName constants
const (
	auditEventTableName = "audit_events"
)

// AuditEvent holds all information related to a user interaction.
type AuditEvent struct {
	ID         uint      `gorm:"primary_key"`
	Time       time.Time `gorm:"index"`
	ClientIP   string    `gorm:"size:45"`
	UserAgent  string
	Path       string `gorm:"size:8000"`
	Method     string `gorm:"size:7"`
	UserID     uint
	StatusCode int
	Body       *string `gorm:"type:json"`
	Headers    string  `gorm:"type:json"`
}

// TableName specifies a database table name for the model.
func (AuditEvent) TableName() string {
	return auditEventTableName
}
