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
	"github.com/banzaicloud/pipeline/internal/ark/api"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
)

// ClusterBackupBucketsModel describes a cluster backup bucket
type ClusterBackupBucketsModel struct {
	ID uint `gorm:"primary_key"`

	Cloud          string
	SecretID       pkgSecret.SecretID
	BucketName     string
	Location       string
	StorageAccount string
	ResourceGroup  string

	Status        string
	StatusMessage string `sql:"type:text;"`

	Organization   auth.Organization             `gorm:"foreignkey:OrganizationID"`
	OrganizationID uint                          `gorm:"index;not null"`
	Deployment     ClusterBackupDeploymentsModel `gorm:"foreignkey:BucketID"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// TableName changes the default table name
func (ClusterBackupBucketsModel) TableName() string {
	return clusterBackupBucketsTableName
}

// BeforeDelete sets status to DELETED before soft delete the record
func (m *ClusterBackupBucketsModel) BeforeDelete(db *gorm.DB) error {
	m.Status = "DELETED"
	m.StatusMessage = ""
	return db.Save(&m).Error
}

// ConvertModelToEntity converts a ClusterBackupBucketsModel to Bucket
func (m *ClusterBackupBucketsModel) ConvertModelToEntity() *api.Bucket {

	inUse := false
	if m.Deployment.Cluster.ID > 0 {
		inUse = true
	}

	return &api.Bucket{
		ID:       m.ID,
		Name:     m.BucketName,
		Cloud:    m.Cloud,
		SecretID: m.SecretID,
		Location: m.Location,
		AzureBucketProperties: api.AzureBucketProperties{
			StorageAccount: m.StorageAccount,
			ResourceGroup:  m.ResourceGroup,
		},
		Status: m.Status,
		InUse:  inUse,

		DeploymentID:        m.Deployment.ID,
		ClusterID:           m.Deployment.Cluster.ID,
		ClusterCloud:        m.Deployment.Cluster.Cloud,
		ClusterDistribution: m.Deployment.Cluster.Distribution,
	}
}
