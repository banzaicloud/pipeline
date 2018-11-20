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
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"time"

	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	"github.com/goph/emperror"
	"github.com/oracle/oci-go-sdk/common"
)

type objectStore struct {
	config      Config
	credentials Credentials

	client   *oci.OCI
	osClient *oci.ObjectStorage
}

// Config defines configuration
type Config struct {
	Region string
}

// Credentials represents credentials necessary for access
type Credentials struct {
	UserOCID          string
	TenancyOCID       string
	APIKey            string
	APIKeyFingerprint string
	CompartmentOCID   string
}

// New returns an objectStore instance that manages Oracle object store
func New(config Config, credentials Credentials) (*objectStore, error) {
	client, err := newClient(config, credentials)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get oci client")
	}

	osClient, err := client.NewObjectStorageClient()
	if err != nil {
		return nil, emperror.Wrap(err, "could not get object store client")
	}

	return &objectStore{
		config:      config,
		credentials: credentials,

		client:   client,
		osClient: osClient,
	}, nil
}

func newClient(config Config, credentials Credentials) (*oci.OCI, error) {
	client, err := oci.NewOCI(&oci.Credential{
		UserOCID:          credentials.UserOCID,
		TenancyOCID:       credentials.TenancyOCID,
		CompartmentOCID:   credentials.CompartmentOCID,
		APIKey:            credentials.APIKey,
		APIKeyFingerprint: credentials.APIKeyFingerprint,
		Region:            config.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

// CreateBucket creates a new bucket in the object store
func (o *objectStore) CreateBucket(bucketName string) error {
	_, err := o.osClient.CreateBucket(bucketName)
	if err != nil {
		return emperror.Wrap(o.convertBucketError(err, bucketName), "could not create bucket")
	}

	return nil
}

// ListBuckets lists the current buckets in the object store
func (o *objectStore) ListBuckets() ([]string, error) {
	var keys []string

	buckets, err := o.osClient.GetBuckets()
	if err != nil {
		return nil, emperror.Wrap(err, "could not list buckets")
	}

	for _, bucket := range buckets {
		if bucket.Name != nil {
			keys = append(keys, *bucket.Name)
		}
	}

	return keys, nil
}

// CheckBucket checks the status of the given bucket
func (o *objectStore) CheckBucket(bucketName string) (err error) {
	_, err = o.osClient.GetBucket(bucketName)
	if err != nil {
		return emperror.Wrap(o.convertBucketError(err, bucketName), "could not check bucket")
	}

	return nil
}

// DeleteBucket removes a bucket from the object store
func (o *objectStore) DeleteBucket(bucketName string) error {
	obj, err := o.ListObjects(bucketName)
	if err != nil {
		return emperror.With(emperror.Wrap(err, "could not list objects"), "bucket", bucketName)
	}

	if len(obj) > 0 {
		return emperror.With(pkgErrors.ErrorBucketDeleteNotEmpty, "bucket", bucketName)
	}

	err = o.osClient.DeleteBucket(bucketName)
	if err != nil {
		return emperror.Wrap(o.convertBucketError(err, bucketName), "could not delete bucket")
	}

	return nil
}

// ListObjects gets all keys in the bucket
func (o *objectStore) ListObjects(bucketName string) ([]string, error) {
	var keys []string

	objects, err := o.osClient.ListObjects(bucketName)
	if err != nil {
		return nil, emperror.Wrap(o.convertBucketError(err, bucketName), "could not list objects")
	}

	for _, object := range objects {
		keys = append(keys, *object.Name)
	}

	return keys, nil
}

// ListObjectsWithPrefix gets all keys with the given prefix from the bucket
func (o *objectStore) ListObjectsWithPrefix(bucketName, prefix string) ([]string, error) {
	var keys []string

	objects, err := o.osClient.ListObjectsWithPrefix(bucketName, prefix)
	if err != nil {
		return nil, emperror.With(emperror.Wrap(o.convertBucketError(err, bucketName), "could not list objects"), "prefix", prefix)
	}

	for _, object := range objects {
		keys = append(keys, *object.Name)
	}

	return keys, nil
}

// ListObjectKeyPrefixes gets a list of all object key prefixes that come before the provided delimiter
func (o *objectStore) ListObjectKeyPrefixes(bucketName, delimiter string) ([]string, error) {
	var prefixes []string

	oprefixes, err := o.osClient.ListObjectKeyPrefixes(bucketName, delimiter)
	if err != nil {
		return nil, emperror.With(emperror.Wrap(o.convertBucketError(err, bucketName), "could not list object key prefixes"), "delimeter", delimiter)
	}

	for _, prefix := range oprefixes {
		prefixes = append(prefixes, prefix[0:strings.LastIndex(prefix, delimiter)])
	}

	return prefixes, nil
}

// GetObject retrieves the object by it's key from the given bucket
func (o *objectStore) GetObject(bucketName string, key string) (io.ReadCloser, error) {
	reader, err := o.osClient.GetObject(bucketName, key)
	if err != nil {
		return nil, emperror.Wrap(o.convertObjectError(err, bucketName, key), "could not get object")
	}

	return reader, nil
}

// PutObject creates a new object using the data in body with the given key
func (o *objectStore) PutObject(bucketName string, key string, body io.Reader) error {

	buf := &bytes.Buffer{}
	length, err := io.Copy(buf, body)
	if err != nil {
		return emperror.Wrap(o.convertObjectError(err, bucketName, key), "could not create object")
	}

	err = o.osClient.PutObject(bucketName, key, length, ioutil.NopCloser(buf))
	if err != nil {
		return emperror.Wrap(o.convertObjectError(err, bucketName, key), "could not create object")
	}

	return nil
}

// DeleteObject deletes the object from the given bucket by it's key
func (o *objectStore) DeleteObject(bucketName string, key string) error {
	err := o.osClient.DeleteObject(bucketName, key)
	if err != nil {
		return emperror.Wrap(o.convertObjectError(err, bucketName, key), "could not delete object")
	}

	return nil
}

// GetSignedURL gives back a signed URL for the object that expires after the given ttl
func (o *objectStore) GetSignedURL(bucketName, key string, ttl time.Duration) (string, error) {
	url, err := o.osClient.GetSignedURL(bucketName, key, ttl)
	if err != nil {
		return "", emperror.Wrap(o.convertObjectError(err, bucketName, key), "could not get signed url")
	}

	return url, nil
}

func (o *objectStore) convertBucketError(err error, bucketName string) error {
	if ociErr, ok := err.(common.ServiceError); ok {
		switch ociErr.GetCode() {
		case "BucketNotFound":
			return errBucketNotFound{bucketName: bucketName}
		case "BucketAlreadyExists":
			return errBucketAlreadyExists{bucketName: bucketName}
		}
	}

	return emperror.With(err, "bucketName", bucketName)
}

func (o *objectStore) convertObjectError(err error, bucketName, objectName string) error {
	if ociErr, ok := err.(common.ServiceError); ok {
		switch ociErr.GetCode() {
		case "ObjectNotFound":
			return errObjectNotFound{bucketName: bucketName, objectName: objectName}
		case "BucketNotFound":
			return o.convertBucketError(err, bucketName)
		}
	}

	return emperror.With(err, "bucket", bucketName, "object", objectName)
}
