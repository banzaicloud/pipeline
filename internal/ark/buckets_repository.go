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
	"github.com/banzaicloud/pipeline/src/model"
)

// BucketsRepository descibes a repository for storing ARK backup buckets
type BucketsRepository struct {
	org    *auth.Organization
	db     *gorm.DB
	logger logrus.FieldLogger
}

// NewBucketsRepository returns a new BucketsRepository instance
func NewBucketsRepository(org *auth.Organization, db *gorm.DB, logger logrus.FieldLogger) *BucketsRepository {

	return &BucketsRepository{
		org:    org,
		db:     db,
		logger: logger,
	}
}

// Find returns ClusterBackupBucketsModel instances
func (s *BucketsRepository) Find() (buckets []*ClusterBackupBucketsModel, err error) {

	err = s.db.Where(&ClusterBackupBucketsModel{
		OrganizationID: s.org.ID,
	}).Preload("Deployment").Preload("Deployment.Cluster").Find(&buckets).Error

	return
}

// FindOneByName returns a ClusterBackupBucketsModel instance by name
func (s *BucketsRepository) FindOneByName(name string) (*ClusterBackupBucketsModel, error) {
	var bucket ClusterBackupBucketsModel

	err := s.db.Where(&ClusterBackupBucketsModel{
		OrganizationID: s.org.ID,
		BucketName:     name,
	}).Preload("Deployment").Preload("Deployment.Cluster").Find(&bucket).Error

	return &bucket, err
}

// FindOneByID returns a ClusterBackupBucketsModel instance by ID
func (s *BucketsRepository) FindOneByID(id uint) (*ClusterBackupBucketsModel, error) {
	var bucket ClusterBackupBucketsModel

	err := s.db.Where(&ClusterBackupBucketsModel{
		OrganizationID: s.org.ID,
		ID:             id,
	}).Preload("Deployment").Preload("Deployment.Cluster").Find(&bucket).Error

	return &bucket, err
}

// GetActiveDeploymentModel gets the active ARK deployment, if any
func (s *BucketsRepository) GetActiveDeploymentModel(bucket *ClusterBackupBucketsModel) (
	deployment ClusterBackupDeploymentsModel, err error) {

	err = s.db.Model(&bucket).Related(&deployment, "Deployment").Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			err = errors.Wrap(err, "error getting deployment")
		}
		return
	}

	var cluster model.ClusterModel
	err = s.db.Model(deployment).Related(&cluster, "Cluster").Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			err = errors.Wrap(err, "error getting deployment")
		}
		return
	}

	return
}

// FindOneByRequest finds a ClusterBackupBucketsModel by a FindBucketRequest
func (s *BucketsRepository) FindOneByRequest(req api.FindBucketRequest) (*ClusterBackupBucketsModel, error) {

	var bucket ClusterBackupBucketsModel

	err := s.db.Where(ClusterBackupBucketsModel{
		Cloud:      req.Cloud,
		BucketName: req.BucketName,
		Location:   req.Location,

		OrganizationID: s.org.ID,
	}).Preload("Deployment").Preload("Deployment.Cluster").First(&bucket).Error
	if err != nil {
		return nil, err
	}

	return &bucket, nil
}

// FindOneOrCreateByRequest finds or creates a ClusterBackupBucketsModel by a CreateBucketRequest
func (s *BucketsRepository) FindOneOrCreateByRequest(req *api.CreateBucketRequest) (*ClusterBackupBucketsModel, error) {

	var bucket ClusterBackupBucketsModel

	err := s.db.FirstOrInit(&bucket, ClusterBackupBucketsModel{
		Cloud:          req.Cloud,
		BucketName:     req.BucketName,
		Location:       req.Location,
		StorageAccount: req.StorageAccount,
		ResourceGroup:  req.ResourceGroup,

		OrganizationID: s.org.ID,
	}).Error
	if err != nil {
		return nil, err
	}

	bucket.SecretID = req.SecretID

	err = s.db.Save(&bucket).Error
	if err != nil {
		return nil, err
	}

	return &bucket, nil
}

// Delete deletes a ClusterBackupBucketsModel
func (s *BucketsRepository) Delete(bucket *ClusterBackupBucketsModel) error {

	err := s.IsInUse(bucket)
	if err != nil {
		return err
	}

	return s.db.Delete(bucket).Error
}

// IsInUse checks whether a ClusterBackupBucketsModel is used in an active ARK deployment
func (s *BucketsRepository) IsInUse(bucket *ClusterBackupBucketsModel) error {

	_, err := s.GetActiveDeploymentModel(bucket)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	return errors.New("bucket is used in a deployment")
}

// UpdateStatus updates the status of a ClusterBackupBucketsModel
func (s *BucketsRepository) UpdateStatus(bucket *ClusterBackupBucketsModel, status, message string) error {

	bucket.Status = status
	bucket.StatusMessage = message

	return s.db.Save(&bucket).Error
}
