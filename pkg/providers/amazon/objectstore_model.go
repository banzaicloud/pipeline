package amazon

import "github.com/banzaicloud/pipeline/auth"

// TableName constants
const (
	bucketsTableName = "amazon_buckets"
)

// ObjectStoreModel is the schema for the DB.
type ObjectStoreModel struct {
	ID uint `gorm:"primary_key"`

	Organization   auth.Organization `gorm:"foreignkey:OrganizationID"`
	OrganizationID uint              `gorm:"index;not null"`

	Name   string `gorm:"unique_index:idx_bucket_name"`
	Region string
}

// TableName changes the default table name.
func (ObjectStoreModel) TableName() string {
	return bucketsTableName
}
