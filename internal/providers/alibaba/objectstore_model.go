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

package alibaba

import (
	"github.com/banzaicloud/pipeline/auth"
)

// TableName constants
const (
	bucketsTableName = "alibaba_buckets"
)

// ObjectStoreBucketModel is the schema for the DB
type ObjectStoreBucketModel struct {
	ID           uint              `gorm:"primary_key"`
	Organization auth.Organization `gorm:"foreignkey:OrgID"`
	OrgID        uint              `gorm:"index;not null"`
	Name         string            `gorm:"unique_index:idx_bucket_name"`
	Region       string
	SecretRef    string

	Status    string
	StatusMsg string `sql:"type:text;"`
}

// TableName changes the default table name.
func (ObjectStoreBucketModel) TableName() string {
	return bucketsTableName
}
