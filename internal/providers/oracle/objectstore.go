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
	"fmt"

	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	osecret "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
	"github.com/banzaicloud/pipeline/secret"
)

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }

// ObjectStore stores all required parameters for container creation
type ObjectStore struct {
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
) *ObjectStore {
	return &ObjectStore{
		location: location,
		secret:   secret,
		org:      org,
		db:       db,
		logger:   logger,
		force:    force,
	}
}

// CreateBucket creates an Oracle object store bucket with the given name and stores it in the database
func (o *ObjectStore) CreateBucket(name string) error {
	logger := o.getLogger().WithField("bucket", name)

	oci, err := oci.NewOCI(osecret.CreateOCICredential(o.secret.Values))
	if err != nil {
		return errors.Wrap(err, "OCI client initialization failed")
	}

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := o.newBucketSearchCriteria(name, o.location, oci.CompartmentOCID)
	if err := o.getBucketFromDB(searchCriteria, bucket); err != nil {
		if _, ok := err.(bucketNotFoundError); !ok {
			return errors.Wrap(err, "Error happened during getting bucket description from DB")
		}
	}

	if bucket.Name == name {
		logger.Debug("Bucket already exists")

		return nil
	}

	err = oci.ChangeRegion(o.location)
	if err != nil {
		return errors.Wrap(err, "changing region failed")
	}

	client, err := oci.NewObjectStorageClient()
	if err != nil {
		return errors.Wrap(err, "creating Oracle object storage client failed")
	}

	bucket.Name = name
	bucket.Organization = *o.org
	bucket.CompartmentID = oci.CompartmentOCID
	bucket.Location = o.location
	bucket.SecretRef = o.secret.ID
	bucket.Status = providers.BucketCreating

	if err = o.persistBucketToDB(bucket); err != nil {
		return errors.Wrap(err, "error happened during persisting bucket description to DB")
	}

	if _, err := client.CreateBucket(name); err != nil {
		bucket.Status = providers.BucketCreateError
		bucket.StatusMsg = err.Error()
		if e := o.persistBucketToDB(bucket); e != nil {
			logger.Error(e.Error())
		}

		return errors.Wrap(err, "failed to create bucket")
	}

	bucket.Status = providers.BucketCreated
	if e := o.persistBucketToDB(bucket); e != nil {
		logger.Error(e.Error())
	}

	logger.Infof("%s bucket created", name)

	return nil
}

// ListBuckets list all buckets in Oracle object store
func (o *ObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {

	logger := o.getLogger()

	logger.Debug("Initializing OCI client")
	oci, err := oci.NewOCI(osecret.CreateOCICredential(o.secret.Values))
	if err != nil {
		logger.Errorf("OCI client initialization failed: %s", err.Error())
		return nil, err
	}
	identity, err := oci.NewIdentityClient()
	if err != nil {
		logger.Errorf("Creating Oracle object identity client failed: %s", err.Error())
		return nil, err
	}
	regions, err := identity.GetSubscribedRegionNames()
	if err != nil {
		return nil, err
	}

	var bucketList []*objectstore.BucketInfo
	for _, region := range regions {
		err := oci.ChangeRegion(region)
		if err != nil {
			logger.Errorf("Changing region failed: %s", err.Error())
			return nil, err
		}

		client, err := oci.NewObjectStorageClient()
		if err != nil {
			logger.Errorf("Creating Oracle object storage client failed: %s", err.Error())
			return nil, err
		}

		logger.WithField("location", region).Debug("Retrieving bucket list")
		buckets, err := client.GetBuckets()
		if err != nil {
			logger.WithField("location", region).Errorf("Retrieving bucket list failed: %s", err.Error())
			return nil, err
		}

		for _, bucket := range buckets {
			bucketInfo := &objectstore.BucketInfo{Name: *bucket.Name, Location: region, Managed: false}
			bucketList = append(bucketList, bucketInfo)
		}
	}

	o.markManagedBuckets(bucketList, oci.CompartmentOCID)

	return bucketList, nil
}

func (o *ObjectStore) ListManagedBuckets() ([]*objectstore.BucketInfo, error) {

	var objectStores []ObjectStoreBucketModel
	err := o.db.Where(&ObjectStoreBucketModel{OrgID: o.org.ID}).Order("name asc").Find(&objectStores).Error
	if err != nil {
		return nil, fmt.Errorf("retrieving managed buckets failed: %s", err.Error())
	}

	bucketList := make([]*objectstore.BucketInfo, 0)
	for _, bucket := range objectStores {
		bucketInfo := &objectstore.BucketInfo{Name: bucket.Name, Managed: true}
		bucketInfo.Location = bucket.Location
		bucketInfo.Cloud = providers.Oracle
		bucketInfo.SecretRef = bucket.SecretRef
		bucketInfo.Status = bucket.Status
		bucketInfo.StatusMsg = bucket.StatusMsg
		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil

}

// DeleteBucket deletes the managed bucket with the given name from Oracle object store
func (o *ObjectStore) DeleteBucket(name string) error {

	logger := o.getLogger().WithField("bucket", name)

	logger.Debug("Initializing OCI client")
	oci, err := oci.NewOCI(osecret.CreateOCICredential(o.secret.Values))
	if err != nil {
		logger.Errorf("OCI client initialization failed: %s", err.Error())
		return err
	}

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := o.newBucketSearchCriteria(name, o.location, oci.CompartmentOCID)
	if err := o.getBucketFromDB(searchCriteria, bucket); err != nil {
		return err
	}

	if err := o.deleteFromProvider(bucket); err != nil {
		if !o.force {
			// if delete is not forced return here
			return err
		}
	}

	if err := o.deleteBucketFromDB(bucket); err != nil {
		return o.deleteFailed(bucket, err)
	}

	return nil
}

func (o *ObjectStore) deleteFromProvider(bucket *ObjectStoreBucketModel) error {

	logger := o.getLogger().WithField("bucket", bucket.Name)
	// todo the assumption here is, that a bucket in 'ERROR_CREATE' doesn't exist on the provider
	// todo however there might be -presumably rare cases- when a bucket in 'ERROR_DELETE' that has already been deleted on the provider
	if bucket.Status == providers.BucketCreateError {
		logger.Debug("bucket doesn't exist on provider")
		return nil
	}

	bucket.Status = providers.BucketDeleting
	if err := o.persistBucketToDB(bucket); err != nil {
		return errors.Wrap(err, "could not persist bucket state")
	}

	logger.Debug("Initializing OCI client")
	oci, err := oci.NewOCI(osecret.CreateOCICredential(o.secret.Values))
	if err != nil {
		logger.Errorf("OCI client initialization failed: %s", err.Error())
		return o.deleteFailed(bucket, err)
	}

	err = oci.ChangeRegion(o.location)
	if err != nil {
		logger.Errorf("Changing region failed: %s", err.Error())
		return o.deleteFailed(bucket, err)
	}

	client, err := oci.NewObjectStorageClient()
	if err != nil {
		logger.Errorf("Creating object storage client failed: %s", err.Error())
		return o.deleteFailed(bucket, err)
	}

	if err := client.DeleteBucket(bucket.Name); err != nil {
		return o.deleteFailed(bucket, err)
	}

	if err = o.deleteBucketFromDB(bucket); err != nil {
		logger.Errorf("Deleting managed Oracle bucket from database failed: %s", err.Error())
		return o.deleteFailed(bucket, err)
	}

	return nil
}

func (o *ObjectStore) deleteFailed(bucket *ObjectStoreBucketModel, reason error) error {
	bucket.Status = providers.BucketDeleteError
	bucket.StatusMsg = reason.Error()
	db := o.db.Save(bucket)
	if db.Error != nil {
		return emperror.With(db.Error, "could not delete bucket", bucket.Name)
	}
	return reason
}

// CheckBucket check the status of the given Oracle object store bucket
func (o *ObjectStore) CheckBucket(name string) error {

	logger := o.getLogger().WithField("bucket", name)

	logger.Debug("Initializing OCI client")
	oci, err := oci.NewOCI(osecret.CreateOCICredential(o.secret.Values))
	if err != nil {
		logger.Errorf("OCI client initialization failed: %s", err.Error())
		return err
	}

	err = oci.ChangeRegion(o.location)
	if err != nil {
		logger.Errorf("Changing region failed: %s", err.Error())
		return err
	}

	client, err := oci.NewObjectStorageClient()
	if err != nil {
		logger.Errorf("Creating Oracle object storage client failed: %s", err.Error())
		return err
	}

	logger.Debug("Getting bucket")
	_, err = client.GetBucket(name)
	if err, ok := err.(common.ServiceError); ok {
		if err.GetHTTPStatusCode() == 404 {
			return bucketNotFoundError{}
		}
	}

	return err
}

// markManagedBucket marks buckets exists in the database to 'managed'
func (o *ObjectStore) markManagedBuckets(buckets []*objectstore.BucketInfo, compartmentID string) error {

	logger := o.getLogger()

	// get managed buckets from database
	managedBuckets, err := o.getBucketsFromDB()
	if err != nil {
		return err
	}

	// make map for search
	mBuckets := make(map[string]string, 0)
	for _, mBucket := range managedBuckets {
		key := fmt.Sprintf("%s-%s-%s", mBucket.Name, mBucket.Location, mBucket.CompartmentID)
		mBuckets[key] = key
	}

	logger.Debug("Marking managed buckets")
	for _, bucketInfo := range buckets {
		key := fmt.Sprintf("%s-%s-%s", bucketInfo.Name, bucketInfo.Location, compartmentID)
		if mBuckets[key] == key {
			bucketInfo.Managed = true
		}
	}

	return nil
}

// getBucketsFromDB gives back object store buckets from DB
func (o *ObjectStore) getBucketsFromDB() ([]ObjectStoreBucketModel, error) {

	logger := o.getLogger()
	logger.Debug("Retrieving managed buckets from DB")

	var buckets []ObjectStoreBucketModel
	if err := o.db.Where(&ObjectStoreBucketModel{OrgID: o.org.ID}).Order("name asc, location asc").Find(&buckets).Error; err != nil {
		logger.Errorf("Retrieving managed buckets failed: %s", err.Error())
		return nil, err
	}

	return buckets, nil
}

// getBucketFromDB looks up the managed bucket record in the database based on the specified searchCriteria
// If no db record is found than returns with bucketNotFoundError
func (o *ObjectStore) getBucketFromDB(searchCriteria *ObjectStoreBucketModel, managedBucket *ObjectStoreBucketModel) error {

	logger := o.getLogger()
	logger.Debug("Searching for managed bucket in DB")

	if err := o.db.Where(searchCriteria).Find(managedBucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}

		return err
	}

	return nil
}

// newBucketSearchCriteria returns the database search criteria to find a bucket in db
func (o *ObjectStore) newBucketSearchCriteria(bucketName string, location string, compartmentID string) *ObjectStoreBucketModel {
	return &ObjectStoreBucketModel{
		OrgID:         o.org.ID,
		Name:          bucketName,
		CompartmentID: compartmentID,
		Location:      location,
	}
}

// persistBucketToDB persists bucket into DB
func (o *ObjectStore) persistBucketToDB(m *ObjectStoreBucketModel) error {
	logger := o.getLogger().WithField("bucket", m.Name)
	logger.Debug("Persisting to DB")

	return o.db.Save(m).Error
}

// deleteBucketFromDB deletes a bucket from DB
func (o *ObjectStore) deleteBucketFromDB(m *ObjectStoreBucketModel) error {
	logger := o.getLogger().WithField("bucket", m.Name)
	logger.Debug("Deleting from DB")

	return o.db.Delete(m).Error
}

// getLogger initializes and gives back a logger instance with some basic fields
func (o *ObjectStore) getLogger() logrus.FieldLogger {

	fields := logrus.Fields{
		"organization": o.org.ID,
	}

	if o.location != "" {
		fields["location"] = o.location
	}

	return o.logger.WithFields(fields)
}
