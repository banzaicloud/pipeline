package database

import "github.com/jinzhu/gorm"

// IsRecordNotFoundError returns true if the error was triggered by a record not being found.
func IsRecordNotFoundError(err error) bool {
	return gorm.IsRecordNotFoundError(err)
}
