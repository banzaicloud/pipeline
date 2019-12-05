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

package azure

import (
	"github.com/banzaicloud/pipeline/src/auth"
)

// TableName constants
const (
	bucketsTableName = "azure_buckets"
)

// ObjectStoreBucketModel is the schema for the DB.
type ObjectStoreBucketModel struct {
	ID uint `gorm:"primary_key"`

	Organization   auth.Organization `gorm:"foreignkey:OrganizationID"`
	OrganizationID uint              `gorm:"index;not null"`

	Name           string `gorm:"unique_index:idx_azure_bucket_name"`
	ResourceGroup  string `gorm:"unique_index:idx_azure_bucket_name"`
	StorageAccount string `gorm:"unique_index:idx_azure_bucket_name"`
	Location       string

	SecretRef       string
	Status          string
	StatusMsg       string `sql:"type:text;"`
	AccessSecretRef string
}

// TableName changes the default table name.
func (ObjectStoreBucketModel) TableName() string {
	return bucketsTableName
}
