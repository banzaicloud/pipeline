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
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	pipelineAuth "github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/objectstore"
	commonObjectstore "github.com/banzaicloud/pipeline/pkg/objectstore"
	"github.com/banzaicloud/pipeline/pkg/providers"
	azureObjectstore "github.com/banzaicloud/pipeline/pkg/providers/azure/objectstore"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Default storage account name when none is provided.
// This must between 3-23 letters and can only contain small letters and numbers.
const defaultStorageAccountName = "pipelinegenstorageacc"

var (
	alfanumericRegexp = regexp.MustCompile(`[^a-zA-Z0-9]`)
	falseVal          = false
	trueVal           = true
)

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
	ostore, err := getProviderObjectStore(secret, resourceGroup, storageAccount, location)
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

func getProviderObjectStore(secret *secret.SecretItemResponse, resourceGroup, storageAccount, location string) (azureObjectStore, error) {
	// when no secrets provided build an object store with no provider client/session setup
	// eg. usage: list managed buckets
	if secret == nil {
		return azureObjectstore.NewPlainObjectStore()
	}

	credentials := azureObjectstore.Credentials{
		SubscriptionID: secret.Values[pkgSecret.AzureSubscriptionId],
		ClientID:       secret.Values[pkgSecret.AzureClientId],
		ClientSecret:   secret.Values[pkgSecret.AzureClientSecret],
		TenantID:       secret.Values[pkgSecret.AzureTenantId],
	}

	config := azureObjectstore.Config{
		ResourceGroup:  resourceGroup,
		StorageAccount: storageAccount,
		Location:       location,
	}

	ostore, err := azureObjectstore.New(config, credentials)
	if err != nil {
		return nil, err
	}

	return ostore, nil
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
		"resource_group":  s.getResourceGroup(),
		"storage_account": s.getStorageAccount(),
	})
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
	bucket.ResourceGroup = s.getResourceGroup()
	bucket.StorageAccount = s.getStorageAccount()
	bucket.Organization = *s.org
	bucket.SecretRef = s.secret.ID
	bucket.Status = providers.BucketCreating

	logger.Info("creating bucket...")

	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to save bucket", "bucket", bucketName)
	}

	if err := s.createResourceGroup(s.getResourceGroup()); err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to create resource group"))
	}

	// update here so the bucket list does not get inconsistent
	updateField := &ObjectStoreBucketModel{StorageAccount: s.storageAccount}
	if err := s.db.Model(bucket).Update(updateField).Error; err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "could not update bucket with storage account"))
	}

	exists, err := s.checkStorageAccountExistence(s.getResourceGroup(), s.getStorageAccount())
	if err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to check storage account"))
	}

	if !*exists {
		if err = s.createStorageAccount(s.getResourceGroup(), s.getStorageAccount()); err != nil {
			return s.createFailed(bucket, emperror.Wrap(err, "failed to create storage account"))
		}
	}

	key, err := GetStorageAccountKey(s.getResourceGroup(), s.getStorageAccount(), s.secret, s.logger)
	if err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to retrieve storage account"))
	}

	// update here so the bucket list does not get inconsistent
	updateField = &ObjectStoreBucketModel{Name: bucketName, Location: s.location}
	if err = s.db.Model(bucket).Update(updateField).Error; err != nil {
		return s.createFailed(bucket, emperror.WrapWith(err, "failed to update bucket", "fields", []string{"Name", "Location"}))
	}

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(s.getStorageAccount(), key), azblob.PipelineOptions{})
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", s.getStorageAccount(), bucketName))
	containerURL := azblob.NewContainerURL(*URL, p)

	_, err = containerURL.GetPropertiesAndMetadata(context.TODO(), azblob.LeaseAccessConditions{})
	if err != nil && err.(azblob.StorageError).ServiceCode() == azblob.ServiceCodeContainerNotFound {
		if _, err = containerURL.Create(context.TODO(), azblob.Metadata{}, azblob.PublicAccessNone); err != nil {
			return s.createFailed(bucket, emperror.Wrap(err, "failed to access bucket"))
		}
	}

	accSecretId, accSecretName, err := s.createUpdateStorageAccountSecret(key)
	if err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to create or update access-secret"))
	}
	logger.WithField("acc-secret-name", accSecretName).Info("secret created/updated")

	bucket.Status = providers.BucketCreated
	bucket.AccessSecretRef = accSecretId
	err = s.db.Save(bucket).Error
	if err != nil {
		return s.createFailed(bucket, emperror.Wrap(err, "failed to save bucket"))
	}

	return nil
}

func (s *ObjectStore) createUpdateStorageAccountSecret(accesskey string) (string, string, error) {

	var secretId string
	storageAccountName := alfanumericRegexp.ReplaceAllString(s.getStorageAccount(), "-")
	secretName := fmt.Sprintf("%v-key", storageAccountName)

	secretRequest := secret.CreateSecretRequest{
		Name: secretName,
		Type: "azureStorageAccount",
		Values: map[string]string{
			"storageAccount": s.getStorageAccount(),
			"accessKey":      accesskey,
		},
		Tags: []string{
			fmt.Sprintf("azureStorageAccount:%v", s.getStorageAccount()),
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
		return emperror.WrapWith(e, "failed to save bucket", "gorm", "db")
	}

	return emperror.With(err, "create failed")
}

func (s *ObjectStore) deleteFailed(bucket *ObjectStoreBucketModel, reason error) error {

	bucket.Status = providers.BucketDeleteError
	bucket.StatusMsg = reason.Error()

	if err := s.db.Save(bucket).Error; err != nil {
		return emperror.WrapWith(err, "failed to save bucket", "gorm", "db")
	}

	return reason
}

func (s *ObjectStore) createResourceGroup(resourceGroup string) error {
	logger := s.getLogger()
	gclient := resources.NewGroupsClient(s.secret.Values[pkgSecret.AzureSubscriptionId])

	logger.Info("creating resource group")

	authorizer, err := newAuthorizer(s.secret)
	if err != nil {
		return fmt.Errorf("authentication failed: %s", err.Error())
	}

	gclient.Authorizer = authorizer
	res, _ := gclient.Get(context.TODO(), resourceGroup)

	if res.StatusCode == http.StatusNotFound {
		result, err := gclient.CreateOrUpdate(
			context.TODO(),
			resourceGroup,
			resources.Group{Location: to.StringPtr(s.location)},
		)
		if err != nil {
			return err
		}

		logger.Info(result.Status)
	}

	logger.Info("resource group created")

	return nil
}

func (s *ObjectStore) checkStorageAccountExistence(resourceGroup string, storageAccount string) (*bool, error) {
	storageAccountsClient, err := createStorageAccountClient(s.secret)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create storage account client")
	}

	logger := s.getLogger()

	logger.Info("retrieving storage account name availability...")
	result, err := storageAccountsClient.CheckNameAvailability(
		context.TODO(),
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(storageAccount),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
		},
	)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve storage account name availability")
	}

	if *result.NameAvailable == false {
		// account name is already taken or it is invalid
		// retrieve the storage account
		if _, err = storageAccountsClient.GetProperties(context.TODO(), resourceGroup, storageAccount); err != nil {
			logger.Errorf("could not retrieve storage account, %s", *result.Message)
			return nil, emperror.WrapWith(err, *result.Message, "storage_account", storageAccount, "resource_group", resourceGroup)
		}
		// storage name exists, available
		return &trueVal, nil
	}

	// storage name doesn't exist
	return &falseVal, nil
}

func (s *ObjectStore) createStorageAccount(resourceGroup string, storageAccount string) error {
	storageAccountsClient, err := createStorageAccountClient(s.secret)
	if err != nil {
		return err
	}

	logger := s.getLogger()

	logger.Info("creating storage account")

	future, err := storageAccountsClient.Create(
		context.TODO(),
		resourceGroup,
		storageAccount,
		storage.AccountCreateParameters{
			Sku: &storage.Sku{
				Name: storage.StandardLRS,
			},
			Kind:     storage.BlobStorage,
			Location: to.StringPtr(s.location),
			AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{
				AccessTier: storage.Hot,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("cannot create storage account: %v", err)
	}

	logger.Info("storage account creation request sent")
	if future.WaitForCompletion(context.TODO(), storageAccountsClient.Client) != nil {
		return err
	}

	logger.Info("storage account created")

	return nil
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
			return s.deleteFailed(bucket, emperror.WrapWith(err, "failed to delete from provider", "bucket", bucketName))
		}
	}

	if err := s.db.Delete(bucket).Error; err != nil {
		return s.deleteFailed(bucket, emperror.WrapWith(err, "failed to delete", "bucket", bucketName))
	}

	return nil

}

func (s *ObjectStore) deleteFromProvider(bucket *ObjectStoreBucketModel) error {
	logger := s.getLogger().WithField("bucket", bucket.Name)
	logger.Info("deleting bucket on provider")

	// the assumption here is, that a bucket in 'ERROR_CREATE' doesn't exist on the provider
	// however there might be -presumably rare cases- when a bucket in 'ERROR_DELETE' that has already been deleted on the provider
	if bucket.Status == providers.BucketCreateError {
		logger.Debug("bucket doesn't exist on provider")
		return nil
	}

	bucket.Status = providers.BucketDeleting
	db := s.db.Save(bucket)
	if db.Error != nil {
		return s.deleteFailed(bucket, emperror.WrapWith(db.Error, "failed to update", "bucket", bucket.Name))
	}

	key, err := GetStorageAccountKey(s.getResourceGroup(), s.getStorageAccount(), s.secret, s.logger)
	if err != nil {
		return emperror.WrapWith(err, "filed to retrieve storage account key", "bucket", bucket.Name)
	}

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(s.getStorageAccount(), key), azblob.PipelineOptions{})
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", s.getStorageAccount(), bucket.Name))
	containerURL := azblob.NewContainerURL(*URL, p)

	if _, err = containerURL.Delete(context.TODO(), azblob.ContainerAccessConditions{}); err != nil {
		return emperror.WrapWith(err, "failed to delete container", "bucket", bucket.Name)
	}

	return nil
}

// CheckBucket checks the status of the given Azure blob.
func (s *ObjectStore) CheckBucket(bucketName string) error {
	resourceGroup := s.getResourceGroup()
	storageAccount := s.getStorageAccount()

	logger := s.getLogger().WithField("bucket", bucketName)
	logger.Info("looking up the bucket")

	if _, err := s.checkStorageAccountExistence(resourceGroup, storageAccount); err != nil {
		return emperror.WrapWith(err, "failed to check storage account", "bucket", bucketName)
	}

	key, err := GetStorageAccountKey(resourceGroup, storageAccount, s.secret, s.logger)
	if err != nil {
		return emperror.WrapWith(err, "failed to get storage account key", "bucket", bucketName)
	}

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(s.storageAccount, key), azblob.PipelineOptions{})
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", s.storageAccount, bucketName))
	containerURL := azblob.NewContainerURL(*URL, p)

	if _, err = containerURL.GetPropertiesAndMetadata(context.TODO(), azblob.LeaseAccessConditions{}); err != nil {
		return emperror.WrapWith(err, "failed to get container metadata", "bucket", bucketName)
	}

	return nil
}

// ListBuckets returns a list of Azure storage containers buckets that can be accessed with the credentials
// referenced by the secret field. Azure storage containers buckets that were created by a user in the current
// org are marked as 'managed'.
func (s *ObjectStore) ListBuckets() ([]*objectstore.BucketInfo, error) {
	logger := s.getLogger()

	logger.Info("getting all resource groups for subscription")

	resourceGroups, err := getAllResourceGroups(s.secret)
	if err != nil {
		return nil, fmt.Errorf("getting all resource groups failed: %s", err.Error())
	}

	var buckets []*objectstore.BucketInfo

	for _, rg := range resourceGroups {
		logger.Info("getting all storage accounts under resource group")

		storageAccounts, err := getAllStorageAccounts(s.secret, *rg.Name)
		if err != nil {
			return nil, fmt.Errorf("getting all storage accounts under resource group=%s failed: %s", *(rg.Name), err.Error())
		}

		// get all Blob containers under the storage account
		for i := 0; i < len(*storageAccounts); i++ {
			accountName := *(*storageAccounts)[i].Name

			logger.Info("getting all blob containers under storage account")

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

func (s *ObjectStore) ListManagedBuckets() ([]*objectstore.BucketInfo, error) {

	logger := s.getLogger()
	logger.Info("getting all resource groups for subscription")

	var objectStores []ObjectStoreBucketModel
	err := s.db.
		Where(&ObjectStoreBucketModel{OrganizationID: s.org.ID}).
		Order("resource_group asc, storage_account asc, name asc").
		Find(&objectStores).Error

	if err != nil {
		return nil, fmt.Errorf("retrieving managed buckets failed: %s", err.Error())
	}

	bucketList := make([]*objectstore.BucketInfo, 0)
	for _, bucket := range objectStores {
		bucketInfo := &objectstore.BucketInfo{Name: bucket.Name, Managed: true}
		bucketInfo.Location = bucket.Location
		bucketInfo.SecretRef = bucket.SecretRef
		bucketInfo.AccessSecretRef = bucket.AccessSecretRef
		bucketInfo.Cloud = providers.Azure
		bucketInfo.Status = bucket.Status
		bucketInfo.StatusMsg = bucket.StatusMsg
		bucketInfo.Azure = &objectstore.BlobStoragePropsForAzure{
			ResourceGroup:  bucket.ResourceGroup,
			StorageAccount: bucket.StorageAccount,
		}
		bucketList = append(bucketList, bucketInfo)
	}

	return bucketList, nil
}

func GetStorageAccountKey(resourceGroup string, storageAccount string, s *secret.SecretItemResponse, log logrus.FieldLogger) (string, error) {
	client, err := createStorageAccountClient(s)
	if err != nil {
		return "", emperror.WrapWith(err, "failed to create storage accounts client", "storageaccount", storageAccount)
	}

	logger := log.WithFields(logrus.Fields{
		"resource_group":  resourceGroup,
		"storage_account": storageAccount,
	})

	logger.Info("getting key for storage account")

	keys, err := client.ListKeys(context.TODO(), resourceGroup, storageAccount)
	if err != nil {
		return "", emperror.WrapWith(err, "failed to retrieve keys  for storage account", "storageaccount", storageAccount)
	}

	key := (*keys.Keys)[0].Value

	return *key, nil
}

func createStorageAccountClient(s *secret.SecretItemResponse) (*storage.AccountsClient, error) {
	accountClient := storage.NewAccountsClient(s.Values[pkgSecret.AzureSubscriptionId])

	authorizer, err := newAuthorizer(s)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to authenticate")
	}

	accountClient.Authorizer = authorizer

	return &accountClient, nil
}

// getAllResourceGroups returns all resource groups using
// the Azure credentials referenced by the provided secret.
func getAllResourceGroups(s *secret.SecretItemResponse) ([]*resources.Group, error) {
	rgClient := resources.NewGroupsClient(s.GetValue(pkgSecret.AzureSubscriptionId))
	authorizer, err := newAuthorizer(s)
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

		if err = resourceGroupsPages.Next(); err != nil {
			return nil, err
		}
	}

	return groups, nil
}

// getAllStorageAccounts returns all storage accounts under the specified resource group
// using the Azure credentials referenced by the provided secret.
func getAllStorageAccounts(s *secret.SecretItemResponse, resourceGroup string) (*[]storage.Account, error) {
	client, err := createStorageAccountClient(s)
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

func newAuthorizer(s *secret.SecretItemResponse) (autorest.Authorizer, error) {
	authorizer, err := auth.NewClientCredentialsConfig(
		s.Values[pkgSecret.AzureClientId],
		s.Values[pkgSecret.AzureClientSecret],
		s.Values[pkgSecret.AzureTenantId]).Authorizer()

	if err != nil {
		return nil, err
	}

	return authorizer, nil
}

// searchCriteria returns the database search criteria to find a bucket with the given name
// within the scope of the specified resource group and storage account.
func (s *ObjectStore) searchCriteria(bucketName string) *ObjectStoreBucketModel {
	return &ObjectStoreBucketModel{
		OrganizationID: s.org.ID,
		Name:           bucketName,
		ResourceGroup:  s.getResourceGroup(),
		StorageAccount: s.getStorageAccount(),
		//Location:       s.location,
	}
}
