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

package azure

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/go-autorest/autorest/to"
	pipelineAuth "github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Default storage account name when none is provided.
// This must between 3-23 letters and can only contain small letters and numbers.
const defaultStorageAccountName = "pipelinegenstorageacc"

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }

// ObjectStore stores all required parameters for container creation.
//
// Note: calling methods on this struct is not thread safe currently.
type ObjectStore struct {
	storageAccount string
	resourceGroup  string
	location       string
	secret         *secret.SecretItemResponse

	org *pipelineAuth.Organization

	db     *gorm.DB
	logger logrus.FieldLogger
}

// NewObjectStore returns a new object store instance.
func NewObjectStore(
	location string,
	resourceGroup string,
	storageAccount string,
	secret *secret.SecretItemResponse,
	org *pipelineAuth.Organization,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *ObjectStore {
	return &ObjectStore{
		location:       location,
		resourceGroup:  resourceGroup,
		storageAccount: storageAccount,
		secret:         secret,
		db:             db,
		logger:         logger,
		org:            org,
	}
}

// getResourceGroup returns the given resource group or generates one.
func (s *ObjectStore) getResourceGroup() string {
	resourceGroup := s.resourceGroup

	// generate a default resource group name if none given
	if resourceGroup == "" {
		resourceGroup = fmt.Sprintf("pipeline-auto-%s", s.location)
	}

	return resourceGroup
}

// getStorageAccount returns the given storage account or or falls back to a default one.
func (s *ObjectStore) getStorageAccount() string {
	storageAccount := s.storageAccount

	if storageAccount == "" {
		storageAccount = defaultStorageAccountName
	}

	return storageAccount
}

func (s *ObjectStore) getLogger(bucketName string) logrus.FieldLogger {
	return s.logger.WithFields(logrus.Fields{
		"organization":    s.org.ID,
		"bucket":          bucketName,
		"resource_group":  s.getResourceGroup(),
		"storage_account": s.getStorageAccount(),
	})
}

func (s *ObjectStore) checkStorageAccountExistence(resourceGroup string, storageAccount string) (bool, error) {
	client, err := NewAuthorizedStorageAccountClientFromSecret(s.secret.Values)
	if err != nil {
		return false, err
	}

	logger := s.logger.WithFields(logrus.Fields{
		"resource_group":  resourceGroup,
		"storage_account": storageAccount,
	})

	logger.Info("checking storage account existence")

	result, err := client.CheckNameAvailability(
		context.TODO(),
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(storageAccount),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
		},
	)
	if err != nil {
		return false, err
	}

	logger.Info("storage account availability: ", result.Reason)

	if *result.NameAvailable == false {
		_, err := client.GetProperties(context.TODO(), resourceGroup, storageAccount)
		if err != nil {
			logger.Error("storage account exists but it is not in your resource group: ", err)

			return false, err
		}

		logger.Warnf("storage account name not available because: %s", *result.Message)

		return true, fmt.Errorf("storage account name %s is already taken", storageAccount)
	}

	return false, nil
}

// DeleteBucket deletes the Azure storage container identified by the specified name
// under the current resource group, storage account provided the storage container is of 'managed' type.
func (s *ObjectStore) DeleteBucket(bucketName string) error {
	resourceGroup := s.getResourceGroup()
	storageAccount := s.getStorageAccount()

	logger := s.getLogger(bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	logger.Info("looking for bucket")

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
	}

	key, err := GetStorageAccountKey(resourceGroup, storageAccount, s.secret, s.logger)
	if err != nil {
		return err
	}

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(storageAccount, key), azblob.PipelineOptions{})
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", storageAccount, bucketName))
	containerURL := azblob.NewContainerURL(*URL, p)

	_, err = containerURL.Delete(context.TODO(), azblob.ContainerAccessConditions{})

	if err != nil {
		return err
	}

	err = s.db.Delete(bucket).Error
	if err != nil {
		return fmt.Errorf("deleting bucket failed: %s", err.Error())
	}

	return nil
}

// CheckBucket checks the status of the given Azure blob.
func (s *ObjectStore) CheckBucket(bucketName string) error {
	resourceGroup := s.getResourceGroup()
	storageAccount := s.getStorageAccount()

	logger := s.getLogger(bucketName)
	logger.Info("looking for bucket")

	_, err := s.checkStorageAccountExistence(resourceGroup, storageAccount)
	if err != nil && !strings.Contains(err.Error(), "is already taken") {
		logger.Error(err.Error())

		return err
	}

	key, err := GetStorageAccountKey(resourceGroup, storageAccount, s.secret, s.logger)
	if err != nil {
		logger.Error(err.Error())

		return err
	}

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(s.storageAccount, key), azblob.PipelineOptions{})
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", s.storageAccount, bucketName))
	containerURL := azblob.NewContainerURL(*URL, p)

	_, err = containerURL.GetPropertiesAndMetadata(context.TODO(), azblob.LeaseAccessConditions{})
	if err != nil {
		return err
	}

	return nil
}

// ListBuckets returns a list of Azure storage containers buckets that can be accessed with the credentials
// referenced by the secret field. Azure storage containers buckets that were created by a user in the current
// org are marked as 'managed'.
func (s *ObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"organization":    s.org.ID,
		"subscription_id": s.secret.GetValue(pkgSecret.AzureSubscriptionId),
	})

	logger.Info("getting all resource groups for subscription")

	resourceGroups, err := getAllResourceGroups(s.secret)
	if err != nil {
		return nil, fmt.Errorf("getting all resource groups failed: %s", err.Error())
	}

	var buckets []*objectstore.BucketInfo

	for _, rg := range resourceGroups {
		logger.WithField("resource_group", *(rg.Name)).Info("getting all storage accounts under resource group")

		storageAccounts, err := getAllStorageAccounts(s.secret, *rg.Name)
		if err != nil {
			return nil, fmt.Errorf("getting all storage accounts under resource group=%s failed: %s", *(rg.Name), err.Error())
		}

		// get all Blob containers under the storage account
		for i := 0; i < len(*storageAccounts); i++ {
			accountName := *(*storageAccounts)[i].Name

			logger.WithFields(logrus.Fields{
				"resource_group":  *(rg.Name),
				"storage_account": accountName,
			}).Info("getting all blob containers under storage account")

			accountKey, err := GetStorageAccountKey(*rg.Name, accountName, s.secret, s.logger)
			if err != nil {
				return nil, fmt.Errorf("getting all storage accounts under resource group=%s, storage account=%s failed: %s", *(rg.Name), accountName, err.Error())
			}

			blobContainers, err := getAllBlobContainers(accountName, accountKey)
			if err != nil {
				return nil, fmt.Errorf("getting all storage accounts under resource group=%s, storage account=%s failed: %s", *(rg.Name), accountName, err.Error())
			}

			for i := 0; i < len(blobContainers); i++ {
				blobContainer := blobContainers[i]

				bucketInfo := &objectstore.BucketInfo{
					Name:    blobContainer.Name,
					Managed: false,
					Azure: &objectstore.BlobStoragePropsForAzure{
						StorageAccount: accountName,
						ResourceGroup:  *rg.Name,
					},
				}

				buckets = append(buckets, bucketInfo)
			}

		}
	}

	var objectStores []ObjectStoreBucketModel

	err = s.db.Where(&ObjectStoreBucketModel{OrganizationID: s.org.ID}).Order("resource_group asc, storage_account asc, name asc").Find(&objectStores).Error
	if err != nil {
		return nil, fmt.Errorf("retrieving managed buckets failed: %s", err.Error())
	}

	for _, bucketInfo := range buckets {
		// managedAzureBlobStores must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(objectStores), func(i int) bool {
			return strings.Compare(objectStores[i].ResourceGroup, bucketInfo.Azure.ResourceGroup) >= 0 &&
				strings.Compare(objectStores[i].StorageAccount, bucketInfo.Azure.StorageAccount) >= 0 &&
				strings.Compare(objectStores[i].Name, bucketInfo.Name) >= 0
		})

		if idx < len(objectStores) &&
			strings.Compare(objectStores[idx].ResourceGroup, bucketInfo.Azure.ResourceGroup) >= 0 &&
			strings.Compare(objectStores[idx].StorageAccount, bucketInfo.Azure.StorageAccount) >= 0 &&
			strings.Compare(objectStores[idx].Name, bucketInfo.Name) >= 0 {
			bucketInfo.Managed = true
		}
	}

	return buckets, nil
}

func GetStorageAccountKey(resourceGroup string, storageAccount string, s *secret.SecretItemResponse, log logrus.FieldLogger) (string, error) {
	client, err := NewAuthorizedStorageAccountClientFromSecret(s.Values)
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
		return "", errors.Wrap(err, "error retrieving keys for StorageAccount")
	}

	key := (*keys.Keys)[0].Value

	return *key, nil
}

// getAllResourceGroups returns all resource groups using
// the Azure credentials referenced by the provided secret.
func getAllResourceGroups(s *secret.SecretItemResponse) ([]*resources.Group, error) {
	client, err := NewAuthorizedResourceGroupClientFromSecret(s.Values)
	if err != nil {
		return nil, err
	}

	resourceGroupsPages, err := client.List(context.TODO(), "", nil)
	if err != nil {
		return nil, err
	}

	var groups []*resources.Group
	for resourceGroupsPages.NotDone() {
		resourceGroupsChunk := resourceGroupsPages.Values()

		for i := 0; i < len(resourceGroupsChunk); i++ {
			groups = append(groups, &resourceGroupsChunk[i])
		}

		if err = resourceGroupsPages.Next(); err != nil {
			return nil, err
		}
	}

	return groups, nil
}

// getAllStorageAccounts returns all storage accounts under the specified resource group
// using the Azure credentials referenced by the provided secret.
func getAllStorageAccounts(s *secret.SecretItemResponse, resourceGroup string) (*[]storage.Account, error) {
	client, err := NewAuthorizedStorageAccountClientFromSecret(s.Values)
	if err != nil {
		return nil, err
	}

	storageAccountList, err := client.ListByResourceGroup(context.TODO(), resourceGroup)
	if err != nil {
		return nil, err
	}

	return storageAccountList.Value, nil
}

// getAllBlobContainers returns all blob container that belong to the specified storage account using
// the given storage account key.
func getAllBlobContainers(storageAccount string, storageAccountKey string) ([]azblob.Container, error) {
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", storageAccount))

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(storageAccount, storageAccountKey), azblob.PipelineOptions{})
	serviceURL := azblob.NewServiceURL(*u, p)

	resp, err := serviceURL.ListContainers(context.TODO(), azblob.Marker{}, azblob.ListContainersOptions{})
	if err != nil {
		return nil, err
	}

	return resp.Containers, nil
}

// searchCriteria returns the database search criteria to find a bucket with the given name
// within the scope of the specified resource group and storage account.
func (s *ObjectStore) searchCriteria(bucketName string) *ObjectStoreBucketModel {
	return &ObjectStoreBucketModel{
		OrganizationID: s.org.ID,
		Name:           bucketName,
		ResourceGroup:  s.getResourceGroup(),
		StorageAccount: s.getStorageAccount(),
	}
}
