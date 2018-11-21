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
	"sort"
	"strings"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	commonObjectstore "github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers"
	alibabaObjectstore "github.com/banzaicloud/pipeline/pkg/providers/alibaba/objectstore"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	ossEndpointFmt = "https://oss-%s.aliyuncs.com"
)

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }

type alibabaObjectStore interface {
	commonObjectstore.ObjectStore
}

type objectStore struct {
	objectStore alibabaObjectStore

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
	force bool,
) (*objectStore, error) {
	ostore, err := getProviderObjectStore(secret, region)
	if err != nil {
		return nil, errors.Wrap(err, "could not create Alibaba object storage client")
	}

	return &objectStore{
		objectStore: ostore,
		region:      region,
		secret:      secret,
		org:         org,
		db:          db,
		force:       force,
		logger:      logger,
	}, nil
}

func getProviderObjectStore(secret *secret.SecretItemResponse, region string) (alibabaObjectStore, error) {
	// when no secrets provided build an object store with no provider client/session setup
	// eg. usage: list managed buckets
	if secret == nil {
		return alibabaObjectstore.NewPlainObjectStore()
	}

	credentials := alibabaObjectstore.Credentials{
		AccessKeyID:     secret.Values[pkgSecret.AlibabaAccessKeyId],
		SecretAccessKey: secret.Values[pkgSecret.AlibabaSecretAccessKey],
	}

	config := alibabaObjectstore.Config{
		Region: region,
	}

	ostore, err := alibabaObjectstore.New(config, credentials)
	if err != nil {
		return nil, err
	}

	return ostore, nil
}

func (os *objectStore) getLogger() logrus.FieldLogger {
	return os.logger.WithFields(logrus.Fields{
		"organization": os.org.ID,
		"region":       os.region,
	})
}

func (os *objectStore) CreateBucket(bucketName string) error {
	logger := os.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := os.newBucketSearchCriteria(bucketName)

	dbr := os.db.Where(searchCriteria).Find(bucket)

	switch dbr.Error {
	case nil:
		return emperror.WrapWith(dbr.Error, "the bucket already exists", "bucket", bucketName)
	case gorm.ErrRecordNotFound:
		// proceed to creation
	default:
		return emperror.WrapWith(dbr.Error, "failed to retrieve bucket", "bucket", bucketName)
	}

	bucket.Name = bucketName
	bucket.Organization = *os.org
	bucket.Region = os.region
	bucket.SecretRef = os.secret.ID
	bucket.Status = providers.BucketCreating

	logger.Info("creating bucket...")

	if err := os.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to save bucket", "bucket", bucketName)
	}

	if err := os.objectStore.CreateBucket(bucketName); err != nil {
		bucket.Status = providers.BucketCreateError
		bucket.StatusMsg = err.Error()
		if e := os.db.Save(bucket).Error; e != nil {
			return emperror.WrapWith(e, "failed to persist the bucket", "bucket", bucketName)
		}
		return emperror.WrapWith(err, "failed to create the bucket", "bucket", bucketName)
	}

	bucket.Status = providers.BucketCreated
	bucket.StatusMsg = "bucket successfully created"
	if err := os.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to persist the bucket", "bucket", bucketName)
	}
	logger.Info("bucket created")

	return nil
}

func (os *objectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	logger := os.getLogger()

	logger.Info("retrieving buckets from provider...")
	aliBuckets, err := os.objectStore.ListBuckets()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve buckets")
	}

	logger.Info("retrieving managed buckets...")
	var managedBuckets []ObjectStoreBucketModel

	err = os.db.Where(ObjectStoreBucketModel{OrgID: os.org.ID}).Order("name asc").Find(&managedBuckets).Error
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve managed buckets")
	}

	var bucketList []*objectstore.BucketInfo
	for _, bucket := range aliBuckets {
		// managedAlibabaBuckets must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(managedBuckets), func(i int) bool {
			return strings.Compare(managedBuckets[i].Name, bucket) >= 0
		})

		bucketInfo := &objectstore.BucketInfo{Name: bucket, Managed: false}
		if idx < len(managedBuckets) && strings.Compare(managedBuckets[idx].Name, bucket) == 0 {
			bucketInfo.Managed = true
		}

		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

func (os *objectStore) ListManagedBuckets() ([]*objectstore.BucketInfo, error) {
	logger := os.getLogger()
	logger.Debug("retrieving managed bucket list")

	var alibabaBuckets []ObjectStoreBucketModel

	if err := os.db.Where(ObjectStoreBucketModel{OrgID: os.org.ID}).Order("name asc").Find(&alibabaBuckets).Error; err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve managed buckets")
	}

	bucketList := make([]*objectstore.BucketInfo, 0)
	for _, bucket := range alibabaBuckets {
		bucketList = append(bucketList, &objectstore.BucketInfo{
			Cloud:     providers.Alibaba,
			Managed:   true,
			Name:      bucket.Name,
			Location:  bucket.Region,
			SecretRef: bucket.SecretRef,
		})
	}

	return bucketList, nil
}

func (os *objectStore) DeleteBucket(bucketName string) error {
	logger := os.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := os.newBucketSearchCriteria(bucketName)

	logger.Info("looking up the bucket...")

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

	if err := os.db.Delete(bucket).Error; err != nil {
		return os.deleteFailed(bucket, err)
	}

	return nil

}

func (os *objectStore) deleteFromProvider(bucket *ObjectStoreBucketModel) error {
	logger := os.getLogger().WithField("bucket", bucket.Name)
	logger.Info("deleting bucket on provider...")

	// todo the assumption here is, that a bucket in 'ERROR_CREATE' doesn't exist on the provider
	// todo however there might be -presumably rare cases- when a bucket in 'ERROR_DELETE' that has already been deleted on the provider
	if bucket.Status == providers.BucketCreateError {
		logger.Debug("bucket doesn't exist on provider")
		return nil
	}

	bucket.Status = providers.BucketDeleting
	if err := os.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to update bucket", "bucket", bucket.Name)
	}

	objectStore, err := getProviderObjectStore(os.secret, bucket.Region)
	if err != nil {
		return emperror.WrapWith(err, "failed to create object store", "bucket", bucket.Name)
	}

	if err := objectStore.DeleteBucket(bucket.Name); err != nil {
		return emperror.WrapWith(err, "failed to delete bucket from provider", "bucket", bucket.Name)
	}

	return nil
}

func (os *objectStore) CheckBucket(bucketName string) error {
	logger := os.getLogger().WithField("bucket", bucketName)
	logger.Info("looking up the bucket...")

	if err := os.objectStore.CheckBucket(bucketName); err != nil {
		return emperror.WrapWith(err, "failed to check the bucket", "bucket", bucketName)
	}

	return nil
}

// newBucketSearchCriteria returns the database search criteria to find managed bucket with the given name
func (os *objectStore) newBucketSearchCriteria(bucketName string) *ObjectStoreBucketModel {
	return &ObjectStoreBucketModel{
		OrgID: os.org.ID,
		Name:  bucketName,
	}
}

func (os *objectStore) deleteFailed(bucket *ObjectStoreBucketModel, reason error) error {
	bucket.Status = providers.BucketDeleteError
	bucket.StatusMsg = reason.Error()
	if err := os.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to delete bucket", "bucket", bucket.Name)
	}
	return reason
}
