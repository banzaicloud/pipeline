package objectstore

import (
	"errors"
	"sort"
	"strings"

	pipelineAuth "github.com/banzaicloud/pipeline/auth"
	model "github.com/banzaicloud/pipeline/pkg/providers/oracle/model/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	verify "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
	pkgStorage "github.com/banzaicloud/pipeline/pkg/storage"
	"github.com/banzaicloud/pipeline/secret"
)

// OCIObjectStore stores all required parameters for container creation
type OCIObjectStore struct {
	secret        *secret.SecretItemResponse
	org           *pipelineAuth.Organization
	compartmentID string
}

// CreateBucket creates an Oracle object store bucket with the given name
func (o *OCIObjectStore) CreateBucket(name string) {

	managedBucket := &model.ManagedOracleBucket{}
	searchCriteria := o.newManagedBucketSearchCriteria(name)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		switch err.(type) {
		case ManagedBucketNotFoundError:
		default:
			log.Errorf("Error happened during getting bucket description from DB %s", err.Error())
			return
		}
	}

	oci, err := oci.NewOCI(verify.CreateOCICredential(o.secret.Values))
	if err != nil {
		log.Errorf("Bucket creation error: %s", err)
	}

	client, err := oci.NewObjectStorageClient()
	if err != nil {
		log.Errorf("Bucket creation error: %s", err)
	}

	managedBucket.Name = name
	managedBucket.Organization = *o.org
	managedBucket.CompartmentID = oci.CompartmentOCID

	err = persistToDb(managedBucket)
	if err != nil {
		log.Errorf("Error happened during persisting bucket description to DB")
		return
	}
	if err := client.CreateBucket(name); err != nil {
		log.Errorf("Failed to create bucket: %s", err.Error())
		if e := deleteFromDbByPK(managedBucket); e != nil {
			log.Error(e.Error())
		}
		return
	}
	log.Infof("%s bucket created", name)
}

// ListBuckets list all buckets in Oracle object store
func (o *OCIObjectStore) ListBuckets() ([]*pkgStorage.BucketInfo, error) {

	log.Info("Getting credentials")
	oci, err := oci.NewOCI(verify.CreateOCICredential(o.secret.Values))
	if err != nil {
		log.Errorf("Getting credentials failed: %s", err.Error())
		return nil, err
	}

	client, err := oci.NewObjectStorageClient()
	if err != nil {
		log.Errorf("Creating Oracle object storage client failed: %s", err.Error())
		return nil, err
	}

	log.Info("Retrieving bucket list from Oracle")
	buckets, err := client.ListBuckets()
	if err != nil {
		return nil, err
	}

	var bucketList []*pkgStorage.BucketInfo
	for _, bucket := range buckets {
		bucketInfo := &pkgStorage.BucketInfo{Name: *bucket.Name, Managed: false}
		bucketList = append(bucketList, bucketInfo)
	}

	o.MarkManagedBuckets(bucketList)

	return bucketList, nil
}

// DeleteBucket deletes the managed bucket with the given name from Oracle object store
func (o *OCIObjectStore) DeleteBucket(name string) error {

	managedBucket := &model.ManagedOracleBucket{}
	searchCriteria := o.newManagedBucketSearchCriteria(name)

	log.Info("Looking up managed bucket: name=%s", name)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		return err
	}

	log.Info("Getting credentials")
	oci, err := oci.NewOCI(verify.CreateOCICredential(o.secret.Values))
	if err != nil {
		log.Errorf("Getting credentials failed: %s", err.Error())
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

	managedBucket := &model.ManagedOracleBucket{}
	searchCriteria := o.newManagedBucketSearchCriteria(name)

	log.Infof("Looking up managed bucket: name=%s", name)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		log.Error(err)
		return err
	}

	log.Info("Getting credentials")
	oci, err := oci.NewOCI(verify.CreateOCICredential(o.secret.Values))
	if err != nil {
		log.Errorf("Getting credentials failed: %s", err.Error())
		return err
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

// WithRegion always return "not implemented" error
func (o *OCIObjectStore) WithRegion(name string) error {
	return errors.New("not implemented")
}

// newManagedBucketSearchCriteria returns the database search criteria to find managed bucket with the given name
func (o *OCIObjectStore) newManagedBucketSearchCriteria(bucketName string) *model.ManagedOracleBucket {
	return &model.ManagedOracleBucket{
		OrgID: o.org.ID,
		Name:  bucketName,
	}
}

// MarkManagedBuckets marks buckets by name to 'managed'
func (o *OCIObjectStore) MarkManagedBuckets(buckets []*pkgStorage.BucketInfo) error {

	managedBuckets, err := o.GetManagedBuckets()
	if err != nil {
		return err
	}

	log.Infof("Marking managed buckets by name")
	for _, bucket := range buckets {
		// managedBuckets must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(managedBuckets), func(i int) bool {
			return strings.Compare(managedBuckets[i].Name, bucket.Name) >= 0
		})
		if idx < len(managedBuckets) && strings.Compare(managedBuckets[idx].Name, bucket.Name) == 0 {
			bucket.Managed = true
		}
	}

	return nil
}

// GetManagedBuckets gives back managed Oracle object store buckets from DB
func (o *OCIObjectStore) GetManagedBuckets() ([]model.ManagedOracleBucket, error) {

	log.Infof("Retrieving managed buckets")

	var managedBuckets []model.ManagedOracleBucket
	if err := queryWithOrderByDb(&model.ManagedOracleBucket{OrgID: o.org.ID}, "name asc", &managedBuckets); err != nil {
		log.Errorf("Retrieving managed buckets in organisation id=%s failed: %s", err.Error())
		return nil, err
	}

	return managedBuckets, nil
}
