package oracle

import (
	"github.com/banzaicloud/pipeline/auth"
)

// TableName constants
const (
	bucketsTableName = "oracle_buckets"
)

// ObjectStoreModel is the schema for the DB
type ObjectStoreModel struct {
	ID uint `gorm:"primary_key"`

	Organization auth.Organization `gorm:"foreignkey:OrgID"`
	OrgID        uint              `gorm:"index;not null"`

	CompartmentID string `gorm:"unique_index:bucketNameLocationCompartment"`
	Name          string `gorm:"unique_index:bucketNameLocationCompartment"`
	Location      string `gorm:"unique_index:bucketNameLocationCompartment"`
}

// TableName sets the ObjectStoreModel table name
func (ObjectStoreModel) TableName() string {
	return bucketsTableName
}
