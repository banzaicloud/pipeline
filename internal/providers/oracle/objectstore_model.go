package oracle

import (
	"github.com/banzaicloud/pipeline/auth"
)

// TableName constants
const (
	bucketsTableName = "oracle_buckets"
)

// ObjectStoreBucketModel is the schema for the DB.
type ObjectStoreBucketModel struct {
	ID uint `gorm:"primary_key"`

	Organization auth.Organization `gorm:"foreignkey:OrgID"`
	OrgID        uint              `gorm:"index;not null"`

	CompartmentID string `gorm:"unique_index:bucketNameLocationCompartment"`
	Name          string `gorm:"unique_index:bucketNameLocationCompartment"`
	Location      string `gorm:"unique_index:bucketNameLocationCompartment"`
}

// TableName changes the default table name.
func (ObjectStoreBucketModel) TableName() string {
	return bucketsTableName
}
