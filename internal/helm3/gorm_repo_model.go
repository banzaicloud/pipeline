// Copyright © 2020 Banzai Cloud
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

package helm3

import (
	"github.com/jinzhu/gorm"
)

// TableName constants
const (
	tableName = "helm_repositories"
)

// ClusterModel describes the common cluster model.
type RepositoryModel struct {
	gorm.Model

	Name             string
	URL              string
	OrganizationID   uint // FK to organizations
	PasswordSecretID string
	TlsSecretID      string
}

// TableName changes the default table name.
func (RepositoryModel) TableName() string {
	return tableName
}
