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
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/goph/emperror"
)

const endpointURLTemplate = "https://oss-%s.aliyuncs.com"

type objectStore struct {
	config      Config
	credentials Credentials

	client *oss.Client
}

// Config defines configuration
type Config struct {
	Region string
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

// New returns an Object Store instance that manages Alibaba object store
func New(config Config, credentials Credentials) (*objectStore, error) {
	client, err := newClient(config, credentials)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create Alibaba client")
	}

	return &objectStore{
		config:      config,
		credentials: credentials,

		client: client,
	}, nil
}

func newClient(config Config, credentials Credentials) (*oss.Client, error) {
	endpoint := fmt.Sprintf(endpointURLTemplate, config.Region)

	client, err := oss.New(endpoint, credentials.AccessKeyID, credentials.SecretAccessKey)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// CreateBucket creates a new bucket in the object store
func (o *objectStore) CreateBucket(bucketName string) (err error) {
	err = o.client.CreateBucket(bucketName)
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "bucket creation failed", "bucket", bucketName)
	}

	return nil
}

// ListBuckets lists the current buckets in the object store
func (o *objectStore) ListBuckets() ([]string, error) {
	buckets := make([]string, 0)

	result, err := o.client.ListBuckets()
	if err != nil {
		return nil, emperror.Wrap(err, "could not list buckets")
	}

	for _, bucket := range result.Buckets {
		buckets = append(buckets, bucket.Name)
	}

	return buckets, nil
}

// CheckBucket checks the status of the given bucket
func (o *objectStore) CheckBucket(bucketName string) error {
	client, err := o.getClientForBucket(bucketName)
	if err != nil {
		return err
	}

	_, err = client.GetBucketInfo(bucketName)
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "checking bucket failed", "bucket", bucketName)
	}

	return nil
}

// DeleteBucket deletes a bucket from the object store
func (o *objectStore) DeleteBucket(bucketName string) error {
	obj, err := o.ListObjects(bucketName)
	if err != nil {
		return emperror.WrapWith(err, "failed to list objects", "bucket", bucketName)
	}

	if len(obj) > 0 {
		return emperror.With(pkgErrors.ErrorBucketDeleteNotEmpty, "bucket", bucketName)
	}

	client, err := o.getClientForBucket(bucketName)
	if err != nil {
		return err
	}

	err = client.DeleteBucket(bucketName)
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "bucket deletion failed", "bucket", bucketName)
	}

	return nil
}

// ListObjects gets all keys in the bucket
func (o *objectStore) ListObjects(bucketName string) ([]string, error) {
	var keys []string

	result, err := o.listObjectsWithOptions(bucketName)
	if err != nil {
		return nil, emperror.WrapWith(err, "error listing object for bucket", "bucket", bucketName)
	}

	for _, object := range result.Objects {
		keys = append(keys, object.Key)
	}

	return keys, nil
}

// ListObjectsWithPrefix gets all keys with the given prefix from the bucket
func (o *objectStore) ListObjectsWithPrefix(bucketName, prefix string) ([]string, error) {
	var keys []string

	result, err := o.listObjectsWithOptions(bucketName, oss.Prefix(prefix))
	if err != nil {
		return nil, emperror.WrapWith(err, "error listing object for bucket", "bucket", bucketName, "prefix", prefix)
	}

	for _, object := range result.Objects {
		keys = append(keys, object.Key)
	}

	return keys, nil
}

// ListObjectKeyPrefixes gets a list of all object key prefixes that come before the provided delimiter
func (o *objectStore) ListObjectKeyPrefixes(bucketName string, delimiter string) ([]string, error) {
	var prefixes []string

	result, err := o.listObjectsWithOptions(bucketName, oss.Delimiter(delimiter))
	if err != nil {
		return nil, emperror.WrapWith(err, "error getting prefixes for bucket", "bucket", bucketName, "delimeter", delimiter)
	}

	for _, prefix := range result.CommonPrefixes {
		prefixes = append(prefixes, prefix[0:strings.LastIndex(prefix, delimiter)])
	}

	return prefixes, nil
}

// GetObject retrieves the object by it's key from the given bucket
func (o *objectStore) GetObject(bucketName string, key string) (io.ReadCloser, error) {
	b, err := o.client.Bucket(bucketName)
	if err != nil {
		return nil, emperror.WrapWith(err, "error getting bucket instance", "bucket", bucketName)
	}

	reader, err := b.GetObject(key)
	if err != nil {
		err = o.convertError(err)
		return nil, emperror.WrapWith(err, "error getting object", "bucket", bucketName, "object", key)
	}

	return reader, nil
}

// PutObject creates a new object using the data in body with the given key
func (o *objectStore) PutObject(bucketName string, key string, body io.Reader) error {
	b, err := o.client.Bucket(bucketName)
	if err != nil {
		return emperror.WrapWith(err, "error getting bucket instance", "bucket", bucketName)
	}

	err = b.PutObject(key, body)
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "error putting object", "bucket", bucketName, "object", key)
	}

	return nil
}

// DeleteObject deletes the object from the given bucket by it's key
func (o *objectStore) DeleteObject(bucketName string, key string) error {
	b, err := o.client.Bucket(bucketName)
	if err != nil {
		return emperror.WrapWith(err, "error getting bucket instance", "bucket", bucketName)
	}

	err = b.DeleteObject(key)
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "error deleting object", "bucket", bucketName, "object", key)
	}

	return nil
}

// GetSignedURL gives back a signed URL for the object that expires after the given ttl
func (o *objectStore) GetSignedURL(bucketName, key string, ttl time.Duration) (string, error) {
	b, err := o.client.Bucket(bucketName)
	if err != nil {
		return "", emperror.WrapWith(err, "error getting bucket instance", "bucket", bucketName)
	}

	url, err := b.SignURL(key, oss.HTTPGet, int64(ttl.Seconds()))
	if err != nil {
		err = o.convertError(err)
		return "", emperror.WrapWith(err, "could not get signed url", "bucket", bucketName, "object", key)
	}

	return url, nil
}

func (o *objectStore) listObjectsWithOptions(bucketName string, options ...oss.Option) (oss.ListObjectsResult, error) {
	b, err := o.client.Bucket(bucketName)
	if err != nil {
		return oss.ListObjectsResult{}, err
	}

	result, err := b.ListObjects(options...)
	if err != nil {
		err = o.convertError(err)
		return oss.ListObjectsResult{}, err
	}

	return result, nil
}

func (o *objectStore) getClientForBucket(bucketName string) (*oss.Client, error) {
	result, err := o.client.GetBucketLocation(bucketName)
	if err != nil {
		err = o.convertError(err)
		return nil, emperror.WrapWith(err, "get bucket location failed", "bucket", bucketName)
	}
	location := strings.TrimPrefix(result, "oss-")

	return o.getClientForLocation(location)
}

func (o *objectStore) getClientForLocation(location string) (*oss.Client, error) {
	var err error
	client := o.client
	if location != o.config.Region {
		config := o.config
		config.Region = location
		client, err = newClient(config, o.credentials)
		if err != nil {
			return nil, emperror.WrapWith(err, "could not create Alibaba client for location", "location", location)
		}
	}

	return client, nil
}

func (o *objectStore) convertError(err error) error {

	if ossErr, ok := err.(oss.ServiceError); ok {
		switch ossErr.Code {
		case "BucketAlreadyExists":
			err = errBucketAlreadyExists{}
		case "NoSuchBucket":
			err = errBucketNotFound{}
		case "NoSuchKey":
			err = errObjectNotFound{}
		}
	}

	return err
}
