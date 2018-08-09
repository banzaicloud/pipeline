package route53model

import (
	"time"

	"github.com/banzaicloud/pipeline/auth"
)

// Route53Domain describes the database model
// for storing the state of domains registered with with Amazon Route53 DNS service
type Route53Domain struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Organization auth.Organization `gorm:"foreignkey:OrganizationId"`

	OrganizationId uint   `gorm:"unique_index;not null"`
	Domain         string `gorm:"unique_index;not null"`
	HostedZoneId   string
	PolicyArn      string
	IamUser        string
	AwsAccessKeyId string
	Status         string `gorm:"not null"`
	ErrorMessage   string `sql:"type:text;"`
}
