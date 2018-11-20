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
	"fmt"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	ossEndpointFmt = "https://oss-%s.aliyuncs.com"
)

type ObjectStore struct {
	region string

	secret *secret.SecretItemResponse
	org    *auth.Organization
	db     *gorm.DB

	logger logrus.FieldLogger
	force  bool
}

func NewObjectStore(
	region string,
	secret *secret.SecretItemResponse,
	org *auth.Organization,
	db *gorm.DB,
	logger logrus.FieldLogger,
	force bool) *ObjectStore {

	return &ObjectStore{
		region: region,
		secret: secret,
		org:    org,
		db:     db,
		force:  force,
		logger: logger,
	}
}

func (os *ObjectStore) CreateBucket(bucketName string) error {
	log := os.getLogger(bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := os.newBucketSearchCriteria(bucketName)

	dbr := os.db.Where(searchCriteria).Find(bucket)

	switch dbr.Error {
	case nil:
		return errors.Wrapf(dbr.Error, "the bucket [%s] already exists", bucketName)
	case gorm.ErrRecordNotFound:
		// proceed to creation
	default:
		return errors.Wrapf(dbr.Error, "error while retrieving bucket [%s]", bucketName)

	}

	bucket.Name = bucketName
	bucket.Organization = *os.org
	bucket.Region = os.region
	bucket.SecretRef = os.secret.ID
	bucket.Status = providers.BucketCreating

	log.Info("persisting bucket...")

	err := os.db.Save(bucket).Error
	if err != nil {
		return errors.Wrap(err, "failed to persist bucket")
	}

	log.Info("creating OSSClient...")
	svc, err := os.createOSSClient(os.region)
	if err != nil {
		return os.createFailed(bucket, errors.Wrap(err, "failed to create OSS client"))
	}

	err = svc.CreateBucket(bucket.Name)
	if err != nil {
		return os.createFailed(bucket, errors.Wrap(err, "failed to create OSS bucket"))
	}

	log.Debug("Waiting for bucket to be created...")

	bucket.Status = providers.BucketCreated
	bucket.StatusMsg = "bucket successfully created"
	if err := os.db.Save(bucket).Error; err != nil {
		log.WithError(err).Error("could not update bucket status")
	}

	return nil
}

func (os *ObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	svc, err := os.createOSSClient(os.region)
	if err != nil {
		return nil, err
	}

	os.logger.Info("retrieving bucket list from provider...")
	buckets, err := svc.ListBuckets()
	if err != nil {
		os.logger.WithError(err).Error("failed to retrieve bucket list")
		return nil, err
	}

	os.logger.Info("retrieving managed buckets...")

	var managedBuckets []ObjectStoreBucketModel
	if err = os.queryWithOrderByDb(&ObjectStoreBucketModel{OrgID: os.org.ID}, "name asc", &managedBuckets); err != nil {
		os.logger.WithError(err).Error("failed to retrieve managed buckets")
		return nil, err
	}

	var bucketList []*objectstore.BucketInfo
	for _, bucket := range buckets.Buckets {
		// managedAlibabaBuckets must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(managedBuckets), func(i int) bool {
			return strings.Compare(managedBuckets[i].Name, bucket.Name) >= 0
		})

		bucketInfo := &objectstore.BucketInfo{Name: bucket.Name, Managed: false}
		if idx < len(managedBuckets) && strings.Compare(managedBuckets[idx].Name, bucket.Name) == 0 {
			bucketInfo.Managed = true
		}

		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

func (os *ObjectStore) ListManagedBuckets() ([]*objectstore.BucketInfo, error) {

	var managedAlibabaBuckets []ObjectStoreBucketModel

	if err := os.queryWithOrderByDb(&ObjectStoreBucketModel{OrgID: os.org.ID}, "name asc", &managedAlibabaBuckets); err != nil {
		os.logger.WithError(err).Error("retrieving managed buckets")
		return nil, err
	}

	bucketInfos := make([]*objectstore.BucketInfo, 0)
	for _, mb := range managedAlibabaBuckets {
		bucketInfos = append(bucketInfos, &objectstore.BucketInfo{
			Cloud:     providers.Alibaba,
			Managed:   true,
			Name:      mb.Name,
			Location:  mb.Region,
			SecretRef: mb.SecretRef,
		})
	}

	return bucketInfos, nil
}

func (os *ObjectStore) DeleteBucket(bucketName string) error {

	logger := os.getLogger(bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := os.newBucketSearchCriteria(bucketName)

	logger.Info("looking up the bucket")

	if err := os.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
	}

	if err := os.deleteFromProvider(bucket); err != nil {
		if !os.force {
			// if delete is not forced return here
			return os.deleteFailed(bucket, err)
		}
	}

	db := os.db.Delete(bucket)
	if db.Error != nil {
		return os.deleteFailed(bucket, db.Error)
	}

	return nil

}

func (os *ObjectStore) deleteFromProvider(bucket *ObjectStoreBucketModel) error {

	// todo the assumption here is, that a bucket in 'ERROR_CREATE' doesn't exist on the provider
	// todo however there might be -presumably rare cases- when a bucket in 'ERROR_DELETE' that has already been deleted on the provider
	if bucket.Status == providers.BucketCreateError {
		os.logger.Debug("bucket doesn't exist on provider")
		return nil
	}
	svc, err := os.createOSSClient(bucket.Region)
	if err != nil {
		os.logger.WithError(err).Error("failed to create OSSClient")
		return err
	}

	os.logger.Info("deleting from provider")
	return svc.DeleteBucket(bucket.Name)
}

func (os *ObjectStore) CheckBucket(bucketName string) error {
	svc, err := os.createOSSClient(os.region)

	if err != nil {
		log.Errorf("Creating AlibabaOSSClient failed: %s", err.Error())
		return errors.New("failed to create AlibabaOSSClient")
	}
	_, err = svc.GetBucketInfo(bucketName)
	if err != nil {
		log.Errorf("%s", err.Error())
		return err
	}

	return nil
}

// newBucketSearchCriteria returns the database search criteria to find managed bucket with the given name
func (os *ObjectStore) newBucketSearchCriteria(bucketName string) *ObjectStoreBucketModel {
	return &ObjectStoreBucketModel{
		OrgID: os.org.ID,
		Name:  bucketName,
	}
}

func (os *ObjectStore) getLogger(bucketName string) logrus.FieldLogger {
	return os.logger.WithFields(logrus.Fields{
		"organization": os.org.ID,
		"region":       os.region,
		"bucket":       bucketName,
	})
}

func (os *ObjectStore) createOSSClient(region string) (*oss.Client, error) {

	endpoint := fmt.Sprintf(ossEndpointFmt, region)
	serviceAccount := verify.CreateAlibabaCredentials(os.secret.Values)

	return oss.New(endpoint, serviceAccount.AccessKeyId, serviceAccount.AccessKeySecret)
}

func (os *ObjectStore) createFailed(b *ObjectStoreBucketModel, err error) error {
	os.logger.WithError(err).Info("create bucket failed")
	b.Status = providers.BucketCreateError
	b.StatusMsg = err.Error()
	return os.db.Save(b).Error
}

func (os *ObjectStore) deleteFailed(b *ObjectStoreBucketModel, err error) error {
	os.logger.WithError(err).Info("delete bucket failed")
	b.Status = providers.BucketDeleteError
	b.StatusMsg = err.Error()
	if err := os.db.Save(b).Error; err != nil {
		return emperror.WrapWith(err, "failed to delete bucket", "bucket", b.Name)
	}
	return emperror.WrapWith(err, "bucket", b.Name)
}

// queryWithOrderByDb queries the database using the specified searchCriteria
// and populates the returned records into result
func (os *ObjectStore) queryWithOrderByDb(searchCriteria interface{}, orderBy interface{}, result interface{}) error {
	return os.db.Where(searchCriteria).Order(orderBy).Find(result).Error
}

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }
