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
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/src/auth"
)

// BackupsRepository describes a repository for storing backups
type BackupsRepository struct {
	org    *auth.Organization
	db     *gorm.DB
	logger logrus.FieldLogger
}

// NewBackupsRepository returns a new BackupsRepository instance
func NewBackupsRepository(org *auth.Organization, db *gorm.DB, logger logrus.FieldLogger) *BackupsRepository {

	return &BackupsRepository{
		org:    org,
		logger: logger,
		db:     db,
	}
}

// Find returns ClusterBackupsModel instances
func (r *BackupsRepository) Find() (backups []*ClusterBackupsModel, err error) {

	query := ClusterBackupsModel{
		OrganizationID: r.org.ID,
	}

	err = r.db.Where(&query).Preload("Bucket").Preload("Bucket.Deployment").Preload("Organization").Find(&backups).Error

	return
}

// FindOneByName returns a ClusterBackupsModel instance by name
func (r *BackupsRepository) FindOneByName(name string) (*ClusterBackupsModel, error) {
	var backup ClusterBackupsModel

	query := &ClusterBackupsModel{
		OrganizationID: r.org.ID,
		Name:           name,
	}

	err := r.db.Where(&query).Preload("Bucket").Preload("Bucket.Deployment").First(&backup).Error

	return &backup, err
}

// FindOneByID returns a ClusterBackupsModel instance by ID
func (r *BackupsRepository) FindOneByID(id uint) (*ClusterBackupsModel, error) {
	var backup ClusterBackupsModel

	query := &ClusterBackupsModel{
		OrganizationID: r.org.ID,
		ID:             id,
	}

	err := r.db.Where(&query).Preload("Bucket").Preload("Bucket.Deployment").First(&backup).Error

	return &backup, err
}

// FindByPersistRequest returns a ClusterBackupsModel by PersistBackupRequest
func (r *BackupsRepository) FindByPersistRequest(req *api.PersistBackupRequest) (
	*ClusterBackupsModel, error) {

	var backup ClusterBackupsModel

	query := ClusterBackupsModel{
		Name:           req.Backup.Name,
		BucketID:       req.BucketID,
		OrganizationID: r.org.ID,
	}

	err := r.db.First(&backup, &query).Error

	return &backup, err
}

// Persist persist ClusterBackupsModel from PersistBackupRequest
func (r *BackupsRepository) Persist(req *api.PersistBackupRequest) (backup ClusterBackupsModel, err error) {

	query := ClusterBackupsModel{
		Name:           req.Backup.Name,
		BucketID:       req.BucketID,
		OrganizationID: r.org.ID,
	}

	err = r.db.FirstOrInit(&backup, &query).Error
	if err != nil {
		return
	}

	err = backup.SetValuesFromRequest(r.db, req)
	if err != nil {
		return
	}

	err = r.db.Save(&backup).Error

	return
}

// DeleteBackupsWithoutBucket deletes backups from DB if their bucket is removed
func (r *BackupsRepository) DeleteBackupsWithoutBucket() error {

	bucketsTableName := ClusterBackupBucketsModel{}.TableName()

	return r.db.Where(
		"bucket_id NOT IN (?)", r.db.Table(bucketsTableName).Select("id").Where("deleted_at IS NULL").QueryExpr(),
	).Delete(&ClusterBackupsModel{}).Error
}

// DeleteBackupsNotInKeys deletes ClusterBackupsModel if their ID not in keys
func (r *BackupsRepository) DeleteBackupsNotInKeys(bucketID uint, keys []int) error {

	query := ClusterBackupsModel{
		OrganizationID: r.org.ID,
	}

	return r.db.Not(keys).Where(&query).Where(&ClusterBackupsModel{
		BucketID: bucketID,
	}).Not(&ClusterBackupsModel{Status: "Creating"}).Delete(&ClusterBackupsModel{}).Error
}

// UpdateStatus updates ClusterBackupsModel status and statusMessage fields
func (r *BackupsRepository) UpdateStatus(backup *ClusterBackupsModel, status, message string) error {

	backup.Status = status
	backup.StatusMessage = message

	return r.db.Save(&backup).Error
}
