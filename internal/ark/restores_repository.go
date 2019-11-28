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
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/src/auth"
)

// RestoresRepository is a repository for managing ARK restore models
type RestoresRepository struct {
	org     *auth.Organization
	cluster api.Cluster
	db      *gorm.DB
	logger  logrus.FieldLogger
}

// NewRestoresRepository creates and returns a RestoresRepository instance
func NewRestoresRepository(
	org *auth.Organization,
	cluster api.Cluster,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *RestoresRepository {

	return &RestoresRepository{
		org:     org,
		cluster: cluster,
		db:      db,
		logger:  logger,
	}
}

// Find finds all ClusterBackupRestoresModel
func (r *RestoresRepository) Find() ([]*ClusterBackupRestoresModel, error) {
	var restores []*ClusterBackupRestoresModel

	query := ClusterBackupRestoresModel{
		OrganizationID: r.org.ID,
		ClusterID:      r.cluster.GetID(),
	}

	err := r.db.Where(&query).Preload("Cluster").Preload("Organization").Find(&restores).Error

	return restores, err
}

// FindOneByName find one ClusterBackupRestoresModel by name
func (r *RestoresRepository) FindOneByName(name string) (*ClusterBackupRestoresModel, error) {
	var restore ClusterBackupRestoresModel

	query := ClusterBackupRestoresModel{
		Name: name,

		OrganizationID: r.org.ID,
		ClusterID:      r.cluster.GetID(),
	}

	err := r.db.Where(&query).Preload("Bucket").Preload("Cluster").Preload("Organization").Find(&restore).Error

	return &restore, err
}

// FindOneByID find one ClusterBackupRestoresModel by ID
func (r *RestoresRepository) FindOneByID(id uint) (*ClusterBackupRestoresModel, error) {
	var restore ClusterBackupRestoresModel

	query := ClusterBackupRestoresModel{
		ID: id,

		OrganizationID: r.org.ID,
		ClusterID:      r.cluster.GetID(),
	}

	err := r.db.Where(&query).Preload("Bucket").Preload("Cluster").Preload("Organization").Find(&restore).Error

	return &restore, err
}

// Persist persists a ClusterBackupRestoresModel by a PersistRestoreRequest
func (r *RestoresRepository) Persist(req *api.PersistRestoreRequest) (restore ClusterBackupRestoresModel, err error) {

	log := r.logger.WithField("restore-name", req.Restore.Name)

	query := ClusterBackupRestoresModel{
		UID:            string(req.Restore.GetUID()),
		BucketID:       req.BucketID,
		ClusterID:      r.cluster.GetID(),
		OrganizationID: r.org.ID,
	}

	var existingRecord ClusterBackupRestoresModel
	existingRecordResult := r.db.Where(&query).First(&existingRecord)

	err = r.db.FirstOrInit(&restore, &query).Error
	if err != nil {
		return
	}

	err = restore.SetValuesFromRequest(req)
	if err != nil {
		return
	}

	if existingRecordResult.Error == nil {
		if reflect.DeepEqual(existingRecord.GetState(), restore.GetState()) {
			log.Debug("skip persisting, states in sync")
			return
		}
	}

	err = r.db.Save(&restore).Error

	return restore, err
}

// Delete deletes a ClusterBackupRestoresModel
func (r *RestoresRepository) Delete(restore *ClusterBackupRestoresModel) error {

	return r.db.Delete(restore).Error
}
