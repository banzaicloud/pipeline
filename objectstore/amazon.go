package objectstore

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
)

// ManagedAmazonBucket is the schema for the DB
type ManagedAmazonBucket struct {
	ID           uint              `gorm:"primary_key"`
	Organization auth.Organization `gorm:"foreignkey:OrgID"`
	OrgID        uint              `gorm:"index;not null"`
	Name         string            `gorm:"unique_index:bucketName"`
	Region       string
}

// AmazonObjectStore stores all required parameters for container creation
type AmazonObjectStore struct {
	region string
	secret *secret.SecretsItemResponse
	org    *auth.Organization
}

// WithResourceGroup updates the resource group. Always return "not implemented" error
func (b *AmazonObjectStore) WithResourceGroup(resourceGroup string) error {
	return errors.New("not implemented")
}

// WithStorageAccount updates the storage account. Always return "not implemented" error
func (b *AmazonObjectStore) WithStorageAccount(storageAccount string) error {
	return errors.New("not implemented")
}

// WithRegion updates the region.
func (b *AmazonObjectStore) WithRegion(region string) error {
	b.region = region
	return nil
}

// CreateBucket creates a S3 bucket with the provided name
func (b *AmazonObjectStore) CreateBucket(bucketName string) {
	log := logger.WithFields(logrus.Fields{"tag": "CreateBucket"})

	managedBucket := &ManagedAmazonBucket{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		switch err.(type) {
		case ManagedBucketNotFoundError:
		default:
			log.Errorf("Error happened during getting bucket description from DB %s", err.Error())
			return
		}
	}

	log.Info("Creating S3Client...")
	svc, err := createS3Client(b.region, b.secret)
	if err != nil {
		log.Error("Creating S3Client failed!")
		return
	}
	log.Info("S3Client create succeeded!")
	log.Debugf("Region is: %s", b.region)
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	managedBucket.Name = bucketName
	managedBucket.Organization = *b.org
	managedBucket.Region = b.region

	if err = persistToDb(managedBucket); err != nil {
		log.Errorf("Error happened during persisting bucket description to DB %s", err.Error())
		return
	}
	_, err = svc.CreateBucket(input)
	if err != nil {
		log.Errorf("Could not create a new S3 Bucket, %s", err.Error())
		if e := deleteFromDbByPK(managedBucket); e != nil {
			log.Error(e.Error())
		}
		return
	}
	log.Debugf("Waiting for bucket %s to be created...", bucketName)

	err = svc.WaitUntilBucketExists(&s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		log.Errorf("Error happened during waiting for the bucket to be created, %s", err.Error())
		return
	}
	log.Infof("Bucket %s Created", bucketName)
	return
}

// DeleteBucket deletes the S3 bucket identified by the specified name
// provided the storage container is of 'managed` type
func (b *AmazonObjectStore) DeleteBucket(bucketName string) error {
	log := logger.WithFields(logrus.Fields{"tag": "AmazonObjectStore.DeleteBucket"})

	managedBucket := &ManagedAmazonBucket{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)

	log.Info("Looking up managed bucket: name=%s", bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		return err
	}

	svc, err := createS3Client(managedBucket.Region, b.secret)

	if err != nil {
		log.Errorf("Creating S3Client failed: %s", err.Error())
		return err
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
		log.Errorf("Error occurred while waiting for the S3 Bucket to be deleted, %s", err.Error())
		return err
	}

	if err = deleteFromDbByPK(managedBucket); err != nil {
		log.Errorf("Deleting managed S3 bucket from database failed: %s", err.Error())
		return err
	}

	return nil
}

//CheckBucket check the status of the given S3 bucket
func (b *AmazonObjectStore) CheckBucket(bucketName string) error {
	log := logger.WithFields(logrus.Fields{"tag": "AmazonObjectStore.CheckBucket"})
	managedBucket := &ManagedAmazonBucket{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)
	log.Info("Looking up managed bucket: name=%s", bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		return ManagedBucketNotFoundError{}
	}

	svc, err := createS3Client(managedBucket.Region, b.secret)

	if err != nil {
		log.Errorf("Creating S3Client failed: %s", err.Error())
		return errors.New("failed to create s3client")
	}
	_, err = svc.HeadBucket(&s3.HeadBucketInput{Bucket: aws.String(bucketName)})
	if err != nil {
		log.Errorf("%s", err.Error())
		return err
	}

	return nil
}

// ListBuckets returns a list of S3 buckets that can be accessed with the credentials
// referenced by the secret field. S3 buckets that were created by a user in the current
// org are marked as 'managed`
func (b *AmazonObjectStore) ListBuckets() ([]*components.BucketInfo, error) {
	log := logger.WithFields(logrus.Fields{"tag": "AmazonObjectStore.ListBuckets"})

	svc, err := createS3Client(b.region, b.secret)

	if err != nil {
		log.Errorf("Creating S3Client failed: %s", err.Error())
		return nil, err
	}

	log.Info("Retrieving bucket list from Amazon")
	input := &s3.ListBucketsInput{}
	buckets, err := svc.ListBuckets(input)
	if err != nil {
		log.Errorf("Retrieving bucket list from Amazon failed: %s", err.Error())
		return nil, err
	}

	log.Infof("Retrieving managed buckets")

	var managedAmazonBuckets []ManagedAmazonBucket
	if err = queryWithOrderByDb(&ManagedAmazonBucket{OrgID: b.org.ID}, "name asc", &managedAmazonBuckets); err != nil {
		log.Errorf("Retrieving managed buckets in organisation id=%s failed: %s", err.Error())
		return nil, err
	}

	var bucketList []*components.BucketInfo
	for _, bucket := range buckets.Buckets {
		// managedAmazonBuckets must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(managedAmazonBuckets), func(i int) bool {
			return strings.Compare(managedAmazonBuckets[i].Name, *bucket.Name) >= 0
		})

		bucketInfo := &components.BucketInfo{Name: *bucket.Name, Managed: false}
		if idx < len(managedAmazonBuckets) && strings.Compare(managedAmazonBuckets[idx].Name, *bucket.Name) == 0 {
			bucketInfo.Managed = true
		}

		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

func createS3Client(region string, retrievedSecret *secret.SecretsItemResponse) (*s3.S3, error) {
	log := logger.WithFields(logrus.Fields{"tag": "createS3Client"})
	log.Info("Creating AWS session")
	s, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			retrievedSecret.Values[secret.AwsAccessKeyId],
			retrievedSecret.Values[secret.AwsSecretAccessKey],
			""),
	})

	if err != nil {
		log.Errorf("Error creating AWS session %s", err.Error())
		return nil, err
	}
	log.Info("AWS session successfully created")
	return s3.New(s), nil
}

// newManagedBucketSearchCriteria returns the database search criteria to find managed bucket with the given name
func (b *AmazonObjectStore) newManagedBucketSearchCriteria(bucketName string) *ManagedAmazonBucket {
	return &ManagedAmazonBucket{
		OrgID: b.org.ID,
		Name:  bucketName,
	}
}
