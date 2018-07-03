package objectstore

import (
	pipelineAuth "github.com/banzaicloud/pipeline/auth"
)

// TableName constants
const (
	ManagedOracleBucketTableName = "managed_oracle_buckets"
)

// ManagedOracleBucket is the schema for the DB
type ManagedOracleBucket struct {
	ID            uint                      `gorm:"primary_key"`
	Organization  pipelineAuth.Organization `gorm:"foreignkey:OrgID"`
	OrgID         uint                      `gorm:"index;not null"`
	CompartmentID string
	Name          string `gorm:"unique_index:bucketName"`
}

// TableName sets the NodePools table name
func (ManagedOracleBucket) TableName() string {
	return ManagedOracleBucketTableName
}
