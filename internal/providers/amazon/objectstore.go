package amazon

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	commonObjectstore "github.com/banzaicloud/pipeline/pkg/objectstore"
	amazonObjectstore "github.com/banzaicloud/pipeline/pkg/providers/amazon/objectstore"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }

// objectStore stores all required parameters for bucket creation.
type objectStore struct {
	objectStore commonObjectstore.ObjectStore

	region string
	secret *secret.SecretItemResponse

	org *auth.Organization

	db     *gorm.DB
	logger logrus.FieldLogger
}

// NewObjectStore returns a new object store instance.
func NewObjectStore(
	region string,
	secret *secret.SecretItemResponse,
	org *auth.Organization,
	db *gorm.DB,
	logger logrus.FieldLogger,
) (*objectStore, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: verify.CreateAWSCredentials(secret.Values),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not create aws session")
	}

	return &objectStore{
		objectStore: amazonObjectstore.New(sess, amazonObjectstore.WaitForCompletion(true)),
		region:      region,
		secret:      secret,
		org:         org,
		db:          db,
		logger:      logger,
	}, nil
}

func (s *objectStore) getLogger() logrus.FieldLogger {
	return s.logger.WithFields(logrus.Fields{
		"organization": s.org.ID,
		"secret":       s.secret.ID,
		"region":       s.region,
	})
}

// CreateBucket creates an S3 bucket with the provided name.
func (s *objectStore) CreateBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return errors.Wrap(err, "error happened during getting bucket from DB")
		}
	}

	bucket.Name = bucketName
	bucket.Organization = *s.org
	bucket.Region = s.region

	if err := s.db.Save(bucket).Error; err != nil {
		return errors.Wrap(err, "error happened during saving bucket in DB")
	}

	logger.Info("creating bucket")

	if err := s.objectStore.CreateBucket(bucketName); err != nil {
		e := s.db.Delete(bucket).Error
		if e != nil {
			logger.Error(e.Error())
		}

		return errors.Wrap(err, "could not create bucket (rolling back)")
	}

	logger.Info("bucket created")

	return nil
}

// DeleteBucket deletes the S3 bucket identified by the specified name
// provided the storage container is of 'managed' type.
func (s *objectStore) DeleteBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	logger.Info("looking for bucket")

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
	}

	logger.Info("deleting bucket")

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(bucket.Region), // Region is not provided when deleting a bucket
		Credentials: verify.CreateAWSCredentials(s.secret.Values),
	})
	if err != nil {
		return errors.Wrap(err, "could not create aws session")
	}

	objectStore := amazonObjectstore.New(sess, amazonObjectstore.WaitForCompletion(true))

	if err := objectStore.DeleteBucket(bucketName); err != nil {
		return err
	}

	if err := s.db.Delete(bucket).Error; err != nil {
		return errors.Wrap(err, "deleting bucket from database failed")
	}

	return nil
}

// CheckBucket checks the status of the given S3 bucket.
func (s *objectStore) CheckBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)

	logger.Info("looking for bucket")

	if err := s.objectStore.CheckBucket(bucketName); err != nil {
		return err
	}

	return nil
}

// ListBuckets returns a list of S3 buckets that can be accessed with the credentials
// referenced by the secret field. S3 buckets that were created by a user in the current
// org are marked as 'managed'.
func (s *objectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	logger := s.getLogger()

	logger.Info("retrieving bucket list")

	buckets, err := s.objectStore.ListBuckets()
	if err != nil {
		return nil, err
	}

	logger.Infof("retrieving managed buckets")

	var amazonBuckets []*ObjectStoreBucketModel

	err = s.db.Where(&ObjectStoreBucketModel{OrganizationID: s.org.ID}).Order("name asc").Find(&amazonBuckets).Error
	if err != nil {
		return nil, fmt.Errorf("retrieving managed buckets failed: %s", err.Error())
	}

	var bucketList []*objectstore.BucketInfo
	for _, bucket := range buckets {
		// amazonBuckets must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(amazonBuckets), func(i int) bool {
			return strings.Compare(amazonBuckets[i].Name, bucket) >= 0
		})

		bucketInfo := &objectstore.BucketInfo{Name: bucket, Managed: false}
		if idx < len(amazonBuckets) && strings.Compare(amazonBuckets[idx].Name, bucket) == 0 {
			bucketInfo.Managed = true
		}

		region, err := s.objectStore.GetRegion(bucket)
		if err != nil {
			return nil, err
		}
		bucketInfo.Location = region

		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

// searchCriteria returns the database search criteria to find bucket with the given name.
func (s *objectStore) searchCriteria(bucketName string) *ObjectStoreBucketModel {
	return &ObjectStoreBucketModel{
		OrganizationID: s.org.ID,
		Name:           bucketName,
	}
}
