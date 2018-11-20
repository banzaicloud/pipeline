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

package objectstore

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/goph/emperror"
)

type objectStore struct {
	config      Config
	credentials Credentials

	// client for the specified region is cached here
	// we still need the session to check a bucket in case it's in a different region
	client   *s3.S3
	uploader *s3manager.Uploader
	session  *session.Session

	waitForCompletion bool
}

// Config defines configuration
type Config struct {
	Region string
	Opts   []Option
}

// Credentials represents credentials necessary for access
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

// NewPlainObjectStore creates an objectstore with no configuration.
// Instances created with this function may be used to access methods that don't explicitly access external (cloud) resources
func NewPlainObjectStore() (*objectStore, error) {
	return &objectStore{}, nil
}

// New returns an Object Store instance that manages Amazon S3 buckets.
func New(config Config, credentials Credentials) (*objectStore, error) {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Region),
		Credentials: awsCredentials.NewStaticCredentials(
			credentials.AccessKeyID,
			credentials.SecretAccessKey,
			"",
		),
	})
	if err != nil {
		return nil, emperror.Wrap(err, "cloud not create AWS session")
	}

	s := &objectStore{
		session: sess,

		uploader: s3manager.NewUploader(sess),
		client:   s3.New(sess),

		config:      config,
		credentials: credentials,
	}

	for _, o := range config.Opts {
		o.apply(s)
	}

	return s, nil
}

// CreateBucket creates a new bucket in the object store.
func (s *objectStore) CreateBucket(bucketName string) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := s.client.CreateBucket(input)
	if err != nil {
		err = s.convertError(err)
		return emperror.With(emperror.Wrap(err, "bucket creation failed"), "bucket", bucketName)
	}

	if s.waitForCompletion {
		err := s.client.WaitUntilBucketExists(&s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return emperror.With(emperror.Wrap(err, "could not wait for bucket to be ready"), "bucket", bucketName)
		}
	}

	return nil
}

// ListBuckets lists the current buckets in the object store.
func (s *objectStore) ListBuckets() ([]string, error) {
	input := &s3.ListBucketsInput{}
	buckets, err := s.client.ListBuckets(input)
	if err != nil {
		return nil, emperror.Wrap(err, "could not list buckets")
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
		err = s.convertError(err)
		return "", emperror.WrapWith(err, "failed to get region", "bucket", bucketName)
	}
	return region, nil
}

// CheckBucket checks the status of the given bucket.
func (s *objectStore) CheckBucket(bucketName string) error {
	// Check if the bucket's region matches the current region
	actualRegion, err := s.GetRegion(bucketName)
	if err != nil {
		return emperror.WrapWith(err, "failed to check the bucket", "bucket", bucketName)
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
		err = s.convertError(err)
		return emperror.With(emperror.Wrap(err, "checking bucket failed"), "bucket", bucketName)
	}

	return nil
}

// DeleteBucket removes a bucket from the object store.
func (s *objectStore) DeleteBucket(bucketName string) error {
	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	obj, err := s.ListObjects(bucketName)
	if err != nil {
		return emperror.With(emperror.Wrap(err, "could not list objects"), "bucket", bucketName)
	}

	if len(obj) > 0 {
		return emperror.With(pkgErrors.ErrorBucketDeleteNotEmpty, "bucket", bucketName)
	}
	_, err = s.client.DeleteBucket(input)
	if err != nil {
		err = s.convertError(err)
		return emperror.With(emperror.Wrap(err, "bucket deletion failed"), "bucket", bucketName)
	}

	if s.waitForCompletion {
		err := s.client.WaitUntilBucketNotExists(&s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return emperror.With(emperror.Wrap(err, "could not wait for bucket to be deleted"), "bucket", bucketName)
		}
	}

	return nil
}

// ListObjects gets all keys in the bucket
func (s *objectStore) ListObjects(bucketName string) ([]string, error) {
	var keys []string
	err := s.client.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: &bucketName,
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
		}
		return !lastPage
	})

	if err != nil {
		err = s.convertError(err)
		return nil, emperror.With(emperror.Wrap(err, "error listing object for bucket"), "bucket", bucketName)
	}

	return keys, nil
}

// ListObjectsWithPrefix gets all keys with the given prefix from the bucket
func (s *objectStore) ListObjectsWithPrefix(bucketName, prefix string) ([]string, error) {
	var keys []string
	err := s.client.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: &bucketName,
		Prefix: &prefix,
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
		}
		return !lastPage
	})

	if err != nil {
		err = s.convertError(err)
		return nil, emperror.With(emperror.Wrap(err, "error listing object for bucket"), "bucket", bucketName, "prefix", prefix)
	}

	return keys, nil
}

// ListObjectKeyPrefixes gets a list of all object key prefixes that come before the provided delimiter
func (s *objectStore) ListObjectKeyPrefixes(bucketName string, delimiter string) ([]string, error) {
	var prefixes []string

	err := s.client.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket:    &bucketName,
		Delimiter: &delimiter,
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		var p string
		for _, prefix := range page.CommonPrefixes {
			p = *prefix.Prefix
			prefixes = append(prefixes, p[0:strings.LastIndex(p, delimiter)])
		}
		return !lastPage
	})

	if err != nil {
		err = s.convertError(err)
		return nil, emperror.With(emperror.Wrap(err, "error getting prefixes for bucket"), "bucket", bucketName, "delimeter", delimiter)
	}

	return prefixes, nil
}

// GetObject retrieves the object by it's key from the given bucket
func (s *objectStore) GetObject(bucketName string, key string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	})
	if err != nil {
		err = s.convertError(err)
		return nil, emperror.With(emperror.Wrap(err, "error getting object"), "bucket", bucketName, "object", key)
	}

	return output.Body, nil
}

// PutObject creates a new object using the data in body with the given key
func (s *objectStore) PutObject(bucketName string, key string, body io.Reader) error {
	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &key,
		Body:   body,
	})
	if err != nil {
		err = s.convertError(err)
		return emperror.With(emperror.Wrap(err, "error putting object"), "bucket", bucketName, "object", key)
	}

	if s.waitForCompletion {
		err := s.client.WaitUntilObjectExists(&s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		})
		if err != nil {
			return emperror.With(emperror.Wrap(err, "could not wait for object to be created"), "bucket", bucketName, "object", key)
		}
	}

	return nil
}

// DeleteObject deletes the object from the given bucket by it's key
func (s *objectStore) DeleteObject(bucketName string, key string) error {
	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	})
	if err != nil {
		err = s.convertError(err)
		emperror.With(emperror.Wrap(err, "error deleting object"), "bucket", bucketName, "object", key)
	}

	if s.waitForCompletion {
		err := s.client.WaitUntilObjectNotExists(&s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		})
		if err != nil {
			return emperror.With(emperror.Wrap(err, "could not wait for object to be deleted"), "bucket", bucketName, "object", key)
		}
	}

	return nil
}

// GetSignedURL gives back a signed URL for the object that expires after the given ttl
func (s *objectStore) GetSignedURL(bucketName, key string, ttl time.Duration) (string, error) {
	req, _ := s.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})

	url, err := req.Presign(ttl)
	if err != nil {
		err = s.convertError(err)
		return "", emperror.With(emperror.Wrap(err, "could not get signed url"), "bucket", bucketName, "object", key)
	}

	return url, nil
}

func (s *objectStore) convertError(err error) error {

	if awsErr, ok := err.(awserr.Error); ok {
		switch awsErr.Code() {
		case s3.ErrCodeBucketAlreadyExists:
		case s3.ErrCodeBucketAlreadyOwnedByYou:
			err = errBucketAlreadyExists{}
		case s3.ErrCodeNoSuchBucket:
		case "NotFound":
			err = errBucketNotFound{}
		case s3.ErrCodeNoSuchKey:
			err = errObjectNotFound{}
		}
	}

	return err
}
