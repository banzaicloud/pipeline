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

package amazon

import (
	"sort"
	"strings"

	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	commonObjectstore "github.com/banzaicloud/pipeline/pkg/objectstore"
	amazonObjectstore "github.com/banzaicloud/pipeline/pkg/providers/amazon/objectstore"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }

type amazonObjectStore interface {
	commonObjectstore.ObjectStore
	GetRegion(bucket string) (string, error)
}

// objectStore stores all required parameters for bucket creation.
type objectStore struct {
	objectStore amazonObjectStore

	region string
	secret *secret.SecretItemResponse

	org *auth.Organization

	db     *gorm.DB
	logger logrus.FieldLogger

	force bool
}

// NewObjectStore returns a new object store instance.
func NewObjectStore(
	region string,
	secret *secret.SecretItemResponse,
	org *auth.Organization,
	db *gorm.DB,
	logger logrus.FieldLogger,
	force bool,
) (*objectStore, error) {
	ostore, err := getProviderObjectStore(secret, region)
	if err != nil {
		errors.Wrap(err, "could not create AWS object storage client")
	}

	return &objectStore{
		objectStore: ostore,
		region:      region,
		secret:      secret,
		org:         org,
		db:          db,
		logger:      logger,
		force:       force,
	}, nil
}

func getProviderObjectStore(secret *secret.SecretItemResponse, region string) (amazonObjectStore, error) {
	// when no secrets provided build an object store with no provider client/session setup
	// eg. usage: list managed buckets
	if secret == nil {
		return amazonObjectstore.NewPlainObjectStore()
	}

	credentials := amazonObjectstore.Credentials{
		AccessKeyID:     secret.Values[pkgSecret.AwsAccessKeyId],
		SecretAccessKey: secret.Values[pkgSecret.AwsSecretAccessKey],
	}

	config := amazonObjectstore.Config{
		Region: region,
		Opts: []amazonObjectstore.Option{
			amazonObjectstore.WaitForCompletion(true),
		},
	}

	ostore, err := amazonObjectstore.New(config, credentials)
	if err != nil {
		return nil, err
	}

	return ostore, nil
}

func (s *objectStore) getLogger() logrus.FieldLogger {
	var sId string
	if s.secret == nil {
		sId = ""
	} else {
		sId = s.secret.ID
	}

	return s.logger.WithFields(logrus.Fields{
		"organization": s.org.ID,
		"secret":       sId,
		"region":       s.region,
	})
}

// CreateBucket creates an S3 bucket with the provided name.
func (s *objectStore) CreateBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	dbr := s.db.Where(searchCriteria).Find(bucket)

	if dbr.Error != nil {
		if dbr.Error != gorm.ErrRecordNotFound {
			return emperror.WrapWith(dbr.Error, "failed to retrieve bucket", "bucket", bucketName)
		}
	} else {
		return emperror.WrapWith(errors.New("bucket already exists"), "bucket", bucketName)
	}

	bucket.Name = bucketName
	bucket.Organization = *s.org
	bucket.Region = s.region

	bucket.SecretRef = s.secret.ID
	bucket.Status = providers.BucketCreating

	if e := s.db.Save(bucket).Error; e != nil {
		return emperror.WrapWith(e, "failed to persist the bucket", "bucket", bucketName)
	}

	logger.Info("creating bucket...")

	if err := s.objectStore.CreateBucket(bucketName); err != nil {
		bucket.Status = providers.BucketCreateError
		bucket.StatusMsg = err.Error()
		if e := s.db.Save(bucket).Error; e != nil {
			return emperror.WrapWith(e, "failed to persist the bucket", "bucket", bucketName)
		}
		return emperror.WrapWith(err, "failed to create the bucket", "bucket", bucketName)
	}

	bucket.Status = providers.BucketCreated
	if e := s.db.Save(bucket).Error; e != nil {
		return emperror.WrapWith(e, "failed to persist the bucket", "bucket", bucketName)
	}
	logger.Info("bucket created")

	return nil
}

// DeleteBucket deletes the S3 bucket identified by the specified name
// provided the storage container is of 'managed' type.
func (s *objectStore) DeleteBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	logger.Info("looking up the bucket...")

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
	}

	if err := s.deleteFromProvider(bucket); err != nil {
		if !s.force {
			// if delete is not forced return here
			return err
		}
	}

	db := s.db.Delete(bucket)
	if db.Error != nil {
		return s.deleteFailed(bucket, db.Error)
	}

	return nil
}

func (s *objectStore) deleteFromProvider(bucket *ObjectStoreBucketModel) error {
	logger := s.getLogger().WithField("bucket", bucket.Name)
	logger.Info("deleting bucket on provider...")

	// todo the assumption here is, that a bucket in 'ERROR_CREATE' doesn't exist on the provider
	// todo however there might be -presumably rare cases- when a bucket in 'ERROR_DELETE' that has already been deleted on the provider
	if bucket.Status == providers.BucketCreateError {
		logger.Debug("bucket doesn't exist on provider")
		return nil
	}

	bucket.Status = providers.BucketDeleting
	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to update bucket", "bucket", bucket.Name)
	}

	objectStore, err := getProviderObjectStore(s.secret, bucket.Region)
	if err != nil {
		return s.deleteFailed(bucket, emperror.WrapWith(err, "failed create AWS object storage client", "bucket", bucket.Name))
	}

	if err := objectStore.DeleteBucket(bucket.Name); err != nil {
		return s.deleteFailed(bucket, err)
	}

	if err := s.db.Delete(bucket).Error; err != nil {
		return s.deleteFailed(bucket, err)
	}

	return nil

}

func (s *objectStore) deleteFailed(bucket *ObjectStoreBucketModel, reason error) error {
	bucket.Status = providers.BucketDeleteError
	bucket.StatusMsg = reason.Error()
	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to delete bucket", "bucket", bucket.Name)
	}
	return emperror.WrapWith(reason, "bucket", bucket.Name)
}

// CheckBucket checks the status of the given S3 bucket.
func (s *objectStore) CheckBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)
	logger.Info("looking up the bucket...")

	if err := s.objectStore.CheckBucket(bucketName); err != nil {
		return emperror.WrapWith(err, "failed to check the bucket", "bucket", bucketName)
	}

	return nil
}

// ListBuckets returns a list of S3 buckets that can be accessed with the credentials
// referenced by the secret field. S3 buckets that were created by a user in the current
// org are marked as 'managed'.
func (s *objectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	logger := s.getLogger()

	logger.Info("retrieving buckets from provider...")
	s3Buckets, err := s.objectStore.ListBuckets()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve buckets")
	}

	logger.Info("retrieving managed buckets...")
	var managedBuckets []*ObjectStoreBucketModel

	err = s.db.Where(&ObjectStoreBucketModel{OrganizationID: s.org.ID}).Order("name asc").Find(&managedBuckets).Error
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve managed buckets")
	}

	var bucketList []*objectstore.BucketInfo
	for _, bucket := range s3Buckets {
		// managedBuckets must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(managedBuckets), func(i int) bool {
			return strings.Compare(managedBuckets[i].Name, bucket) >= 0
		})

		bucketInfo := &objectstore.BucketInfo{Name: bucket, Managed: false}
		if idx < len(managedBuckets) && strings.Compare(managedBuckets[idx].Name, bucket) == 0 {
			bucketInfo.Managed = true
		}

		if region, err := s.objectStore.GetRegion(bucket); err == nil {
			bucketInfo.Location = region
			bucketList = append(bucketList, bucketInfo)
		} else {
			if objectstore.IsNotFoundError(err) {
				logger.WithField("bucket", bucket).WithError(err).Warn("skipping bucket from the list")
			} else {
				return nil, emperror.WrapWith(err, "failed to retrieve region for bucket", "bucket", bucket)
			}
		}
	}

	return bucketList, nil
}

func (s *objectStore) ListManagedBuckets() ([]*objectstore.BucketInfo, error) {
	logger := s.getLogger()
	logger.Debug("retrieving managed bucket list")

	var amazonBuckets []*ObjectStoreBucketModel

	if err := s.db.Where(&ObjectStoreBucketModel{OrganizationID: s.org.ID}).Order("name asc").Find(&amazonBuckets).Error; err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve managed buckets")
	}

	bucketList := make([]*objectstore.BucketInfo, 0)
	for _, bucket := range amazonBuckets {
		bucketInfo := &objectstore.BucketInfo{Name: bucket.Name, Managed: true}
		bucketInfo.Location = bucket.Region
		bucketInfo.SecretRef = bucket.SecretRef
		bucketInfo.Cloud = providers.Amazon
		bucketInfo.Status = bucket.Status
		bucketInfo.StatusMsg = bucket.StatusMsg
		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

// searchCriteria returns the database search criteria to find bucket with the given name.
func (s *objectStore) searchCriteria(bucketName string) *ObjectStoreBucketModel {
	return &ObjectStoreBucketModel{
		OrganizationID: s.org.ID,
		Name:           bucketName,
	}
}
