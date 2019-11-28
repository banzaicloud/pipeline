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

package ark

import (
	"time"

	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/src/model"
)

// ClusterBackupDeploymentsModel describes an ARK deployment model
type ClusterBackupDeploymentsModel struct {
	ID uint `gorm:"primary_key"`

	Name        string
	Namespace   string
	RestoreMode bool

	Status        string
	StatusMessage string `sql:"type:text;"`

	BucketID       uint               `gorm:"index;not null"`
	Organization   auth.Organization  `gorm:"foreignkey:OrganizationID"`
	OrganizationID uint               `gorm:"index;not null"`
	Cluster        model.ClusterModel `gorm:"foreignkey:ClusterID"`
	ClusterID      uint               `gorm:"index;not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// TableName changes the default table name
func (ClusterBackupDeploymentsModel) TableName() string {
	return clusterBackupDeploymentsTableName
}

// UpdateStatus updates the model's status and status message in database
func (m *ClusterBackupDeploymentsModel) UpdateStatus(db *gorm.DB, status, statusMessage string) error {
	m.Status = status
	m.StatusMessage = statusMessage

	err := db.Save(&m).Error
	if err != nil {
		return err
	}

	return nil
}
