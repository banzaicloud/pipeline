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

	mgmtStorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

type objectStore struct {
	config      Config
	credentials Credentials

	client *storage.BlobStorageClient

	storageAccountKey string
}

// Config defines configuration
type Config struct {
	ResourceGroup  string
	StorageAccount string
}

// Credentials represents credentials necessary for access
type Credentials struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
}

// New returns an Object Store instance that manages Azure object store
func New(config Config, credentials Credentials) (*objectStore, error) {
	o := &objectStore{
		config:      config,
		credentials: credentials,
	}

	key, err := o.getStorageAccountKey()
	if err != nil {
		return nil, err
	}
	o.storageAccountKey = key

	client, err := storage.NewBasicClient(o.config.StorageAccount, o.storageAccountKey)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create Azure client")
	}

	blobStorageClient := client.GetBlobService()
	o.client = &blobStorageClient

	return o, nil
}

// CreateBucket creates a new bucket in the object store
func (o *objectStore) CreateBucket(bucketName string) error {
	err := o.client.GetContainerReference(bucketName).Create(&storage.CreateContainerOptions{})
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "bucket creation failed", "bucket", bucketName)
	}

	return nil
}

// ListBuckets lists the current buckets in the object store
func (o *objectStore) ListBuckets() ([]string, error) {
	buckets := make([]string, 0)

	resp, err := o.client.ListContainers(storage.ListContainersParameters{})
	if err != nil {
		return buckets, emperror.Wrap(err, "could not list buckets")
	}

	for _, container := range resp.Containers {
		buckets = append(buckets, container.Name)
	}

	return buckets, nil
}

// CheckBucket checks the status of the given bucket
func (o *objectStore) CheckBucket(bucketName string) error {
	found, err := o.client.GetContainerReference(bucketName).Exists()
	if !found {
		err = errBucketNotFound{}
	}

	if err != nil {
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

	err = o.client.GetContainerReference(bucketName).Delete(&storage.DeleteContainerOptions{})
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "bucket deletion failed", "bucket", bucketName)
	}

	return nil
}

// ListObjects gets all keys in the bucket
func (o *objectStore) ListObjects(bucketName string) ([]string, error) {
	res, err := o.client.GetContainerReference(bucketName).ListBlobs(storage.ListBlobsParameters{})
	if err != nil {
		err = o.convertError(err)
		return nil, emperror.WrapWith(err, "error listing object for bucket", "bucket", bucketName)
	}

	ret := make([]string, 0, len(res.Blobs))
	for _, blob := range res.Blobs {
		ret = append(ret, blob.Name)
	}

	return ret, nil
}

// ListObjectsWithPrefix gets all keys with the given prefix from the bucket
func (o *objectStore) ListObjectsWithPrefix(bucketName, prefix string) ([]string, error) {
	res, err := o.client.GetContainerReference(bucketName).ListBlobs(storage.ListBlobsParameters{
		Prefix: prefix,
	})
	if err != nil {
		err = o.convertError(err)
		return nil, emperror.WrapWith(err, "error listing object for bucket", "bucket", bucketName, "prefix", prefix)
	}

	ret := make([]string, 0, len(res.Blobs))
	for _, blob := range res.Blobs {
		ret = append(ret, blob.Name)
	}

	return ret, nil
}

// ListObjectKeyPrefixes gets a list of all object key prefixes that come before the provided delimiter
func (o *objectStore) ListObjectKeyPrefixes(bucketName string, delimiter string) ([]string, error) {
	var prefixes []string

	response, err := o.client.GetContainerReference(bucketName).ListBlobs(storage.ListBlobsParameters{
		Delimiter: delimiter,
	})
	if err != nil {
		err = o.convertError(err)
		return nil, emperror.WrapWith(err, "error getting prefixes for bucket", "bucket", bucketName, "delimeter", delimiter)
	}

	for _, prefix := range response.BlobPrefixes {
		prefixes = append(prefixes, prefix[0:strings.LastIndex(prefix, delimiter)])
	}

	return prefixes, nil
}

// GetObject retrieves the object by it's key from the given bucket
func (o *objectStore) GetObject(bucketName string, key string) (io.ReadCloser, error) {
	blob := o.client.GetContainerReference(bucketName).GetBlobReference(key)

	res, err := blob.Get(nil)
	if err != nil {
		err = o.convertError(err)
		return nil, emperror.WrapWith(err, "error getting object", "bucket", bucketName, "object", key)
	}

	return res, nil
}

// PutObject creates a new object using the data in body with the given key
func (o *objectStore) PutObject(bucketName string, key string, body io.Reader) error {
	blob := o.client.GetContainerReference(bucketName).GetBlobReference(key)

	err := blob.CreateBlockBlobFromReader(body, nil)
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "error putting object", "bucket", bucketName, "object", key)
	}

	return nil
}

// DeleteObject deletes the object from the given bucket by it's key
func (o *objectStore) DeleteObject(bucketName string, key string) error {
	blob := o.client.GetContainerReference(bucketName).GetBlobReference(key)

	err := blob.Delete(nil)
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "error deleting object", "bucket", bucketName, "object", key)
	}

	return nil
}

// GetSignedURL gives back a signed URL for the object that expires after the given ttl
func (o *objectStore) GetSignedURL(bucketName, key string, ttl time.Duration) (string, error) {
	blob := o.client.GetContainerReference(bucketName).GetBlobReference(key)

	url, err := blob.GetSASURI(storage.BlobSASOptions{
		BlobServiceSASPermissions: storage.BlobServiceSASPermissions{
			Read: true,
		},
		SASOptions: storage.SASOptions{
			Expiry: time.Now().Add(ttl),
		},
	})
	if err != nil {
		err = o.convertError(err)
		return "", emperror.WrapWith(err, "could not get signed url", "bucket", bucketName, "object", key)
	}

	return url, nil
}

func (o *objectStore) getStorageAccountKey() (string, error) {
	client, err := o.createStorageAccountClient()
	if err != nil {
		return "", err
	}

	keys, err := client.ListKeys(context.TODO(), o.config.ResourceGroup, o.config.StorageAccount)
	if err != nil {
		return "", errors.WithStack(err)
	}

	key := (*keys.Keys)[0].Value

	return *key, nil
}

func (o *objectStore) createStorageAccountClient() (*mgmtStorage.AccountsClient, error) {
	accountClient := mgmtStorage.NewAccountsClient(o.credentials.SubscriptionID)

	authorizer, err := o.newAuthorizer()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	accountClient.Authorizer = authorizer

	return &accountClient, nil
}

func (o *objectStore) newAuthorizer() (autorest.Authorizer, error) {
	authorizer, err := auth.NewClientCredentialsConfig(
		o.credentials.ClientID,
		o.credentials.ClientSecret,
		o.credentials.TenantID).Authorizer()

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return authorizer, nil
}

func (o *objectStore) convertError(err error) error {

	if azureErr, ok := err.(storage.AzureStorageServiceError); ok {
		switch azureErr.Code {
		case string(azblob.ServiceCodeContainerAlreadyExists):
			err = errBucketAlreadyExists{}
		case string(azblob.ServiceCodeContainerNotFound):
			err = errBucketNotFound{}
		case string(azblob.ServiceCodeBlobNotFound):
			err = errObjectNotFound{}
		}
	}

	return err
}
