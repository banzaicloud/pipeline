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
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	azurePipeline "github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-blob-go/azblob"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/goph/emperror"
)

const (
	containerUrlTemplate = "https://%s.blob.core.windows.net/%s"
	serviceUrlTemplate   = "https://%s.blob.core.windows.net"
	fileUrlTemplate      = "https://%s.blob.core.windows.net/%s/%s"
)

type objectStore struct {
	config      Config
	credentials azure.Credentials
}

// Config defines configuration
type Config struct {
	ResourceGroup  string
	StorageAccount string
	Location       string
}

// NewPlainObjectStore creates an objectstore with no configuration.
// Instances created with this function may be used to access methods that don't explicitly access external (cloud) resources
func NewPlainObjectStore() (*objectStore, error) {
	return &objectStore{}, nil
}

// New returns an Object Store instance that manages Azure object store
func New(config Config, credentials azure.Credentials) *objectStore {
	return &objectStore{
		config:      config,
		credentials: credentials,
	}
}

// CreateBucket creates a new bucket in the object store
func (o *objectStore) CreateBucket(bucketName string) error {
	p, err := o.createAzurePipeline()
	if err != nil {
		return emperror.Wrap(err, "failed to create azure pipeline")
	}

	URL, err := url.Parse(fmt.Sprintf(containerUrlTemplate, o.config.StorageAccount, bucketName))
	if err != nil {
		return err
	}
	containerURL := azblob.NewContainerURL(*URL, p)

	_, err = containerURL.GetProperties(context.TODO(), azblob.LeaseAccessConditions{})
	if err != nil {
		if err.(azblob.StorageError).ServiceCode() == azblob.ServiceCodeContainerNotFound { // Bucket not found, so create it
			_, err = containerURL.Create(context.TODO(), azblob.Metadata{}, azblob.PublicAccessNone)
			if err != nil {
				return emperror.WrapWith(err, "failed to create bucket",
					"resource-group", o.config.ResourceGroup, "bucket", bucketName,
				)
			}
		} else {
			return err
		}
	} else {
		return errBucketAlreadyExists{}
	}

	return nil
}

func (o *objectStore) createAzurePipeline() (azurePipeline.Pipeline, error) {
	storageAccountClient, err := NewAuthorizedStorageAccountClientFromSecret(o.credentials)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create storage account client")
	}

	key, err := storageAccountClient.GetStorageAccountKey(o.config.ResourceGroup, o.config.StorageAccount)
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to get storage account key",
			"resource-group", o.config.ResourceGroup,
		)
	}

	credential, err := azblob.NewSharedKeyCredential(o.config.StorageAccount, key)
	if err != nil {
		return nil, err
	}

	return azblob.NewPipeline(credential, azblob.PipelineOptions{}), nil
}

// ListBuckets lists the current buckets in the object store
func (o *objectStore) ListBuckets() ([]string, error) {
	buckets := make([]string, 0)

	p, err := o.createAzurePipeline()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create azure pipeline")
	}

	URL, err := url.Parse(fmt.Sprintf(serviceUrlTemplate, o.config.StorageAccount))
	if err != nil {
		return nil, err
	}

	serviceURL := azblob.NewServiceURL(*URL, p)

	list, err := serviceURL.ListContainersSegment(context.TODO(), azblob.Marker{}, azblob.ListContainersSegmentOptions{})
	if err != nil {
		return nil, err
	}

	for _, item := range list.ContainerItems {
		buckets = append(buckets, item.Name)
	}

	return buckets, nil
}

// CheckBucket checks the status of the given bucket
func (o *objectStore) CheckBucket(bucketName string) error {
	p, err := o.createAzurePipeline()
	if err != nil {
		return emperror.Wrap(err, "failed to create azure pipeline")
	}

	URL, err := url.Parse(fmt.Sprintf(containerUrlTemplate, o.config.StorageAccount, bucketName))
	if err != nil {
		return err
	}
	containerURL := azblob.NewContainerURL(*URL, p)

	_, err = containerURL.GetProperties(context.TODO(), azblob.LeaseAccessConditions{})
	if err != nil {
		if err.(azblob.StorageError).ServiceCode() == azblob.ServiceCodeContainerNotFound {
			return emperror.With(errBucketNotFound{}, "bucket", bucketName)
		}
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

	p, err := o.createAzurePipeline()
	if err != nil {
		return emperror.Wrap(err, "failed to create azure pipeline")
	}

	URL, err := url.Parse(fmt.Sprintf(containerUrlTemplate, o.config.StorageAccount, bucketName))
	if err != nil {
		return err
	}
	containerURL := azblob.NewContainerURL(*URL, p)

	_, err = containerURL.Delete(context.TODO(), azblob.ContainerAccessConditions{})
	if err != nil {
		return emperror.WrapWith(err, "failed to delete bucket",
			"resource-group", o.config.ResourceGroup, "bucket", bucketName,
		)
	}

	return nil
}

// ListObjects gets all keys in the bucket
func (o *objectStore) ListObjects(bucketName string) ([]string, error) {
	blobs := make([]string, 0)

	p, err := o.createAzurePipeline()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create azure pipeline")
	}

	URL, err := url.Parse(fmt.Sprintf(containerUrlTemplate, o.config.StorageAccount, bucketName))
	if err != nil {
		return nil, err
	}
	containerURL := azblob.NewContainerURL(*URL, p)

	list, err := containerURL.ListBlobsFlatSegment(context.TODO(), azblob.Marker{}, azblob.ListBlobsSegmentOptions{})
	if err != nil {
		return nil, err
	}
	for _, item := range list.Segment.BlobItems {
		blobs = append(blobs, item.Name)
	}

	return blobs, nil
}

// ListObjectsWithPrefix gets all keys with the given prefix from the bucket
func (o *objectStore) ListObjectsWithPrefix(bucketName, prefix string) ([]string, error) {
	blobs := make([]string, 0)

	p, err := o.createAzurePipeline()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create azure pipeline")
	}

	URL, err := url.Parse(fmt.Sprintf(containerUrlTemplate, o.config.StorageAccount, bucketName))
	if err != nil {
		return nil, err
	}
	containerURL := azblob.NewContainerURL(*URL, p)

	list, err := containerURL.ListBlobsFlatSegment(context.TODO(), azblob.Marker{}, azblob.ListBlobsSegmentOptions{
		Prefix: prefix,
	})
	if err != nil {
		return nil, err
	}
	for _, item := range list.Segment.BlobItems {
		blobs = append(blobs, item.Name)
	}

	return blobs, nil
}

// ListObjectKeyPrefixes gets a list of all object key prefixes that come before the provided delimiter
func (o *objectStore) ListObjectKeyPrefixes(bucketName string, delimiter string) ([]string, error) {
	var prefixes []string

	p, err := o.createAzurePipeline()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create azure pipeline")
	}

	URL, err := url.Parse(fmt.Sprintf(containerUrlTemplate, o.config.StorageAccount, bucketName))
	if err != nil {
		return nil, err
	}
	containerURL := azblob.NewContainerURL(*URL, p)

	list, err := containerURL.ListBlobsHierarchySegment(context.TODO(), azblob.Marker{}, delimiter, azblob.ListBlobsSegmentOptions{})
	if err != nil {
		err = o.convertError(err)
		return nil, emperror.WrapWith(err, "error getting prefixes for bucket", "bucket", bucketName, "delimiter", delimiter)
	}

	for _, prefix := range list.Segment.BlobPrefixes {
		prefixes = append(prefixes, prefix.Name[0:strings.LastIndex(prefix.Name, delimiter)])
	}

	return prefixes, nil
}

// GetObject retrieves the object by it's key from the given bucket
func (o *objectStore) GetObject(bucketName string, key string) (io.ReadCloser, error) {
	p, err := o.createAzurePipeline()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create azure pipeline")
	}

	URL, err := url.Parse(fmt.Sprintf(fileUrlTemplate, o.config.StorageAccount, bucketName, key))
	if err != nil {
		return nil, err
	}

	blobURL := azblob.NewBlobURL(*URL, p)

	downloadResponse, err := blobURL.Download(context.TODO(), 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		err = o.convertError(err)
		return nil, emperror.WrapWith(err, "error getting object", "bucket", bucketName, "object", key)
	}

	return downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 3}), nil
}

// PutObject creates a new object using the data in body with the given key
func (o *objectStore) PutObject(bucketName string, key string, body io.Reader) error {
	p, err := o.createAzurePipeline()
	if err != nil {
		return emperror.Wrap(err, "failed to create azure pipeline")
	}

	URL, err := url.Parse(fmt.Sprintf(fileUrlTemplate, o.config.StorageAccount, bucketName, key))
	if err != nil {
		return err
	}
	blobURL := azblob.NewBlockBlobURL(*URL, p)

	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	_, err = azblob.UploadBufferToBlockBlob(context.TODO(), b, blobURL, azblob.UploadToBlockBlobOptions{})
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "error putting object", "bucket", bucketName, "object", key)
	}

	return nil
}

// DeleteObject deletes the object from the given bucket by it's key
func (o *objectStore) DeleteObject(bucketName string, key string) error {
	p, err := o.createAzurePipeline()
	if err != nil {
		return emperror.Wrap(err, "failed to create azure pipeline")
	}

	URL, err := url.Parse(fmt.Sprintf(fileUrlTemplate, o.config.StorageAccount, bucketName, key))
	if err != nil {
		return err
	}
	blobURL := azblob.NewBlobURL(*URL, p)

	_, err = blobURL.Delete(context.TODO(), "", azblob.BlobAccessConditions{})
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "error deleting object", "bucket", bucketName, "object", key)
	}

	return nil
}

// GetSignedURL gives back a signed URL for the object that expires after the given ttl
func (o *objectStore) GetSignedURL(bucketName, key string, ttl time.Duration) (string, error) {
	storageAccountClient, err := NewAuthorizedStorageAccountClientFromSecret(o.credentials)
	if err != nil {
		return "", emperror.Wrap(err, "failed to create storage account client")
	}

	skey, err := storageAccountClient.GetStorageAccountKey(o.config.ResourceGroup, o.config.StorageAccount)
	if err != nil {
		return "", emperror.WrapWith(err, "failed to get storage account key",
			"resource-group", o.config.ResourceGroup,
		)
	}

	credential, err := azblob.NewSharedKeyCredential(o.config.StorageAccount, skey)
	if err != nil {
		return "", err
	}

	sasQueryParams, err := azblob.BlobSASSignatureValues{
		Protocol:      azblob.SASProtocolHTTPS,
		ExpiryTime:    time.Now().Add(ttl),
		Permissions:   azblob.BlobSASPermissions{Add: true, Read: true, Write: true}.String(),
		ContainerName: bucketName,
		BlobName:      key,
	}.NewSASQueryParameters(credential)
	if err != nil {
		return "", err
	}

	qp := sasQueryParams.Encode()
	signedUrl := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s", o.config.StorageAccount, bucketName, key, qp)

	return signedUrl, nil
}

func (o *objectStore) convertError(err error) error {

	if azureErr, ok := err.(azblob.StorageError); ok {
		switch azureErr.ServiceCode() {
		case azblob.ServiceCodeContainerAlreadyExists:
			err = errBucketAlreadyExists{}
		case azblob.ServiceCodeContainerNotFound:
			err = errBucketNotFound{}
		case azblob.ServiceCodeBlobNotFound:
			err = errObjectNotFound{}
		}
	}

	return err
}
