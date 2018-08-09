package amazon

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }

// ObjectStore stores all required parameters for bucket creation.
type ObjectStore struct {
	db     *gorm.DB
	logger logrus.FieldLogger

	secret *secret.SecretItemResponse
	org    *auth.Organization

	region string
}

// NewObjectStore returns a new object store instance.
func NewObjectStore(
	org *auth.Organization,
	secret *secret.SecretItemResponse,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *ObjectStore {
	return &ObjectStore{
		db:     db,
		logger: logger,
		org:    org,
		secret: secret,
	}
}

// WithResourceGroup updates the resource group. Always return "not implemented" error
func (s *ObjectStore) WithResourceGroup(resourceGroup string) error {
	return errors.New("not implemented")
}

// WithStorageAccount updates the storage account. Always return "not implemented" error
func (s *ObjectStore) WithStorageAccount(storageAccount string) error {
	return errors.New("not implemented")
}

// WithRegion updates the region.
func (s *ObjectStore) WithRegion(region string) error {
	s.region = region

	return nil
}

func (s *ObjectStore) getLogger(bucketName string) logrus.FieldLogger {
	return s.logger.WithFields(logrus.Fields{
		"organization": s.org.ID,
		"region":       s.region,
		"bucket":       bucketName,
	})
}

// CreateBucket creates an S3 bucket with the provided name.
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

	logger.Info("creating S3 client")
	svc, err := createS3Client(s.region, s.secret)
	if err != nil {
		logger.Errorf("creating S3 client failed: %s", err.Error())

		return
	}

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	bucket.Name = bucketName
	bucket.Organization = *s.org
	bucket.Region = s.region

	err = s.db.Save(bucket).Error
	if err != nil {
		logger.Errorf("error happened during saving bucket in DB: %s", err.Error())

		return
	}

	_, err = svc.CreateBucket(input)
	if err != nil {
		logger.Errorf("could not create a new S3 Bucket (rolling back), %s", err.Error())

		err = s.db.Delete(bucket).Error
		if err != nil {
			logger.Error(err.Error())
		}

		return
	}

	logger.Debug("waiting for bucket to be created")

	err = svc.WaitUntilBucketExists(&s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		logger.Errorf("error happened during waiting for the bucket to be created, %s", err.Error())

		return
	}

	logger.Info("bucket created")

	return
}

// DeleteBucket deletes the S3 bucket identified by the specified name
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

	logger.Info("creating S3 client")
	svc, err := createS3Client(bucket.Region, s.secret)
	if err != nil {
		return fmt.Errorf("creating S3 client failed: %s", err.Error())
	}

	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = svc.DeleteBucket(input)
	if err != nil {
		return err
	}

	err = svc.WaitUntilBucketNotExists(&s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("error occurred while waiting for the S3 Bucket to be deleted: %s", err.Error())
	}

	err = s.db.Delete(bucket).Error
	if err != nil {
		return fmt.Errorf("deleting S3 bucket from database failed: %s", err.Error())
	}

	return nil
}

// CheckBucket checks the status of the given S3 bucket.
func (s *ObjectStore) CheckBucket(bucketName string) error {
	logger := s.getLogger(bucketName)

	bucket := &ObjectStoreModel{}
	searchCriteria := s.searchCriteria(bucketName)

	logger.Info("looking for bucket")

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
	}

	logger.Info("creating S3 client")
	svc, err := createS3Client(bucket.Region, s.secret)
	if err != nil {
		return fmt.Errorf("creating S3 client failed: %s", err.Error())
	}

	input := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = svc.HeadBucket(input)
	if err != nil {
		return err
	}

	return nil
}

// ListBuckets returns a list of S3 buckets that can be accessed with the credentials
// referenced by the secret field. S3 buckets that were created by a user in the current
// org are marked as 'managed'.
func (s *ObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"region": s.region,
	})

	logger.Info("creating S3 client")
	svc, err := createS3Client(s.region, s.secret)
	if err != nil {
		return nil, fmt.Errorf("creating S3 client failed: %s", err.Error())
	}

	logger.Info("retrieving bucket list from Amazon")

	input := &s3.ListBucketsInput{}
	buckets, err := svc.ListBuckets(input)
	if err != nil {
		return nil, fmt.Errorf("retrieving bucket list from Amazon failed: %s", err.Error())
	}

	logger.Infof("retrieving managed buckets")

	var amazonBuckets []*ObjectStoreModel

	err = s.db.Where(&ObjectStoreModel{OrganizationID: s.org.ID}).Order("name asc").Find(&amazonBuckets).Error
	if err != nil {
		return nil, fmt.Errorf("retrieving managed buckets failed: %s", err.Error())
	}

	var bucketList []*objectstore.BucketInfo
	for _, bucket := range buckets.Buckets {
		// amazonBuckets must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(amazonBuckets), func(i int) bool {
			return strings.Compare(amazonBuckets[i].Name, *bucket.Name) >= 0
		})

		bucketInfo := &objectstore.BucketInfo{Name: *bucket.Name, Managed: false}
		if idx < len(amazonBuckets) && strings.Compare(amazonBuckets[idx].Name, *bucket.Name) == 0 {
			bucketInfo.Managed = true
		}

		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

func createS3Client(region string, retrievedSecret *secret.SecretItemResponse) (*s3.S3, error) {
	s, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: verify.CreateAWSCredentials(retrievedSecret.Values),
	})

	if err != nil {
		return nil, fmt.Errorf("error creating AWS session %s", err.Error())
	}

	return s3.New(s), nil
}

// searchCriteria returns the database search criteria to find bucket with the given name.
func (s *ObjectStore) searchCriteria(bucketName string) *ObjectStoreModel {
	return &ObjectStoreModel{
		OrganizationID: s.org.ID,
		Name:           bucketName,
	}
}
