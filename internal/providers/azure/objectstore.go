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
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/banzaicloud/pipeline/pkg/providers/azure"

	pipelineAuth "github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	commonObjectstore "github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers"
	azureObjectstore "github.com/banzaicloud/pipeline/pkg/providers/azure/objectstore"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	alfanumericRegexp        = regexp.MustCompile(`[^a-zA-Z0-9]`)
	storageAccountNameRegexp = regexp.MustCompile(`^[a-z0-9]{3,24}$`)
)

// Default storage account name when none is provided.
// This must between 3-23 letters and can only contain small letters and numbers.
const defaultStorageAccountName = "pipelinegenstorageacc"
const defaultResourceGroupName = "pipelinegenresourcegroup"

type azureObjectStore interface {
	commonObjectstore.ObjectStore
}

type bucketNotFoundError struct{}

func (bucketNotFoundError) Error() string  { return "bucket not found" }
func (bucketNotFoundError) NotFound() bool { return true }

// ObjectStore stores all required parameters for container creation.
//
// Note: calling methods on this struct is not thread safe currently.
type ObjectStore struct {
	objectStore azureObjectStore

	storageAccount string
	resourceGroup  string
	location       string
	secret         *secret.SecretItemResponse

	org *pipelineAuth.Organization

	db     *gorm.DB
	logger logrus.FieldLogger
	force  bool
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
	force bool,
) (*ObjectStore, error) {
	storageAccount = getStorageAccount(storageAccount)
	resourceGroup = getResourceGroup(resourceGroup)
	location = getLocation(location)
	ostore, err := getProviderObjectStore(secret, resourceGroup, storageAccount, location, logger)
	if err != nil {
		return nil, errors.Wrap(err, "could not create Azure object storage client")
	}

	return &ObjectStore{
		objectStore:    ostore,
		location:       location,
		resourceGroup:  resourceGroup,
		storageAccount: storageAccount,
		secret:         secret,
		db:             db,
		logger:         logger,
		org:            org,
		force:          force,
	}, nil
}

// getResourceGroup returns the given resource group or generates one.
func getResourceGroup(resourceGroup string) string {
	// generate a default resource group name if none given
	if resourceGroup == "" {
		resourceGroup = defaultResourceGroupName
	}

	return resourceGroup
}

// GetStorageAccount returns the given storage account or falls back to a default one.
func getStorageAccount(storageAccount string) string {
	if storageAccount == "" {
		storageAccount = defaultStorageAccountName
	}

	return storageAccount
}

// getLocation returns the given location or falls back to a default one.
func getLocation(location string) string {
	if location == "" {
		location = "westeurope"
	}

	return location
}

func getProviderObjectStore(secret *secret.SecretItemResponse, resourceGroup, storageAccount, location string, logger logrus.FieldLogger) (azureObjectStore, error) {
	// when no secrets provided build an object store with no provider client/session setup
	// eg. usage: list managed buckets
	if secret == nil {
		return azureObjectstore.NewPlainObjectStore()
	}

	if !isValidStorageAccountName(storageAccount) {
		return nil, errors.New("storage account must be 3 to 24 characters long, and can contain only lowercase letters and numbers")
	}

	credentials := getCredentials(secret)

	config := azureObjectstore.Config{
		ResourceGroup:  resourceGroup,
		StorageAccount: storageAccount,
		Location:       location,
	}

	return azureObjectstore.New(config, credentials), nil
}

func (s *ObjectStore) getLogger() logrus.FieldLogger {
	var sId string
	if s.secret == nil {
		sId = ""
	} else {
		sId = s.secret.ID
	}

	return s.logger.WithFields(logrus.Fields{
		"organization":    s.org.ID,
		"secret":          sId,
		"resource_group":  s.resourceGroup,
		"storage_account": s.storageAccount,
		"location":        s.location,
	})
}

// https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-storage-account-name-errors
func isValidStorageAccountName(storageAccount string) bool {
	match := storageAccountNameRegexp.MatchString(storageAccount)

	return match
}

func getCredentials(secret *secret.SecretItemResponse) azure.Credentials {
	return *azure.NewCredentials(secret.Values)
}

// CreateBucket creates an Azure Object Store Blob with the provided name
// within a generated/provided ResourceGroup and StorageAccount
func (s *ObjectStore) CreateBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	dbr := s.db.Where(searchCriteria).Find(bucket)

	switch dbr.Error {
	case nil:
		return emperror.WrapWith(dbr.Error, "the bucket already exists", "bucket", bucketName)
	case gorm.ErrRecordNotFound:
		// proceed to creation
	default:
		return emperror.WrapWith(dbr.Error, "failed to retrieve bucket", "bucket", bucketName)
	}

	bucket.Name = bucketName
	bucket.ResourceGroup = s.resourceGroup
	bucket.StorageAccount = s.storageAccount
	bucket.Organization = *s.org
	bucket.SecretRef = s.secret.ID
	bucket.Status = providers.BucketCreating
	bucket.Location = s.location

	logger.Info("creating bucket...")

	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to save bucket", "bucket", bucketName)
	}

	err := s.createStorageAccountAndResourceGroup()
	if err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to create storage account or resource group"))
	}

	if err := s.objectStore.CreateBucket(bucketName); err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to create the bucket"))
	}

	storageAccountClient, err := azureObjectstore.NewAuthorizedStorageAccountClientFromSecret(getCredentials(s.secret))
	if err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to create storage account client"))
	}

	key, err := storageAccountClient.GetStorageAccountKey(s.resourceGroup, s.storageAccount)
	if err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to retrieve storage account"))
	}

	accSecretId, accSecretName, err := s.createUpdateStorageAccountSecret(key)
	if err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to create or update access-secret"))
	}
	logger.WithField("acc-secret-name", accSecretName).Info("secret created/updated")

	bucket.Status = providers.BucketCreated
	bucket.AccessSecretRef = accSecretId
	bucket.StatusMsg = "bucket successfully created"
	if err := s.db.Save(bucket).Error; err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to save bucket"))
	}
	logger.Info("bucket created")

	return nil
}

func (s *ObjectStore) createUpdateStorageAccountSecret(accesskey string) (string, string, error) {

	var secretId string
	storageAccountName := alfanumericRegexp.ReplaceAllString(s.storageAccount, "-")
	secretName := fmt.Sprintf("%v-key", storageAccountName)

	secretRequest := secret.CreateSecretRequest{
		Name: secretName,
		Type: "azureStorageAccount",
		Values: map[string]string{
			"storageAccount": s.storageAccount,
			"accessKey":      accesskey,
		},
		Tags: []string{
			fmt.Sprintf("azureStorageAccount:%v", s.storageAccount),
		},
	}
	secretId, err := secret.Store.CreateOrUpdate(s.org.ID, &secretRequest)
	if err != nil {
		return secretId, secretName, emperror.WrapWith(err, "failed to create/update secret", "secret", secretName)
	}
	return secretId, secretName, nil
}

func (s *ObjectStore) createFailed(bucket *ObjectStoreBucketModel, err error) error {
	bucket.Status = providers.BucketCreateError
	bucket.StatusMsg = err.Error()

	if e := s.db.Save(bucket).Error; e != nil {
		return emperror.WrapWith(e, "failed to save bucket", "bucket", bucket.Name)
	}

	return emperror.With(err, "create failed")
}

func (s *ObjectStore) deleteFailed(bucket *ObjectStoreBucketModel, reason error) error {
	bucket.Status = providers.BucketDeleteError
	bucket.StatusMsg = reason.Error()
	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to save bucket", "bucket", bucket.Name)
	}

	return reason
}

// DeleteBucket deletes the Azure storage container identified by the specified name
// under the current resource group, storage account provided the storage container is of 'managed' type.
func (s *ObjectStore) DeleteBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	logger.Info("looking up the bucket...")

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
		return emperror.WrapWith(err, "failed to lookup", "bucket", bucketName)
	}

	if err := s.deleteFromProvider(bucket); err != nil {
		if !s.force {
			// if delete is not forced return here
			return s.deleteFailed(bucket, err)
		}
	}

	if err := s.db.Delete(bucket).Error; err != nil {
		return s.deleteFailed(bucket, err)
	}

	return nil
}

func (s *ObjectStore) deleteFromProvider(bucket *ObjectStoreBucketModel) error {
	logger := s.getLogger().WithField("bucket", bucket.Name)
	logger.Info("deleting bucket on provider")

	// todo the assumption here is, that a bucket in 'ERROR_CREATE' doesn't exist on the provider
	// todo however there might be -presumably rare cases- when a bucket in 'ERROR_DELETE' that has already been deleted on the provider
	if bucket.Status == providers.BucketCreateError {
		logger.Debug("bucket doesn't exist on provider")
		return nil
	}

	bucket.Status = providers.BucketDeleting
	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to update bucket", "bucket", bucket.Name)
	}

	if err := s.objectStore.DeleteBucket(bucket.Name); err != nil {
		return emperror.WrapWith(err, "failed to delete bucket from provider", "bucket", bucket.Name)
	}

	return nil
}

// CheckBucket checks the status of the given Azure blob.
func (s *ObjectStore) CheckBucket(bucketName string) error {
	logger := s.getLogger().WithField("bucket", bucketName)
	logger.Info("looking up the bucket...")

	if err := s.objectStore.CheckBucket(bucketName); err != nil {
		return emperror.WrapWith(err, "failed to check the bucket", "bucket", bucketName)
	}

	return nil
}

// createStorageAccountAndResourceGroup create storage account and resource group
func (s *ObjectStore) createStorageAccountAndResourceGroup() error {
	resourceGroupClient, err := azureObjectstore.NewAuthorizedResourceGroupClientFromSecret(getCredentials(s.secret))
	if err != nil {
		return emperror.Wrap(err, "failed to create resource group client")
	}
	if err := resourceGroupClient.CreateResourceGroup(s.resourceGroup, s.location, s.logger); err != nil {
		return emperror.Wrap(err, "failed to create resource group")
	}

	storageAccountClient, err := azureObjectstore.NewAuthorizedStorageAccountClientFromSecret(getCredentials(s.secret))
	if err != nil {
		return emperror.With(err, "failed to create storage account client")
	}

	exists, err := storageAccountClient.CheckStorageAccountExistence(s.resourceGroup, s.storageAccount, s.logger)
	if err != nil {
		return emperror.Wrap(err, "failed to check storage account")
	}

	if !*exists {
		if err = storageAccountClient.CreateStorageAccount(s.resourceGroup, s.storageAccount, s.location, s.logger); err != nil {
			return emperror.Wrap(err, "failed to create storage account")
		}
	}

	return nil
}

// ListBuckets returns a list of Azure storage containers buckets that can be accessed with the credentials
// referenced by the secret field. Azure storage containers buckets that were created by a user in the current
// org are marked as 'managed'.
func (s *ObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"organization":    s.org.ID,
		"subscription_id": s.secret.GetValue(pkgSecret.AzureSubscriptionID),
	})

	logger.Info("getting all resource groups for subscription")

	resourceGroupClient, err := azureObjectstore.NewAuthorizedResourceGroupClientFromSecret(getCredentials(s.secret))
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create resource group client")
	}

	resourceGroups, err := resourceGroupClient.GetAllResourceGroups()
	if err != nil {
		return nil, errors.Wrap(err, "getting all resource groups failed")
	}

	buckets := make([]*objectstore.BucketInfo, 0)

	for _, rg := range resourceGroups {
		logger.WithField("resource_group", *(rg.Name)).Info("getting all storage accounts under resource group")

		storageAccountClient, err := azureObjectstore.NewAuthorizedStorageAccountClientFromSecret(getCredentials(s.secret))
		if err != nil {
			return nil, emperror.Wrap(err, "failed to create storage account client")
		}

		storageAccounts, err := storageAccountClient.GetAllStorageAccounts(*rg.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "getting all storage accounts under resource group=%s failed", *(rg.Name))
		}

		// get all Blob containers under the storage account
		for i := 0; i < len(*storageAccounts); i++ {
			accountName := *(*storageAccounts)[i].Name

			logger.WithFields(logrus.Fields{
				"resource_group":  *(rg.Name),
				"storage_account": accountName,
			}).Info("getting all blob containers under storage account")

			objectStore, err := getProviderObjectStore(s.secret, *(rg.Name), accountName, s.location, s.logger)
			if err != nil {
				return nil, emperror.Wrap(err, "failed to create object store")
			}

			blobContainers, err := objectStore.ListBuckets()
			if err != nil {
				return nil, emperror.Wrap(err, "failed to retrieve buckets")
			}

			for i := 0; i < len(blobContainers); i++ {
				blobContainer := blobContainers[i]

				bucketInfo := &objectstore.BucketInfo{
					Name:    blobContainer,
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
		return nil, errors.Wrap(err, "retrieving managed buckets failed")
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

func (s *ObjectStore) ListManagedBuckets() ([]*objectstore.BucketInfo, error) {
	logger := s.getLogger()
	logger.Debug("retrieving managed bucket list")

	var azureBuckets []ObjectStoreBucketModel

	if err := s.db.Where(ObjectStoreBucketModel{OrganizationID: s.org.ID}).
		Order("resource_group asc, storage_account asc, name asc").Find(&azureBuckets).Error; err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve managed buckets")
	}

	bucketList := make([]*objectstore.BucketInfo, 0)
	for _, bucket := range azureBuckets {
		bucketList = append(bucketList, &objectstore.BucketInfo{
			Name:            bucket.Name,
			Managed:         true,
			Location:        bucket.Location,
			SecretRef:       bucket.SecretRef,
			AccessSecretRef: bucket.AccessSecretRef,
			Cloud:           providers.Azure,
			Status:          bucket.Status,
			StatusMsg:       bucket.StatusMsg,
			Azure: &objectstore.BlobStoragePropsForAzure{
				ResourceGroup:  bucket.ResourceGroup,
				StorageAccount: bucket.StorageAccount,
			},
		})
	}

	return bucketList, nil
}

// searchCriteria returns the database search criteria to find a bucket with the given name
// within the scope of the specified resource group and storage account.
func (s *ObjectStore) searchCriteria(bucketName string) *ObjectStoreBucketModel {
	return &ObjectStoreBucketModel{
		OrganizationID: s.org.ID,
		Name:           bucketName,
		ResourceGroup:  s.resourceGroup,
		StorageAccount: s.storageAccount,
		//Location:       s.location,
	}
}
