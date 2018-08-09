package azure

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	pipelineAuth "github.com/banzaicloud/pipeline/auth"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	pkgStorage "github.com/banzaicloud/pipeline/pkg/storage"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/jinzhu/gorm"
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
	db     *gorm.DB
	logger logrus.FieldLogger

	org    *pipelineAuth.Organization
	secret *secret.SecretItemResponse

	storageAccount string
	resourceGroup  string
	location       string
}

// NewObjectStore returns a new object store instance.
func NewObjectStore(
	org *pipelineAuth.Organization,
	secret *secret.SecretItemResponse,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *ObjectStore {
	return &ObjectStore{
		db:     db,
		logger: logger,
		org:    org,
		secret: secret,
	}
}

// WithResourceGroup updates the resource group.
func (s *ObjectStore) WithResourceGroup(resourceGroup string) error {
	s.resourceGroup = resourceGroup

	return nil
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

// WithStorageAccount updates the storage account.
func (s *ObjectStore) WithStorageAccount(storageAccount string) error {
	s.storageAccount = storageAccount

	return nil
}

// getStorageAccount returns the given storage account or or falls back to a default one.
func (s *ObjectStore) getStorageAccount() string {
	storageAccount := s.storageAccount

	if storageAccount == "" {
		storageAccount = defaultStorageAccountName
	}

	return storageAccount
}

// WithRegion updates the region.
func (s *ObjectStore) WithRegion(region string) error {
	s.location = region

	return nil
}

func (s *ObjectStore) getLogger(bucketName string) logrus.FieldLogger {
	return s.logger.WithFields(logrus.Fields{
		"organization":    s.org.ID,
		"bucket":          bucketName,
		"resource_group":  s.getResourceGroup(),
		"storage_account": s.getStorageAccount(),
	})
}

// CreateBucket creates an Azure Object Store Blob with the provided name
// within a generated/provided ResourceGroup and StorageAccount
func (s *ObjectStore) CreateBucket(bucketName string) {
	resourceGroup := s.getResourceGroup()
	storageAccount := s.getStorageAccount()

	logger := s.getLogger(bucketName)

	bucket := &ObjectStoreModel{}
	searchCriteria := s.searchCriteria(bucketName)

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			logger.Errorf("error happened during getting bucket from DB: %s", err.Error())

			return
		}
	}

	// TODO: create the bucket in the database later so that we don't have to roll back
	bucket.ResourceGroup = resourceGroup
	bucket.Organization = *s.org

	logger.Info("saving bucket in DB")

	err := s.db.Save(bucket).Error
	if err != nil {
		logger.Errorf("error happened during saving bucket in DB: %s", err.Error())

		return
	}

	err = s.createResourceGroup(resourceGroup)
	if err != nil {
		s.rollback(logger, "resource group creation failed", err, bucket)

		return
	}

	// update here so the bucket list does not get inconsistent
	updateField := &ObjectStoreModel{StorageAccount: s.storageAccount}
	err = s.db.Model(bucket).Update(updateField).Error
	if err != nil {
		logger.Errorf("error happened during updating storage account: %s", err.Error())

		return
	}

	exists, err := s.checkStorageAccountExistence(resourceGroup, storageAccount)
	if !exists && err == nil {
		err = s.createStorageAccount(resourceGroup, storageAccount)
		if err != nil {
			s.rollback(logger, "storage account creation failed", err, bucket)

			return
		}
	}

	if err != nil && !strings.Contains(err.Error(), "is already taken") {
		s.rollback(logger, "storage account is already taken", err, bucket)

		return
	}

	key, err := s.getStorageAccountKey(resourceGroup, storageAccount)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	// update here so the bucket list does not get inconsistent
	updateField = &ObjectStoreModel{Name: bucketName, Location: s.location}
	err = s.db.Model(bucket).Update(updateField).Error
	if err != nil {
		logger.Errorf("error happened during updating bucket name: %s", err.Error())

		return
	}

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(storageAccount, key), azblob.PipelineOptions{})
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", storageAccount, bucketName))
	containerURL := azblob.NewContainerURL(*URL, p)

	_, err = containerURL.GetPropertiesAndMetadata(context.TODO(), azblob.LeaseAccessConditions{})
	if err != nil && err.(azblob.StorageError).ServiceCode() == azblob.ServiceCodeContainerNotFound {
		_, err = containerURL.Create(context.TODO(), azblob.Metadata{}, azblob.PublicAccessNone)
		if err != nil {
			s.rollback(logger, "cannot access bucket", err, bucket)

			return
		}
	}

	return
}

func (s *ObjectStore) rollback(logger logrus.FieldLogger, msg string, err error, bucket *ObjectStoreModel) {
	logger.Errorf("%s (rolling back): %s", msg, err.Error())

	err = s.db.Delete(bucket).Error
	if err != nil {
		logger.Error(err.Error())
	}
}

func (s *ObjectStore) createResourceGroup(resourceGroup string) error {
	logger := s.logger.WithField("resource_group", resourceGroup)
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

func (s *ObjectStore) checkStorageAccountExistence(resourceGroup string, storageAccount string) (bool, error) {
	storageAccountsClient, err := createStorageAccountClient(s.secret)
	if err != nil {
		return false, err
	}

	logger := s.logger.WithFields(logrus.Fields{
		"resource_group":  resourceGroup,
		"storage_account": storageAccount,
	})

	logger.Info("checking storage account existence")

	result, err := storageAccountsClient.CheckNameAvailability(
		context.TODO(),
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(storageAccount),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
		},
	)
	if err != nil {
		return false, err
	}

	if *result.NameAvailable == false {
		_, err := storageAccountsClient.GetProperties(context.TODO(), resourceGroup, storageAccount)
		if err != nil {
			logger.Error("storage account exists but it is not in your resource group")

			return false, err
		}

		logger.Warnf("storage account name not available because: %s", *result.Message)

		return true, fmt.Errorf("storage account name %s is already taken", storageAccount)
	}

	return false, nil
}

func (s *ObjectStore) createStorageAccount(resourceGroup string, storageAccount string) error {
	storageAccountsClient, err := createStorageAccountClient(s.secret)
	if err != nil {
		return err
	}

	logger := s.logger.WithFields(logrus.Fields{
		"resource_group":  resourceGroup,
		"storage_account": storageAccount,
	})

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
	resourceGroup := s.getResourceGroup()
	storageAccount := s.getStorageAccount()

	logger := s.getLogger(bucketName)

	bucket := &ObjectStoreModel{}
	searchCriteria := s.searchCriteria(bucketName)

	logger.Info("looking for bucket")

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
	}

	key, err := s.getStorageAccountKey(resourceGroup, storageAccount)
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

	bucket := &ObjectStoreModel{}
	searchCriteria := s.searchCriteria(bucketName)

	logger.Info("looking for bucket")

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return bucketNotFoundError{}
		}
	}

	_, err := s.checkStorageAccountExistence(resourceGroup, storageAccount)
	if err != nil && !strings.Contains(err.Error(), "is already taken") {
		logger.Error(err.Error())

		return err
	}

	key, err := s.getStorageAccountKey(resourceGroup, storageAccount)
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
func (s *ObjectStore) ListBuckets() ([]*pkgStorage.BucketInfo, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"organization":    s.org.ID,
		"subscription_id": s.secret.GetValue(pkgSecret.AzureSubscriptionId),
	})

	logger.Info("getting all resource groups for subscription")

	resourceGroups, err := getAllResourceGroups(s.secret)
	if err != nil {
		return nil, fmt.Errorf("getting all resource groups failed: %s", err.Error())
	}

	var buckets []*pkgStorage.BucketInfo

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

			accountKey, err := s.getStorageAccountKey(*rg.Name, accountName)
			if err != nil {
				return nil, fmt.Errorf("getting all storage accounts under resource group=%s, storage account=%s failed: %s", *(rg.Name), accountName, err.Error())
			}

			blobContainers, err := getAllBlobContainers(accountName, accountKey)
			if err != nil {
				return nil, fmt.Errorf("getting all storage accounts under resource group=%s, storage account=%s failed: %s", *(rg.Name), accountName, err.Error())
			}

			for i := 0; i < len(blobContainers); i++ {
				blobContainer := blobContainers[i]

				bucketInfo := &pkgStorage.BucketInfo{
					Name:    blobContainer.Name,
					Managed: false,
					Azure: &pkgStorage.BlobStoragePropsForAzure{
						StorageAccount: accountName,
						ResourceGroup:  *rg.Name,
					},
				}

				buckets = append(buckets, bucketInfo)
			}

		}
	}

	var objectStores []ObjectStoreModel

	err = s.db.Where(&ObjectStoreModel{OrganizationID: s.org.ID}).Order("resource_group asc, storage_account asc, name asc").Find(&objectStores).Error
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

func (s *ObjectStore) getStorageAccountKey(resourceGroup string, storageAccount string) (string, error) {
	client, err := createStorageAccountClient(s.secret)
	if err != nil {
		return "", err
	}

	logger := s.logger.WithFields(logrus.Fields{
		"resource_group":  resourceGroup,
		"storage_account": storageAccount,
	})

	logger.Info("getting key for storage account")

	keys, err := client.ListKeys(context.TODO(), resourceGroup, storageAccount)
	if err != nil {
		return "", fmt.Errorf("error retrieving keys for StorageAccount %s", err.Error())
	}

	key := (*keys.Keys)[0].Value

	return *key, nil
}

func createStorageAccountClient(s *secret.SecretItemResponse) (*storage.AccountsClient, error) {
	accountClient := storage.NewAccountsClient(s.Values[pkgSecret.AzureSubscriptionId])

	authorizer, err := newAuthorizer(s)
	if err != nil {
		return nil, fmt.Errorf("error happened during authentication: %s", err.Error())
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
func (s *ObjectStore) searchCriteria(bucketName string) *ObjectStoreModel {
	return &ObjectStoreModel{
		OrganizationID: s.org.ID,
		Name:           bucketName,
		ResourceGroup:  s.getResourceGroup(),
		StorageAccount: s.getStorageAccount(),
	}
}
