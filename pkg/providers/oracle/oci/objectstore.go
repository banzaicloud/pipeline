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

package oci

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/objectstorage"
)

// ObjectStorage is for managing Object Storage related calls of OCI
type ObjectStorage struct {
	CompartmentOCID string
	Namespace       string

	oci    *OCI
	client *objectstorage.ObjectStorageClient
}

// NewObjectStorageClient creates new ObjectStorage
func (oci *OCI) NewObjectStorageClient() (client *ObjectStorage, err error) {
	client = &ObjectStorage{}

	oClient, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(oci.config)
	if err != nil {
		return client, err
	}

	client.client = &oClient

	r, err := oClient.GetNamespace(context.Background(), objectstorage.GetNamespaceRequest{})
	if err != nil {
		return client, err
	}

	client.oci = oci
	client.Namespace = *r.Value
	client.CompartmentOCID = oci.CompartmentOCID

	return client, nil
}

// CreateBucket creates a bucket with the given name
func (os *ObjectStorage) CreateBucket(name string) (bucket objectstorage.Bucket, err error) {
	response, err := os.client.CreateBucket(context.Background(), objectstorage.CreateBucketRequest{
		NamespaceName: &os.Namespace,
		CreateBucketDetails: objectstorage.CreateBucketDetails{
			CompartmentId:    &os.CompartmentOCID,
			Name:             &name,
			PublicAccessType: objectstorage.CreateBucketDetailsPublicAccessTypeNopublicaccess,
		},
	})
	if err != nil {
		return bucket, err
	}

	return response.Bucket, nil
}

// DeleteBucket deletes an Object Storage bucket by name
// it deletes existing PreauthenticatedRequests before trying to delete the bucket
func (os *ObjectStorage) DeleteBucket(name string) error {
	err := os.deletePreauthenticatedRequests(name)
	if err != nil {
		return err
	}

	_, err = os.client.DeleteBucket(context.Background(), objectstorage.DeleteBucketRequest{
		NamespaceName: &os.Namespace,
		BucketName:    &name,
	})

	return err
}

// GetBucket gets an Object Storage bucket by name
func (os *ObjectStorage) GetBucket(name string) (bucket objectstorage.Bucket, err error) {
	request := objectstorage.GetBucketRequest{
		NamespaceName: &os.Namespace,
		BucketName:    &name,
	}

	response, err := os.client.GetBucket(context.Background(), request)
	if err != nil {
		return bucket, err
	}

	if *response.CompartmentId != os.CompartmentOCID {
		return response.Bucket, &servicefailure{
			StatusCode: 404,
			Code:       "BucketNotFound",
			Message:    fmt.Sprintf("The bucket '%s' does not exist in compartment '%s' in namespace '%s'", name, os.CompartmentOCID, *request.NamespaceName),
		}
	}

	return response.Bucket, nil
}

// GetBuckets gets an Object Storage buckets
func (os *ObjectStorage) GetBuckets() (buckets []objectstorage.BucketSummary, err error) {
	request := objectstorage.ListBucketsRequest{
		CompartmentId: common.String(os.CompartmentOCID),
		NamespaceName: common.String(os.Namespace),
	}
	request.Limit = common.Int(20)

	listFunc := func(request objectstorage.ListBucketsRequest) (objectstorage.ListBucketsResponse, error) {
		return os.client.ListBuckets(context.Background(), request)
	}

	for r, err := listFunc(request); ; r, err = listFunc(request) {
		if err != nil {
			return buckets, err
		}

		for _, item := range r.Items {
			buckets = append(buckets, item)
		}

		if r.OpcNextPage != nil {
			// if there are more items in next page, fetch items from next page
			request.Page = r.OpcNextPage
		} else {
			// no more result, break the loop
			break
		}
	}

	return buckets, err
}

// ListObjects gets all keys in the bucket
func (os *ObjectStorage) ListObjects(bucket string) ([]objectstorage.ObjectSummary, error) {
	request := objectstorage.ListObjectsRequest{
		NamespaceName: &os.Namespace,
		BucketName:    &bucket,
	}

	response, err := os.client.ListObjects(context.Background(), request)
	if err != nil {
		return nil, err
	}

	return response.Objects, nil
}

// ListObjectsWithPrefix gets all keys with the given prefix from the bucket
func (os *ObjectStorage) ListObjectsWithPrefix(bucket, prefix string) ([]objectstorage.ObjectSummary, error) {
	request := objectstorage.ListObjectsRequest{
		NamespaceName: &os.Namespace,
		BucketName:    &bucket,
		Prefix:        &prefix,
	}

	response, err := os.client.ListObjects(context.Background(), request)
	if err != nil {
		return nil, err
	}

	return response.Objects, nil
}

// ListObjectKeyPrefixes gets a list of all object key prefixes that come before the provided delimiter.
func (os *ObjectStorage) ListObjectKeyPrefixes(bucket, delimeter string) ([]string, error) {
	request := objectstorage.ListObjectsRequest{
		NamespaceName: &os.Namespace,
		BucketName:    &bucket,
		Delimiter:     &delimeter,
	}

	response, err := os.client.ListObjects(context.Background(), request)
	if err != nil {
		return nil, err
	}

	return response.Prefixes, nil
}

// GetObject retrieves the object by it's key from the given bucket
func (os *ObjectStorage) GetObject(bucket, key string) (io.ReadCloser, error) {
	request := objectstorage.GetObjectRequest{
		NamespaceName: &os.Namespace,
		BucketName:    &bucket,
		ObjectName:    &key,
	}

	response, err := os.client.GetObject(context.Background(), request)
	if err != nil {
		return nil, err
	}

	return response.Content, nil
}

// PutObject creates a new object using the data in body with the given key
func (os *ObjectStorage) PutObject(bucket, key string, length int64, body io.ReadCloser) error {
	request := objectstorage.PutObjectRequest{
		NamespaceName: &os.Namespace,
		BucketName:    &bucket,
		ObjectName:    &key,
		ContentLength: &length,
		PutObjectBody: body,
	}

	_, err := os.client.PutObject(context.Background(), request)
	if err != nil {
		return err
	}

	return nil
}

// DeleteObject removes the object from the given bucket by it's key
func (os *ObjectStorage) DeleteObject(bucket, key string) error {
	request := objectstorage.DeleteObjectRequest{
		NamespaceName: &os.Namespace,
		BucketName:    &bucket,
		ObjectName:    &key,
	}

	_, err := os.client.DeleteObject(context.Background(), request)
	if err != nil {
		return err
	}

	return nil
}

// GetSignedURL gives back a signed URL for the key that expires after the given ttl
func (os *ObjectStorage) GetSignedURL(bucket, key string, ttl time.Duration) (string, error) {
	name := fmt.Sprintf("%s/%s@%d", bucket, key, time.Now().UnixNano())

	request := objectstorage.CreatePreauthenticatedRequestRequest{
		NamespaceName: &os.Namespace,
		BucketName:    &bucket,
		CreatePreauthenticatedRequestDetails: objectstorage.CreatePreauthenticatedRequestDetails{
			Name:       &name,
			ObjectName: &key,
			TimeExpires: &common.SDKTime{
				Time: time.Now().Add(ttl),
			},
			AccessType: objectstorage.CreatePreauthenticatedRequestDetailsAccessTypeObjectread,
		},
	}

	response, err := os.client.CreatePreauthenticatedRequest(context.Background(), request)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s%s", os.client.Host, *response.AccessUri), nil
}

func (os *ObjectStorage) deletePreauthenticatedRequests(name string) error {
	response, err := os.client.ListPreauthenticatedRequests(context.Background(), objectstorage.ListPreauthenticatedRequestsRequest{
		NamespaceName: &os.Namespace,
		BucketName:    &name,
		Limit:         common.Int(1000),
	})
	if err != nil {
		return err
	}

	for _, item := range response.Items {
		_, err := os.client.DeletePreauthenticatedRequest(context.Background(), objectstorage.DeletePreauthenticatedRequestRequest{
			NamespaceName: &os.Namespace,
			BucketName:    &name,
			ParId:         item.Id,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
