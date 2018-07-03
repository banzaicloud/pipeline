package oci

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/objectstorage"
)

type ObjectStorage struct {
	oci             *OCI
	client          *objectstorage.ObjectStorageClient
	CompartmentOCID string
	Namespace       string
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

	client.Namespace = *r.Value
	client.CompartmentOCID = oci.CompartmentOCID

	return client, nil
}

// GetBucket gets bucket by name
func (c *ObjectStorage) GetBucket(name string) (bucket objectstorage.Bucket, err error) {

	request := objectstorage.GetBucketRequest{
		NamespaceName: &c.Namespace,
	}
	request.BucketName = &name

	r, err := c.client.GetBucket(context.Background(), request)
	if err != nil {
		return bucket, err
	}

	if *r.CompartmentId != c.CompartmentOCID {
		return r.Bucket, fmt.Errorf("Service error:BucketNotFound. The bucket '%s' does not exist in compartment '%s' in namespace '%s'.", name, c.CompartmentOCID, *request.NamespaceName)
	}

	return r.Bucket, nil
}

// CreateBucket creates a bucket with the given name
func (c *ObjectStorage) CreateBucket(name string) error {

	request := objectstorage.CreateBucketRequest{
		NamespaceName: &c.Namespace,
	}
	request.CompartmentId = &c.CompartmentOCID
	request.Name = &name
	request.PublicAccessType = objectstorage.CreateBucketDetailsPublicAccessTypeNopublicaccess

	_, err := c.client.CreateBucket(context.Background(), request)
	if err != nil {
		return err
	}

	return nil
}

// DeleteBucket deletes a bucket by name
func (c *ObjectStorage) DeleteBucket(name string) error {

	request := objectstorage.DeleteBucketRequest{
		NamespaceName: &c.Namespace,
		BucketName:    &name,
	}
	_, err := c.client.DeleteBucket(context.Background(), request)

	return err
}

// ListBuckets gets object store buckets
func (c *ObjectStorage) ListBuckets() (buckets []objectstorage.BucketSummary, err error) {

	request := objectstorage.ListBucketsRequest{
		CompartmentId: common.String(c.CompartmentOCID),
		NamespaceName: common.String(c.Namespace),
	}
	request.Limit = common.Int(1)

	listFunc := func(request objectstorage.ListBucketsRequest) (objectstorage.ListBucketsResponse, error) {
		return c.client.ListBuckets(context.Background(), request)
	}

	buckets = make([]objectstorage.BucketSummary, 0)
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
