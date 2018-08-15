package oracle

import (
	"fmt"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	osecret "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
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
}

// NewObjectStore returns a new object store instance.
func NewObjectStore(
	location string,
	secret *secret.SecretItemResponse,
	org *auth.Organization,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *ObjectStore {
	return &ObjectStore{
		location: location,
		secret:   secret,
		org:      org,
		db:       db,
		logger:   logger,
	}
}

// CreateBucket creates an Oracle object store bucket with the given name and stores it in the database
func (o *ObjectStore) CreateBucket(name string) {

	logger := o.getLogger().WithField("bucket", name)

	oci, err := oci.NewOCI(osecret.CreateOCICredential(o.secret.Values))
	if err != nil {
		logger.Errorf("OCI client initialization failed: %s", err.Error())
		return
	}

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := o.newBucketSearchCriteria(name, o.location, oci.CompartmentOCID)
	if err := o.getBucketFromDB(searchCriteria, bucket); err != nil {
		if _, ok := err.(bucketNotFoundError); !ok {
			logger.Errorf("Error happened during getting bucket description from DB: %s", err.Error())
			return
		}
	}

	if bucket.Name == name {
		logger.Debug("Bucket already exists")
		return
	}

	err = oci.ChangeRegion(o.location)
	if err != nil {
		logger.Errorf("Changing region failed: %s", err.Error())
		return
	}

	client, err := oci.NewObjectStorageClient()
	if err != nil {
		logger.Errorf("Creating Oracle object storage client failed: %s", err.Error())
		return
	}

	bucket.Name = name
	bucket.Organization = *o.org
	bucket.CompartmentID = oci.CompartmentOCID
	bucket.Location = o.location

	if err = o.persistBucketToDB(bucket); err != nil {
		logger.Errorf("Error happened during persisting bucket description to DB: %s", err.Error())
		return
	}

	if _, err := client.CreateBucket(name); err != nil {
		logger.Errorf("Failed to create bucket: %s", err.Error())
		if e := o.deleteBucketFromDB(bucket); e != nil {
			logger.Error(e.Error())
		}
		return
	}

	logger.Infof("%s bucket created", name)
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

	err = oci.ChangeRegion(o.location)
	if err != nil {
		logger.Errorf("Changing region failed: %s", err.Error())
		return err
	}

	client, err := oci.NewObjectStorageClient()
	if err != nil {
		logger.Errorf("Creating object storage client failed: %s", err.Error())
		return err
	}

	if err := client.DeleteBucket(name); err != nil {
		return err
	}

	if err = o.deleteBucketFromDB(bucket); err != nil {
		logger.Errorf("Deleting managed Oracle bucket from database failed: %s", err.Error())
		return err
	}

	return nil
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
