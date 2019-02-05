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

package oracle

import (
	"sort"
	"strings"

	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	commonObjectstore "github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers"
	oracleObjectstore "github.com/banzaicloud/pipeline/pkg/providers/oracle/objectstore"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }

type oracleObjectStore interface {
	commonObjectstore.ObjectStore
}

// ObjectStore stores all required parameters for container creation
type ObjectStore struct {
	objectStore oracleObjectStore

	location string
	secret   *secret.SecretItemResponse

	org *auth.Organization

	db     *gorm.DB
	logger logrus.FieldLogger
	force  bool
}

// NewObjectStore returns a new object store instance.
func NewObjectStore(
	location string,
	secret *secret.SecretItemResponse,
	org *auth.Organization,
	db *gorm.DB,
	logger logrus.FieldLogger,
	force bool,
) (*ObjectStore, error) {
	ostore, err := getProviderObjectStore(secret, location)
	if err != nil {
		return nil, errors.Wrap(err, "could not create Oracle object storage client")
	}

	return &ObjectStore{
		objectStore: ostore,
		location:    location,
		secret:      secret,
		org:         org,
		db:          db,
		logger:      logger,
		force:       force,
	}, nil
}

func getProviderObjectStore(secret *secret.SecretItemResponse, location string) (oracleObjectStore, error) {
	// when no secrets provided build an object store with no provider client/session setup
	// eg. usage: list managed buckets
	if secret == nil {
		return oracleObjectstore.NewPlainObjectStore()
	}

	credentials := oracleObjectstore.Credentials{
		UserOCID:          secret.Values[pkgSecret.OracleUserOCID],
		APIKey:            secret.Values[pkgSecret.OracleAPIKey],
		APIKeyFingerprint: secret.Values[pkgSecret.OracleAPIKeyFingerprint],
		CompartmentOCID:   secret.Values[pkgSecret.OracleCompartmentOCID],
		TenancyOCID:       secret.Values[pkgSecret.OracleTenancyOCID],
	}

	config := oracleObjectstore.Config{
		Region: secret.Values[pkgSecret.OracleRegion],
	}
	if location != "" {
		config.Region = location
	}

	ostore, err := oracleObjectstore.New(config, credentials)
	if err != nil {
		return nil, err
	}

	return ostore, nil
}

// getLogger initializes and gives back a logger instance with some basic fields
func (o *ObjectStore) getLogger() logrus.FieldLogger {
	var sId string
	if o.secret == nil {
		sId = ""
	} else {
		sId = o.secret.ID
	}

	return o.logger.WithFields(logrus.Fields{
		"organization": o.org.ID,
		"secret":       sId,
		"location":     o.location,
	})
}

// CreateBucket creates an Oracle object store bucket with the given name and stores it in the database
func (o *ObjectStore) CreateBucket(bucketName string) error {
	logger := o.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := o.newBucketSearchCriteria(bucketName)

	dbr := o.db.Where(searchCriteria).Find(bucket)

	switch dbr.Error {
	case nil:
		return emperror.WrapWith(dbr.Error, "the bucket already exists", "bucket", bucketName)
	case gorm.ErrRecordNotFound:
		// proceed to creation
	default:
		return emperror.WrapWith(dbr.Error, "failed to retrieve bucket", "bucket", bucketName)
	}

	bucket.Name = bucketName
	bucket.Organization = *o.org
	bucket.CompartmentID = o.secret.Values[pkgSecret.OracleCompartmentOCID]
	bucket.Location = o.location
	bucket.SecretRef = o.secret.ID
	bucket.Status = providers.BucketCreating

	logger.Info("creating bucket...")

	if err := o.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to save bucket", "bucket", bucketName)
	}

	if err := o.objectStore.CreateBucket(bucketName); err != nil {
		return o.createFailed(bucket, emperror.Wrap(err, "failed to create the bucket"))
	}

	bucket.Status = providers.BucketCreated
	bucket.StatusMsg = "bucket successfully created"
	if err := o.db.Save(bucket).Error; err != nil {
		return o.createFailed(bucket, emperror.Wrap(err, "failed to save bucket"))
	}
	logger.Info("bucket created")

	return nil
}

func (o *ObjectStore) createFailed(bucket *ObjectStoreBucketModel, err error) error {
	bucket.Status = providers.BucketCreateError
	bucket.StatusMsg = err.Error()

	if e := o.db.Save(bucket).Error; e != nil {
		return emperror.WrapWith(e, "failed to save bucket", "bucket", bucket.Name)
	}

	return emperror.With(err, "bucket", bucket.Name)
}

// ListBuckets list all buckets in Oracle object store
func (o *ObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	logger := o.getLogger()

	logger.Info("retrieving buckets from provider...")
	oracleBuckets, err := o.objectStore.ListBuckets()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve buckets")
	}

	logger.Info("retrieving managed buckets...")
	var managedBuckets []ObjectStoreBucketModel

	err = o.db.Where(ObjectStoreBucketModel{OrgID: o.org.ID}).Order("name asc").Find(&managedBuckets).Error
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve managed buckets")
	}

	var bucketList []*objectstore.BucketInfo
	for _, bucket := range oracleBuckets {
		// managedBuckets must be sorted in order to be able to perform binary search on it
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

func (o *ObjectStore) ListManagedBuckets() ([]*objectstore.BucketInfo, error) {
	logger := o.getLogger()
	logger.Debug("retrieving managed bucket list")

	var oracleBuckets []ObjectStoreBucketModel

	if err := o.db.Where(ObjectStoreBucketModel{OrgID: o.org.ID}).Order("name asc").Find(&oracleBuckets).Error; err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve managed buckets")
	}

	bucketList := make([]*objectstore.BucketInfo, 0)
	for _, bucket := range oracleBuckets {
		bucketList = append(bucketList, &objectstore.BucketInfo{
			Name:      bucket.Name,
			Managed:   true,
			Location:  bucket.Location,
			SecretRef: bucket.SecretRef,
			Cloud:     providers.Oracle,
			Status:    bucket.Status,
			StatusMsg: bucket.StatusMsg,
		})
	}

	return bucketList, nil
}

// DeleteBucket deletes the managed bucket with the given name from Oracle object store
func (o *ObjectStore) DeleteBucket(bucketName string) error {
	logger := o.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := o.newBucketSearchCriteria(bucketName)

	logger.Info("looking up the bucket...")

	if err := o.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
		return emperror.WrapWith(err, "failed to lookup", "bucket", bucketName)
	}

	if err := o.deleteFromProvider(bucket); err != nil {
		if !o.force {
			// if delete is not forced return here
			return o.deleteFailed(bucket, err)
		}
	}

	if err := o.db.Delete(bucket).Error; err != nil {
		return o.deleteFailed(bucket, err)
	}

	return nil
}

func (o *ObjectStore) deleteFromProvider(bucket *ObjectStoreBucketModel) error {
	logger := o.getLogger().WithField("bucket", bucket.Name)
	logger.Info("deleting bucket on provider...")

	// todo the assumption here is, that a bucket in 'ERROR_CREATE' doesn't exist on the provider
	// todo however there might be -presumably rare cases- when a bucket in 'ERROR_DELETE' that has already been deleted on the provider
	if bucket.Status == providers.BucketCreateError {
		logger.Debug("bucket doesn't exist on provider")
		return nil
	}

	bucket.Status = providers.BucketDeleting
	if err := o.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to update bucket", "bucket", bucket.Name)
	}

	objectStore, err := getProviderObjectStore(o.secret, bucket.Location)
	if err != nil {
		return emperror.WrapWith(err, "failed to create object store", "bucket", bucket.Name)
	}

	if err := objectStore.DeleteBucket(bucket.Name); err != nil {
		return emperror.WrapWith(err, "failed to delete bucket from provider", "bucket", bucket.Name)
	}

	return nil
}

func (o *ObjectStore) deleteFailed(bucket *ObjectStoreBucketModel, reason error) error {
	bucket.Status = providers.BucketDeleteError
	bucket.StatusMsg = reason.Error()
	if err := o.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to save bucket", "bucket", bucket.Name)
	}
	return reason
}

// CheckBucket check the status of the given Oracle object store bucket
func (o *ObjectStore) CheckBucket(bucketName string) error {
	logger := o.getLogger().WithField("bucket", bucketName)
	logger.Info("looking up the bucket...")

	if err := o.objectStore.CheckBucket(bucketName); err != nil {
		return emperror.WrapWith(err, "failed to check the bucket", "bucket", bucketName)
	}

	return nil
}

// newBucketSearchCriteria returns the database search criteria to find a bucket in db
func (o *ObjectStore) newBucketSearchCriteria(bucketName string) *ObjectStoreBucketModel {
	return &ObjectStoreBucketModel{
		OrgID:         o.org.ID,
		Name:          bucketName,
		CompartmentID: o.secret.Values[pkgSecret.OracleCompartmentOCID],
		Location:      o.location,
	}
}
