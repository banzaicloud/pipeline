// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package route53model

import (
	"time"

	"github.com/banzaicloud/pipeline/auth"
)

// TableName constants
const (
	domainsTableName = "amazon_route53_domains"
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

// TableName changes the default table name.
func (Route53Domain) TableName() string {
	return domainsTableName
}
