package auth_identity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// SignLogs record sign in logs
type SignLogs struct {
	Log         string `sql:"-"`
	SignInCount uint
	Logs        []SignLog
}

// Scan scan data into sign logs
func (signLogs *SignLogs) Scan(data interface{}) (err error) {
	switch values := data.(type) {
	case []byte:
		if string(values) != "" {
			return json.Unmarshal(values, signLogs)
		}
	case string:
		return signLogs.Scan([]byte(values))
	case []string:
		for _, str := range values {
			if err := signLogs.Scan(str); err != nil {
				return err
			}
		}
	default:
		err = errors.New("unsupported driver -> Scan pair for SignLogs")
	}
	return
}

// Value return struct's Value
func (signLogs SignLogs) Value() (driver.Value, error) {
	results, err := json.Marshal(signLogs)
	return string(results), err
}

// SignLog sign log
type SignLog struct {
	UserAgent string
	At        *time.Time
	IP        string
}
