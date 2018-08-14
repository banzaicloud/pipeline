package google

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/gin-gonic/gin/json"
	"github.com/jinzhu/gorm"
	"github.com/kubicorn/kubicorn/pkg/logger"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	apiStorage "google.golang.org/api/storage/v1"
)

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }

// ObjectStore stores all required parameters for bucket creation.
type ObjectStore struct {
	db     *gorm.DB
	logger logrus.FieldLogger

	org            *auth.Organization
	serviceAccount *verify.ServiceAccount

	location string
}

// NewObjectStore returns a new object store instance.
func NewObjectStore(
	org *auth.Organization,
	serviceAccount *verify.ServiceAccount,
	location string,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *ObjectStore {
	return &ObjectStore{
		db:             db,
		logger:         logger,
		org:            org,
		serviceAccount: serviceAccount,
		location:       location,
	}
}

func (s *ObjectStore) getLogger(bucketName string) logrus.FieldLogger {
	return s.logger.WithFields(logrus.Fields{
		"organization": s.org.ID,
		"bucket":       bucketName,
		"location":     s.location,
	})
}

// CreateBucket creates a Google Bucket with the provided name and location.
func (s *ObjectStore) CreateBucket(bucketName string) {
	logger := s.getLogger(bucketName)

	bucket := &ObjectStoreModel{}
	searchCriteria := s.searchCriteria(bucketName)

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			logger.Errorf("error happened during getting bucket from DB: %s", err.Error())

			return
		}
	}

	logger.Info("getting credentials")
	credentials, err := s.newGoogleCredentials()

	if err != nil {
		logger.Errorf("getting credentials failed: %s", err.Error())

		return
	}

	logger.Info("creating new storage client")

	ctx := context.Background()

	client, err := storage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		logger.Errorf("failed to create client: %s", err.Error())

		return
	}
	defer client.Close()

	logger.Info("storage client created successfully")

	bucket.Name = bucketName
	bucket.Organization = *s.org
	bucket.Location = s.location

	logger.Info("saving bucket in DB")

	err = s.db.Save(bucket).Error
	if err != nil {
		logger.Errorf("error happened during saving bucket in DB: %s", err.Error())

		return
	}

	bucketHandle := client.Bucket(bucketName)
	bucketAttrs := &storage.BucketAttrs{
		Location:      s.location,
		RequesterPays: false,
	}

	if err := bucketHandle.Create(ctx, s.serviceAccount.ProjectId, bucketAttrs); err != nil {
		logger.Errorf("failed to create bucket (rolling back): %s", err.Error())

		err = s.db.Delete(bucket).Error
		if err != nil {
			logger.Error(err.Error())
		}

		return
	}

	logger.Infof("bucket created")

	return
}

// DeleteBucket deletes the GS bucket identified by the specified name
// provided the storage container is of 'managed' type.
func (s *ObjectStore) DeleteBucket(bucketName string) error {
	logger := s.getLogger(bucketName)

	bucket := &ObjectStoreModel{}
	searchCriteria := s.searchCriteria(bucketName)

	logger.Info("looking for bucket")

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
	}

	logger.Info("getting credentials")
	credentials, err := s.newGoogleCredentials()

	if err != nil {
		return fmt.Errorf("getting credentials failed: %s", err.Error())
	}

	logger.Info("creating new storage client")

	ctx := context.Background()

	client, err := storage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		return fmt.Errorf("failed to create client: %s", err.Error())
	}
	defer client.Close()

	logger.Info("storage client created successfully")

	bucketHandle := client.Bucket(bucketName)

	if err := bucketHandle.Delete(ctx); err != nil {
		return err
	}

	err = s.db.Delete(bucket).Error
	if err != nil {
		return fmt.Errorf("deleting bucket failed: %s", err.Error())
	}

	return nil
}

// CheckBucket checks the status of the given Google bucket.
func (s *ObjectStore) CheckBucket(bucketName string) error {
	logger := s.getLogger(bucketName)
	logger.Info("looking for bucket")

	logger.Info("getting credentials")
	credentials, err := s.newGoogleCredentials()

	if err != nil {
		return fmt.Errorf("getting credentials failed: %s", err.Error())
	}

	logger.Info("creating new storage client")

	ctx := context.Background()

	client, err := storage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		return fmt.Errorf("failed to create client: %s", err.Error())
	}
	defer client.Close()

	logger.Info("storage client created successfully")

	logger.Info("retrieving bucket from Google")
	bucketsIterator := client.Buckets(ctx, s.serviceAccount.ProjectId)
	bucketsIterator.Prefix = bucketName

	for {
		bucket, err := bucketsIterator.Next()
		if err == iterator.Done {
			return bucketNotFoundError{}
		}

		if err != nil {
			return fmt.Errorf("error occurred while iterating over GS buckets: %s", err.Error())
		}

		if bucketName == bucket.Name {
			return nil
		}
	}
}

// ListBuckets returns a list of GS buckets that can be accessed with the credentials
// referenced by the secret field. GS buckets that were created by a user in the current
// org are marked as 'managed`
func (s *ObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	logger.Info("getting credentials")
	credentials, err := s.newGoogleCredentials()

	if err != nil {
		return nil, fmt.Errorf("getting credentials failed: %s", err.Error())
	}

	logger.Info("creating new storage client")

	ctx := context.Background()

	client, err := storage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %s", err.Error())
	}
	defer client.Close()

	logger.Info("storage client created successfully")

	logger.Info("retrieving bucket from Google")
	bucketsIterator := client.Buckets(ctx, s.serviceAccount.ProjectId)

	logger.Info("retrieving managed buckets")

	var objectStores []ObjectStoreModel

	err = s.db.Where(&ObjectStoreModel{OrganizationID: s.org.ID}).Order("name asc").Find(&objectStores).Error
	if err != nil {
		return nil, fmt.Errorf("retrieving managed buckets failed: %s", err.Error())
	}

	var bucketList []*objectstore.BucketInfo

	for {
		bucket, err := bucketsIterator.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("error occurred while iterating over GS buckets: %s", err.Error())
		}

		bucketInfo := &objectstore.BucketInfo{
			Name:    bucket.Name,
			Managed: false,
		}

		// objectStores must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(objectStores), func(i int) bool {
			return strings.Compare(objectStores[i].Name, bucket.Name) >= 0
		})
		if idx < len(objectStores) && strings.Compare(objectStores[idx].Name, bucket.Name) == 0 {
			bucketInfo.Managed = true
		}

		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

func (s *ObjectStore) newGoogleCredentials() (*google.Credentials, error) {
	credentialsJson, err := json.Marshal(s.serviceAccount)
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

// searchCriteria returns the database search criteria to find managed bucket with the given name.
func (s *ObjectStore) searchCriteria(bucketName string) *ObjectStoreModel {
	return &ObjectStoreModel{
		OrganizationID: s.org.ID,
		Name:           bucketName,
	}
}
