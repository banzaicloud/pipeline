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
func (os *ObjectStorage) DeleteBucket(name string) error {

	_, err := os.client.DeleteBucket(context.Background(), objectstorage.DeleteBucketRequest{
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
		return response.Bucket, fmt.Errorf("Service error:BucketNotFound. The bucket '%s' does not exist in compartment '%s' in namespace '%s'.", name, os.CompartmentOCID, *request.NamespaceName)
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
