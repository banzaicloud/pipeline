package objectstore

import (
	"errors"
	"fmt"

	pipelineAuth "github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/pkg/objectstore"
	model "github.com/banzaicloud/pipeline/pkg/providers/oracle/model/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	verify "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
	"github.com/banzaicloud/pipeline/secret"
)

// OCIObjectStore stores all required parameters for container creation
type OCIObjectStore struct {
	secret        *secret.SecretItemResponse
	org           *pipelineAuth.Organization
	compartmentID string
	location      string
}

// CreateBucket creates an Oracle object store bucket with the given name
func (o *OCIObjectStore) CreateBucket(name string) {

	oci, err := oci.NewOCI(verify.CreateOCICredential(o.secret.Values))
	if err != nil {
		log.Errorf("Bucket creation error: %s", err)
	}

	managedBucket := &model.ManagedOracleBucket{}
	searchCriteria := o.newManagedBucketSearchCriteria(name, o.location, oci.CompartmentOCID)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		switch err.(type) {
		case ManagedBucketNotFoundError:
		default:
			log.Errorf("Error happened during getting bucket description from DB %s", err.Error())
			return
		}
	}

	err = oci.ChangeRegion(o.location)
	if err != nil {
		log.Errorf("Bucket creation error: %s", err)
		return
	}

	client, err := oci.NewObjectStorageClient()
	if err != nil {
		log.Errorf("Bucket creation error: %s", err)
		return
	}

	managedBucket.Name = name
	managedBucket.Organization = *o.org
	managedBucket.CompartmentID = oci.CompartmentOCID
	managedBucket.Location = o.location

	err = persistToDb(managedBucket)
	if err != nil {
		log.Errorf("Error happened during persisting bucket description to DB")
		return
	}
	if _, err := client.CreateBucket(name); err != nil {
		log.Errorf("Failed to create bucket: %s", err.Error())
		if e := deleteFromDbByPK(managedBucket); e != nil {
			log.Error(e.Error())
		}
		return
	}
	log.Infof("%s bucket created", name)
}

// ListBuckets list all buckets in Oracle object store
func (o *OCIObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {

	log.Info("Getting credentials")
	oci, err := oci.NewOCI(verify.CreateOCICredential(o.secret.Values))
	if err != nil {
		log.Errorf("Getting credentials failed: %s", err.Error())
		return nil, err
	}

	identity, err := oci.NewIdentityClient()
	if err != nil {
		log.Errorf("Creating Oracle object identity client failed: %s", err.Error())
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
			return nil, err
		}

		client, err := oci.NewObjectStorageClient()
		if err != nil {
			log.Errorf("Creating Oracle object storage client failed: %s", err.Error())
			return nil, err
		}

		log.Infof("Retrieving bucket list from Oracle/%s", region)
		buckets, err := client.GetBuckets()
		if err != nil {
			return nil, err
		}

		for _, bucket := range buckets {
			bucketInfo := &objectstore.BucketInfo{Name: *bucket.Name, Location: region, Managed: false}
			bucketList = append(bucketList, bucketInfo)
		}
	}

	o.MarkManagedBuckets(bucketList, oci.CompartmentOCID)

	return bucketList, nil
}

// DeleteBucket deletes the managed bucket with the given name from Oracle object store
func (o *OCIObjectStore) DeleteBucket(name string) error {

	log.Info("Getting credentials")
	oci, err := oci.NewOCI(verify.CreateOCICredential(o.secret.Values))
	if err != nil {
		log.Errorf("Getting credentials failed: %s", err.Error())
		return err
	}

	managedBucket := &model.ManagedOracleBucket{}
	searchCriteria := o.newManagedBucketSearchCriteria(name, o.location, oci.CompartmentOCID)

	log.Infof("Looking up managed bucket: name=%s", name)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		return err
	}

	err = oci.ChangeRegion(o.location)
	if err != nil {
		log.Errorf("Chaning Oracle region failed: %s", err.Error())
		return err
	}

	client, err := oci.NewObjectStorageClient()
	if err != nil {
		log.Errorf("Creating Oracle object storage client failed: %s", err.Error())
		return err
	}

	if err := client.DeleteBucket(name); err != nil {
		return err
	}

	if err = deleteFromDbByPK(managedBucket); err != nil {
		log.Errorf("Deleting managed Oracle bucket from database failed: %s", err.Error())
		return err
	}

	return nil
}

// CheckBucket check the status of the given Oracle object store bucket
func (o *OCIObjectStore) CheckBucket(name string) error {

	log.Info("Getting credentials")
	oci, err := oci.NewOCI(verify.CreateOCICredential(o.secret.Values))
	if err != nil {
		log.Errorf("Getting credentials failed: %s", err.Error())
		return err
	}

	managedBucket := &model.ManagedOracleBucket{}
	searchCriteria := o.newManagedBucketSearchCriteria(name, o.location, oci.CompartmentOCID)

	log.Infof("Looking up managed bucket: name=%s", name)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		log.Error(err)
		return err
	}

	err = oci.ChangeRegion(managedBucket.Location)
	if err != nil {
		log.Errorf("Bucket creation error: %s", err)
	}

	client, err := oci.NewObjectStorageClient()
	if err != nil {
		log.Errorf("Creating Oracle object storage client failed: %s", err.Error())
		return err
	}

	log.Infof("Getting bucket with name: %s", name)
	_, err = client.GetBucket(name)

	return err
}

// WithResourceGroup always return "not implemented" error
func (o *OCIObjectStore) WithResourceGroup(name string) error {
	return errors.New("not implemented")
}

// WithStorageAccount always return "not implemented" error
func (o *OCIObjectStore) WithStorageAccount(name string) error {
	return errors.New("not implemented")
}

// WithRegion updates the region.
func (o *OCIObjectStore) WithRegion(region string) error {
	o.location = region
	return nil
}

// newManagedBucketSearchCriteria returns the database search criteria to find managed bucket with the given name
func (o *OCIObjectStore) newManagedBucketSearchCriteria(bucketName string, location string, compartmentID string) *model.ManagedOracleBucket {
	return &model.ManagedOracleBucket{
		OrgID:         o.org.ID,
		Name:          bucketName,
		CompartmentID: compartmentID,
		Location:      location,
	}
}

// MarkManagedBuckets marks buckets by name to 'managed'
func (o *OCIObjectStore) MarkManagedBuckets(buckets []*objectstore.BucketInfo, compartmentID string) error {

	// get managed buckets from database
	managedBuckets, err := o.GetManagedBuckets()
	if err != nil {
		return err
	}

	// make map for search
	mBuckets := make(map[string]string, 0)
	for _, mBucket := range managedBuckets {
		key := fmt.Sprintf("%s-%s-%s", mBucket.Name, mBucket.Location, mBucket.CompartmentID)
		mBuckets[key] = key
	}

	log.Infof("Marking managed buckets by name")
	for _, bucketInfo := range buckets {
		key := fmt.Sprintf("%s-%s-%s", bucketInfo.Name, bucketInfo.Location, compartmentID)
		if mBuckets[key] == key {
			bucketInfo.Managed = true
		}
	}

	return nil
}

// GetManagedBuckets gives back managed Oracle object store buckets from DB
func (o *OCIObjectStore) GetManagedBuckets() ([]model.ManagedOracleBucket, error) {

	log.Infof("Retrieving managed buckets")

	var managedBuckets []model.ManagedOracleBucket
	if err := queryWithOrderByDb(&model.ManagedOracleBucket{OrgID: o.org.ID}, "name asc, location asc", &managedBuckets); err != nil {
		log.Errorf("Retrieving managed buckets in organisation id=%s failed: %s", err.Error())
		return nil, err
	}

	return managedBuckets, nil
}
