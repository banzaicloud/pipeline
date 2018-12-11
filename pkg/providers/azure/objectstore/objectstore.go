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
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	mgmtStorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	falseVal = false
	trueVal  = true
)

type objectStore struct {
	config      Config
	credentials Credentials

	client *storage.BlobStorageClient
}

// Config defines configuration
type Config struct {
	ResourceGroup  string
	StorageAccount string
	Location       string
	Logger         logrus.FieldLogger
}

// Credentials represents credentials necessary for access
type Credentials struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
}

// NewPlainObjectStore creates an objectstore with no configuration.
// Instances created with this function may be used to access methods that don't explicitly access external (cloud) resources
func NewPlainObjectStore() (*objectStore, error) {
	return &objectStore{}, nil
}

// New returns an Object Store instance that manages Azure object store
func New(config Config, credentials Credentials) *objectStore {
	return &objectStore{
		config:      config,
		credentials: credentials,
	}
}

func (o *objectStore) createResourceGroup() error {
	logger := o.config.Logger

	logger.Info("creating resource group")

	authorizer, err := newAuthorizer(o.credentials)
	if err != nil {
		return fmt.Errorf("authentication failed: %s", err.Error())
	}

	gclient := resources.NewGroupsClient(o.credentials.SubscriptionID)

	gclient.Authorizer = authorizer
	res, _ := gclient.Get(context.TODO(), o.config.ResourceGroup)

	if res.StatusCode == http.StatusNotFound {
		result, err := gclient.CreateOrUpdate(
			context.TODO(),
			o.config.ResourceGroup,
			resources.Group{Location: to.StringPtr(o.config.Location)},
		)
		if err != nil {
			return err
		}
		logger.Info(result.Status)
	}
	logger.Info("resource group created")

	return nil
}

func (o *objectStore) checkStorageAccountExistence() (*bool, error) {
	logger := o.config.Logger
	storageAccountsClient, err := createStorageAccountClient(o.credentials)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create storage account client")
	}

	logger.Info("retrieving storage account name availability...")
	result, err := storageAccountsClient.CheckNameAvailability(
		context.TODO(),
		mgmtStorage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(o.config.StorageAccount),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
		},
	)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve storage account name availability")
	}

	if *result.NameAvailable == false {
		// account name is already taken or it is invalid
		// retrieve the storage account
		if _, err = storageAccountsClient.GetProperties(context.TODO(), o.config.ResourceGroup, o.config.StorageAccount); err != nil {
			logger.Errorf("could not retrieve storage account, %s", *result.Message)
			return nil, emperror.WrapWith(err, *result.Message, "storage_account", o.config.StorageAccount, "resource_group", o.config.ResourceGroup)
		}
		// storage name exists, available
		return &trueVal, nil
	}

	// storage name doesn't exist
	return &falseVal, nil
}

func (o *objectStore) createStorageAccount() error {
	logger := o.config.Logger
	storageAccountsClient, err := createStorageAccountClient(o.credentials)
	if err != nil {
		return err
	}

	logger.Info("creating storage account")

	future, err := storageAccountsClient.Create(
		context.TODO(),
		o.config.ResourceGroup,
		o.config.StorageAccount,
		mgmtStorage.AccountCreateParameters{
			Sku: &mgmtStorage.Sku{
				Name: mgmtStorage.StandardLRS,
			},
			Kind:     mgmtStorage.BlobStorage,
			Location: to.StringPtr(o.config.Location),
			AccountPropertiesCreateParameters: &mgmtStorage.AccountPropertiesCreateParameters{
				AccessTier: mgmtStorage.Hot,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("cannot create storage account: %v", err)
	}

	logger.Info("storage account creation request sent")
	if future.WaitForCompletionRef(context.TODO(), storageAccountsClient.Client) != nil {
		return err
	}

	logger.Info("storage account created")

	return nil
}

func (o *objectStore) createClient() (*storage.BlobStorageClient, error) {
	if err := o.createResourceGroup(); err != nil {
		return nil, emperror.Wrap(err, "failed to create resource group")
	}

	exists, err := o.checkStorageAccountExistence()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to check storage account")
	}

	if !*exists {
		if err = o.createStorageAccount(); err != nil {
			return nil, emperror.Wrap(err, "failed to create storage account")
		}
	}

	key, err := GetStorageAccountKey(o.config.ResourceGroup, o.config.StorageAccount, o.credentials, o.config.Logger)
	if err != nil {
		return nil, err
	}

	client, err := storage.NewBasicClient(o.config.StorageAccount, key)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create Azure client")
	}

	blobStorageClient := client.GetBlobService()

	return &blobStorageClient, nil
}

// CreateBucket creates a new bucket in the object store
func (o *objectStore) CreateBucket(bucketName string) error {
	client, err := o.createClient()
	if err != nil {
		return emperror.Wrap(err, "could not create Azure client")
	}

	err = client.GetContainerReference(bucketName).Create(&storage.CreateContainerOptions{})
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "bucket creation failed", "bucket", bucketName)
	}

	return nil
}

// ListBuckets lists the current buckets in the object store
func (o *objectStore) ListBuckets() ([]string, error) {
	buckets := make([]string, 0)

	client, err := o.createClient()
	if err != nil {
		return buckets, emperror.Wrap(err, "could not create Azure client")
	}
	resp, err := client.ListContainers(storage.ListContainersParameters{})
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
	client, err := o.createClient()
	if err != nil {
		return emperror.Wrap(err, "could not create Azure client")
	}

	found, err := client.GetContainerReference(bucketName).Exists()
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
	client, err := o.createClient()
	if err != nil {
		return emperror.Wrap(err, "could not create Azure client")
	}

	o.client = client

	obj, err := o.ListObjects(bucketName)
	if err != nil {
		return emperror.WrapWith(err, "failed to list objects", "bucket", bucketName)
	}

	if len(obj) > 0 {
		return emperror.With(pkgErrors.ErrorBucketDeleteNotEmpty, "bucket", bucketName)
	}

	err = client.GetContainerReference(bucketName).Delete(&storage.DeleteContainerOptions{})
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
	client, err := o.createClient()
	if err != nil {
		return nil, emperror.Wrap(err, "could not create Azure client")
	}

	res, err := client.GetContainerReference(bucketName).ListBlobs(storage.ListBlobsParameters{
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

	client, err := o.createClient()
	if err != nil {
		return nil, emperror.Wrap(err, "could not create Azure client")
	}

	response, err := client.GetContainerReference(bucketName).ListBlobs(storage.ListBlobsParameters{
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
	client, err := o.createClient()
	if err != nil {
		return nil, emperror.Wrap(err, "could not create Azure client")
	}

	blob := client.GetContainerReference(bucketName).GetBlobReference(key)

	res, err := blob.Get(nil)
	if err != nil {
		err = o.convertError(err)
		return nil, emperror.WrapWith(err, "error getting object", "bucket", bucketName, "object", key)
	}

	return res, nil
}

// PutObject creates a new object using the data in body with the given key
func (o *objectStore) PutObject(bucketName string, key string, body io.Reader) error {
	client, err := o.createClient()
	if err != nil {
		return emperror.Wrap(err, "could not create Azure client")
	}

	blob := client.GetContainerReference(bucketName).GetBlobReference(key)

	err = blob.CreateBlockBlobFromReader(body, nil)
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "error putting object", "bucket", bucketName, "object", key)
	}

	return nil
}

// DeleteObject deletes the object from the given bucket by it's key
func (o *objectStore) DeleteObject(bucketName string, key string) error {
	client, err := o.createClient()
	if err != nil {
		return emperror.Wrap(err, "could not create Azure client")
	}

	blob := client.GetContainerReference(bucketName).GetBlobReference(key)

	err = blob.Delete(nil)
	if err != nil {
		err = o.convertError(err)
		return emperror.WrapWith(err, "error deleting object", "bucket", bucketName, "object", key)
	}

	return nil
}

// GetSignedURL gives back a signed URL for the object that expires after the given ttl
func (o *objectStore) GetSignedURL(bucketName, key string, ttl time.Duration) (string, error) {
	client, err := o.createClient()
	if err != nil {
		return "", emperror.Wrap(err, "could not create Azure client")
	}

	blob := client.GetContainerReference(bucketName).GetBlobReference(key)

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

func GetStorageAccountKey(resourceGroup string, storageAccount string, credentials Credentials, log logrus.FieldLogger) (string, error) {
	client, err := createStorageAccountClient(credentials)
	if err != nil {
		return "", err
	}

	logger := log.WithFields(logrus.Fields{
		"resource_group":  resourceGroup,
		"storage_account": storageAccount,
	})

	logger.Info("getting key for storage account")

	keys, err := client.ListKeys(context.TODO(), resourceGroup, storageAccount)
	if err != nil {
		return "", errors.WithStack(err)
	}

	key := (*keys.Keys)[0].Value

	return *key, nil
}

// GetAllResourceGroups returns all resource groups using
// the Azure credentials referenced by the provided secret.
func GetAllResourceGroups(credentials Credentials) ([]*resources.Group, error) {
	rgClient := resources.NewGroupsClient(credentials.SubscriptionID)
	authorizer, err := newAuthorizer(credentials)
	if err != nil {
		return nil, err
	}

	rgClient.Authorizer = authorizer

	resourceGroupsPages, err := rgClient.List(context.TODO(), "", nil)
	if err != nil {
		return nil, err
	}

	var groups []*resources.Group
	for resourceGroupsPages.NotDone() {
		resourceGroupsChunk := resourceGroupsPages.Values()

		for i := 0; i < len(resourceGroupsChunk); i++ {
			groups = append(groups, &resourceGroupsChunk[i])
		}

		if err = resourceGroupsPages.NextWithContext(context.TODO()); err != nil {
			return nil, err
		}
	}

	return groups, nil
}

// GetAllStorageAccounts returns all storage accounts under the specified resource group
// using the Azure credentials referenced by the provided secret.
func GetAllStorageAccounts(credentials Credentials, resourceGroup string) (*[]mgmtStorage.Account, error) {
	client, err := createStorageAccountClient(credentials)
	if err != nil {
		return nil, err
	}

	storageAccountList, err := client.ListByResourceGroup(context.TODO(), resourceGroup)
	if err != nil {
		return nil, err
	}

	return storageAccountList.Value, nil
}

func createStorageAccountClient(credentials Credentials) (*mgmtStorage.AccountsClient, error) {
	accountClient := mgmtStorage.NewAccountsClient(credentials.SubscriptionID)

	authorizer, err := newAuthorizer(credentials)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	accountClient.Authorizer = authorizer

	return &accountClient, nil
}

func newAuthorizer(credentials Credentials) (autorest.Authorizer, error) {
	authorizer, err := auth.NewClientCredentialsConfig(
		credentials.ClientID,
		credentials.ClientSecret,
		credentials.TenantID).Authorizer()

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
