package google

import "github.com/banzaicloud/pipeline/auth"

// TableName constants
const (
	bucketsTableName = "google_buckets"
)

// ObjectStoreBucketModel is the schema for the DB.
type ObjectStoreBucketModel struct {
	ID uint `gorm:"primary_key"`

	Organization   auth.Organization `gorm:"foreignkey:OrganizationID"`
	OrganizationID uint              `gorm:"index;not null"`

	Name     string `gorm:"unique_index:idx_bucket_name"`
	Location string
}

// TableName changes the default table name.
func (ObjectStoreBucketModel) TableName() string {
	return bucketsTableName
}
