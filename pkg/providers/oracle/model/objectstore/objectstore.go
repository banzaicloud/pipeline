package objectstore

import (
	"github.com/banzaicloud/pipeline/auth"
)

// TableName constants
const (
	BucketsTableName = "oracle_buckets"
)

// ObjectStoreBucket is the schema for the DB
type ObjectStoreBucket struct {
	ID uint `gorm:"primary_key"`

	Organization auth.Organization `gorm:"foreignkey:OrgID"`
	OrgID        uint              `gorm:"index;not null"`

	CompartmentID string `gorm:"unique_index:bucketNameLocationCompartment"`
	Name          string `gorm:"unique_index:bucketNameLocationCompartment"`
	Location      string `gorm:"unique_index:bucketNameLocationCompartment"`
}

// TableName sets the ObjectStoreBucket table name
func (ObjectStoreBucket) TableName() string {
	return BucketsTableName
}
