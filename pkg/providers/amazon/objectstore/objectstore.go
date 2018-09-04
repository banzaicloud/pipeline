package objectstore

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

type objectStore struct {
	session *session.Session

	// client for the specified region is cached here
	// we still need the session to check a bucket in case it's in a different region
	client *s3.S3

	waitForCompletion bool
}

// New returns an Object Store instance that manages Amazon S3 buckets.
func New(session *session.Session, opts ...Option) *objectStore {
	s := &objectStore{
		session: session,

		client: s3.New(session),
	}

	for _, o := range opts {
		o.apply(s)
	}

	return s
}

// CreateBucket creates a new bucket in the object store.
func (s *objectStore) CreateBucket(bucketName string) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := s.client.CreateBucket(input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == s3.ErrCodeBucketAlreadyExists {
			err = errBucketAlreadyExists{}
		}

		return errors.Wrap(err, "bucket creation failed")
	}

	if s.waitForCompletion {
		err := s.client.WaitUntilBucketExists(&s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return errors.Wrap(err, "could not wait for bucket to be ready")
		}
	}

	return nil
}

// ListBuckets lists the current buckets in the object store.
func (s *objectStore) ListBuckets() ([]string, error) {
	input := &s3.ListBucketsInput{}
	buckets, err := s.client.ListBuckets(input)
	if err != nil {
		return nil, errors.Wrap(err, "could not list buckets")
	}

	var bucketList []string
	for _, bucket := range buckets.Buckets {
		bucketList = append(bucketList, *bucket.Name)
	}

	return bucketList, nil
}

// GetRegion gets the region of the given bucket
func (s *objectStore) GetRegion(bucketName string) (string, error) {
	region, err := s3manager.GetBucketRegionWithClient(context.Background(), s.client, bucketName)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
			err = errBucketNotFound{}
		}

		return "", errors.Wrap(err, "checking bucket region failed")
	}
	return region, nil
}

// CheckBucket checks the status of the given bucket.
func (s *objectStore) CheckBucket(bucketName string) error {
	// Check if the bucket's region matches the current region
	actualRegion, err := s.GetRegion(bucketName)
	if err != nil {
		return err
	}

	client := s.client
	if actualRegion != *s.session.Config.Region {
		sess := s.session.Copy(&aws.Config{
			Region: aws.String(actualRegion),
		})

		client = s3.New(sess)
	}

	input := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = client.HeadBucket(input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
			err = errBucketNotFound{}
		}

		return errors.Wrap(err, "checking bucket failed")
	}

	return nil
}

// DeleteBucket removes a bucket from the object store.
func (s *objectStore) DeleteBucket(bucketName string) error {
	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := s.client.DeleteBucket(input)
	if err != nil {
		return errors.Wrap(err, "bucket deletion failed")
	}

	if s.waitForCompletion {
		err := s.client.WaitUntilBucketNotExists(&s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return errors.Wrap(err, "could not wait for bucket to be deleted")
		}
	}

	return nil
}
