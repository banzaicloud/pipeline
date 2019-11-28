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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/src/auth"
)

// BackupsService is for backups related ARK functions
type BackupsService struct {
	org        *auth.Organization
	logger     logrus.FieldLogger
	repository *BackupsRepository
}

// BackupsServiceFactory creates and returns an initialized BackupsService instance
func BackupsServiceFactory(org *auth.Organization, db *gorm.DB, logger logrus.FieldLogger) *BackupsService {

	return NewBackupsService(org, NewBackupsRepository(org, db, logger), logger)
}

// NewBackupsService creates and returns an initialized BackupsService instance
func NewBackupsService(
	org *auth.Organization,
	repository *BackupsRepository,
	logger logrus.FieldLogger,
) *BackupsService {

	return &BackupsService{
		org:        org,
		logger:     logger,
		repository: repository,
	}
}

// GetModelByName returns a ClusterBackupsModel instance by name
func (s *BackupsService) GetModelByName(name string) (*ClusterBackupsModel, error) {

	model, err := s.repository.FindOneByName(name)
	if err != nil {
		return nil, errors.Wrap(err, "could not get backup from database")
	}

	return model, nil
}

// GetModelByID returns a ClusterBackupsModel instance by ID
func (s *BackupsService) GetModelByID(id uint) (*ClusterBackupsModel, error) {

	model, err := s.repository.FindOneByID(id)
	if err != nil {
		return nil, errors.Wrap(err, "could not get backup from database")
	}

	return model, nil
}

// GetByID returns a Backup instance by ID
func (s *BackupsService) GetByID(id uint) (*api.Backup, error) {

	model, err := s.GetModelByID(id)
	if err != nil {
		return nil, err
	}

	return model.ConvertModelToEntity(), nil
}

// GetByName returns a Backup instance by name
func (s *BackupsService) GetByName(name string) (*api.Backup, error) {

	model, err := s.GetModelByName(name)
	if err != nil {
		return nil, err
	}

	return model.ConvertModelToEntity(), nil
}

// List returns Backup instances
func (s *BackupsService) List() ([]*api.Backup, error) {

	backups := make([]*api.Backup, 0)

	items, err := s.repository.Find()
	if err != nil {
		return backups, err
	}

	for _, item := range items {
		backup := item.ConvertModelToEntity()
		backups = append(backups, backup)
	}

	return backups, nil
}

// FindByPersistRequest returns a ClusterBackupsModel by PersistBackupRequest
func (s *BackupsService) FindByPersistRequest(req *api.PersistBackupRequest) (*ClusterBackupsModel, error) {

	backup, err := s.repository.FindByPersistRequest(req)
	if err != nil {
		return nil, err
	}

	return backup, nil
}

// Persist persist ClusterBackupsModel from PersistBackupRequest
func (s *BackupsService) Persist(req *api.PersistBackupRequest) (ClusterBackupsModel, error) {

	req.ExtendFromLabels()

	return s.repository.Persist(req)
}

// DeleteNonExistingBackupsByBucketAndKeys deletes ClusterBackupsModel if their ID not in keys
func (s *BackupsService) DeleteNonExistingBackupsByBucketAndKeys(bucketID uint, keys []int) error {

	return s.repository.DeleteBackupsNotInKeys(bucketID, keys)
}

// DeleteBackupsWithoutBucket deletes backups from DB if their bucket is removed
func (s *BackupsService) DeleteBackupsWithoutBucket() error {

	return s.repository.DeleteBackupsWithoutBucket()
}
