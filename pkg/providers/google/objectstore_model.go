package google

import "github.com/banzaicloud/pipeline/auth"

// ObjectStoreModel is the schema for the DB
type ObjectStoreModel struct {
	ID uint `gorm:"primary_key"`

	Organization   auth.Organization `gorm:"foreignkey:OrganizationID"`
	OrganizationID uint              `gorm:"index;not null"`

	Name     string `gorm:"unique_index:idx_bucket_name"`
	Location string
}

// TableName changes the default table name.
func (ObjectStoreModel) TableName() string {
	return "google_buckets"
}
