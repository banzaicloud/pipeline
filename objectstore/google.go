package objectstore

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin/json"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	apiStorage "google.golang.org/api/storage/v1"
	"sort"
	"strings"
)

// ManagedGoogleBucket is the schema for the DB
type ManagedGoogleBucket struct {
	ID           uint              `gorm:"primary_key"`
	Organization auth.Organization `gorm:"foreignkey:OrgID"`
	OrgID        uint              `gorm:"index;not null"`
	Name         string            `gorm:"unique_index:bucketName"`
	Location     string
}

// GoogleObjectStore stores all required parameters for container creation
type GoogleObjectStore struct {
	location       string
	serviceAccount *cluster.ServiceAccount // TODO: serviceAccount type should be in a common place?
	org            *auth.Organization
}

// NewGoogleServiceAccount creates a service account for google
// TODO: this logic is duplicate thus should be in a common place so as it can be used from gke.go:newClientFromCredentials() as well
func NewGoogleServiceAccount(s *secret.SecretsItemResponse) *cluster.ServiceAccount {
	return &cluster.ServiceAccount{
		Type:                   s.Values[secret.Type],
		ProjectId:              s.Values[secret.ProjectId],
		PrivateKeyId:           s.Values[secret.PrivateKeyId],
		PrivateKey:             s.Values[secret.PrivateKey],
		ClientEmail:            s.Values[secret.ClientEmail],
		ClientId:               s.Values[secret.ClientId],
		AuthUri:                s.Values[secret.AuthUri],
		TokenUri:               s.Values[secret.TokenUri],
		AuthProviderX50CertUrl: s.Values[secret.AuthX509Url],
		ClientX509CertUrl:      s.Values[secret.ClientX509Url],
	}
}

// WithResourceGroup updates the resource group. Always return "not implemented" error
func (b *GoogleObjectStore) WithResourceGroup(resourceGroup string) error {
	return errors.New("not implemented")
}

// WithStorageAccount updates the storage account. Always return "not implemented" error
func (b *GoogleObjectStore) WithStorageAccount(storageAccount string) error {
	return errors.New("not implemented")
}

// WithRegion updates the region.
func (b *GoogleObjectStore) WithRegion(region string) error {
	b.location = region
	return nil
}

// CreateBucket creates a Google Bucket with the provided name and location
func (b *GoogleObjectStore) CreateBucket(bucketName string) {
	log := logger.WithFields(logrus.Fields{"tag": "CreateBucket"})

	managedBucket := &ManagedGoogleBucket{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		switch err.(type) {
		case ManagedBucketNotFoundError:
		default:
			log.Errorf("Error happened during getting bucket description from DB %s", err.Error())
			return
		}
	}

	ctx := context.Background()
	log.Info("Getting credentials")
	credentials, err := newGoogleCredentials(b)

	if err != nil {
		log.Errorf("Getting credentials failed due to: %s", err.Error())
		return
	}

	log.Info("Creating new storage client")

	client, err := storage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Errorf("Failed to create client: %s", err.Error())
		return
	}
	defer client.Close()

	log.Info("Storage client created successfully")

	bucket := client.Bucket(bucketName)
	bucketAttrs := &storage.BucketAttrs{
		Location:      b.location,
		RequesterPays: false,
	}

	managedBucket.Name = bucketName
	managedBucket.Organization = *b.org
	managedBucket.Location = b.location

	err = persistToDb(managedBucket)
	if err != nil {
		log.Errorf("Error happened during persisting bucket description to DB")
		return
	}
	if err := bucket.Create(ctx, b.serviceAccount.ProjectId, bucketAttrs); err != nil {
		log.Errorf("Failed to create bucket: %s", err.Error())
		if e := deleteFromDbByPK(managedBucket); e != nil {
			log.Error(e.Error())
		}
		return
	}
	log.Infof("%s bucket created in %s location", bucketName, b.location)
	return
}

// DeleteBucket deletes the GS bucket identified by the specified name
// provided the storage container is of 'managed` type
func (b *GoogleObjectStore) DeleteBucket(bucketName string) error {
	log := logger.WithFields(logrus.Fields{"tag": "GoogleObjectStore.DeleteBucket"})

	managedBucket := &ManagedGoogleBucket{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)

	log.Info("Looking up managed bucket: name=%s", bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		return err
	}

	ctx := context.Background()

	log.Info("Getting credentials")
	credentials, err := newGoogleCredentials(b)

	if err != nil {
		log.Errorf("Getting credentials failed: %s", err.Error())
		return err
	}

	client, err := storage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Errorf("Creating Google storage.Client failed: %s", err.Error())
		return err
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)

	if err := bucket.Delete(ctx); err != nil {
		return err
	}

	if err = deleteFromDbByPK(managedBucket); err != nil {
		log.Errorf("Deleting managed GS bucket from database failed: %s", err.Error())
		return err
	}

	return nil
}

//CheckBucket check the status of the given Google bucket
func (b *GoogleObjectStore) CheckBucket(bucketName string) error {
	log := logger.WithFields(logrus.Fields{"tag": "GoogleObjectStore.CheckBucket"})
	managedBucket := &ManagedGoogleBucket{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)
	log.Info("Looking up managed bucket: name=%s", bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		return ManagedBucketNotFoundError{}
	}

	log.Info("Getting credentials")
	credentials, err := newGoogleCredentials(b)
	if err != nil {
		log.Errorf("Getting credentials failed: %s", err.Error())
		return errors.New("getting credentials failed")
	}
	ctx := context.TODO()
	client, err := storage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Errorf("Creating Google storage.Client failed: %s", err.Error())
		return err
	}
	defer client.Close()

	log.Info("Retrieving bucket from Google")
	bucketsIterator := client.Buckets(ctx, b.serviceAccount.ProjectId)
	bucketsIterator.Prefix = bucketName

	for {
		bucket, err := bucketsIterator.Next()
		if err == iterator.Done {
			return ManagedBucketNotFoundError{}
		}
		if err != nil {
			log.Errorf("Error occurred while iterating over GS buckets: %s", err.Error())
			return err
		}
		if bucketName == bucket.Name {
			return nil
		}
	}
}

// ListBuckets returns a list of GS buckets that can be accessed with the credentials
// referenced by the secret field. GS buckets that were created by a user in the current
// org are marked as 'managed`
func (b *GoogleObjectStore) ListBuckets() ([]*components.BucketInfo, error) {
	log := logger.WithFields(logrus.Fields{"tag": "GoogleObjectStore.ListBuckets"})

	ctx := context.Background()

	log.Info("Getting credentials")
	credentials, err := newGoogleCredentials(b)

	if err != nil {
		log.Errorf("Getting credentials failed: %s", err.Error())
		return nil, err
	}

	client, err := storage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Errorf("Creating Google storage.Client failed: %s", err.Error())
		return nil, err
	}
	defer client.Close()

	log.Info("Retrieving bucket list from Google")
	bucketsIterator := client.Buckets(ctx, b.serviceAccount.ProjectId)

	log.Infof("Retrieving managed buckets")

	var managedGoogleBuckets []ManagedGoogleBucket
	if err = queryWithOrderByDb(&ManagedGoogleBucket{OrgID: b.org.ID}, "name asc", &managedGoogleBuckets); err != nil {
		log.Errorf("Retrieving managed buckets in organisation id=%s failed: %s", err.Error())
		return nil, err
	}

	var bucketList []*components.BucketInfo

	for {
		bucket, err := bucketsIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Errorf("Error occurred while iterating over GS buckets: %s", err.Error())
			return nil, err
		}

		bucketInfo := &components.BucketInfo{Name: bucket.Name, Managed: false}

		// managedGoogleBuckets must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(managedGoogleBuckets), func(i int) bool {
			return strings.Compare(managedGoogleBuckets[i].Name, bucket.Name) >= 0
		})
		if idx < len(managedGoogleBuckets) && strings.Compare(managedGoogleBuckets[idx].Name, bucket.Name) == 0 {
			bucketInfo.Managed = true
		}

		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

func newGoogleCredentials(b *GoogleObjectStore) (*google.Credentials, error) {
	credentialsJson, err := json.Marshal(b.serviceAccount)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	credentials, err := google.CredentialsFromJSON(ctx, credentialsJson, apiStorage.DevstorageFullControlScope)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}

// newManagedBucketSearchCriteria returns the database search criteria to find managed bucket with the given name
func (b *GoogleObjectStore) newManagedBucketSearchCriteria(bucketName string) *ManagedGoogleBucket {
	return &ManagedGoogleBucket{
		OrgID: b.org.ID,
		Name:  bucketName,
	}
}
