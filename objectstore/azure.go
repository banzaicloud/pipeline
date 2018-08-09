package objectstore

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
	pkgAzure "github.com/banzaicloud/pipeline/pkg/cluster/aks"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	pkgStorage "github.com/banzaicloud/pipeline/pkg/storage"
	"github.com/banzaicloud/pipeline/secret"
)

// ManagedAzureBlobStore is the schema for the DB
type ManagedAzureBlobStore struct {
	ID             uint                      `gorm:"primary_key"`
	Organization   pipelineAuth.Organization `gorm:"foreignkey:OrgID"`
	OrgID          uint                      `gorm:"index;not null"`
	Name           string                    `gorm:"unique_index:bucketName"`
	ResourceGroup  string                    `gorm:"unique_index:bucketName"`
	StorageAccount string                    `gorm:"unique_index:bucketName"`
	Location       string
}

// AzureObjectStore stores all required parameters for container creation
type AzureObjectStore struct {
	storageAccount string
	secret         *secret.SecretItemResponse
	resourceGroup  string
	location       string
	org            *pipelineAuth.Organization
}

// This must beetween 3-23 letters and can only contain small letters and numbers
const storageAccountName = "pipelinegenstorageacc"

// WithResourceGroup updates the resource group.
func (b *AzureObjectStore) WithResourceGroup(resourceGroup string) error {
	b.resourceGroup = resourceGroup
	return nil
}

// WithStorageAccount updates the storage account.
func (b *AzureObjectStore) WithStorageAccount(storageAccount string) error {
	b.storageAccount = storageAccount
	return nil
}

// WithRegion updates the region.
func (b *AzureObjectStore) WithRegion(region string) error {
	b.location = region
	return nil
}

// CreateBucket creates an Azure Object Store Blob with the provided name
// within a generated/provided ResourceGroup and StorageAccount
func (b *AzureObjectStore) CreateBucket(bucketName string) {

	if b.resourceGroup == "" {
		b.resourceGroup = generateResourceGroupName(b.location)
	}
	if b.storageAccount == "" {
		b.storageAccount = storageAccountName
	}

	managedBucket := &ManagedAzureBlobStore{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)

	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		switch err.(type) {
		case ManagedBucketNotFoundError:
		default:
			log.Errorf("Error happened during getting bucket description from DB %s", err.Error())
			return
		}
	}

	managedBucket.ResourceGroup = b.resourceGroup
	managedBucket.Organization = *b.org

	if err := persistToDb(managedBucket); err != nil {
		log.Errorf("Error happened during persisting bucket description to DB %s", err.Error())
		return
	}

	log.Infof("Creating resource group %s", b.resourceGroup)

	err := createResourceGroup(b)
	if err != nil {
		log.Error(err.Error())
		if e := deleteFromDbByPK(managedBucket); e != nil {
			log.Error(e.Error())
		}
		return
	}
	log.Infof("Resource group %s created", b.resourceGroup)
	updateField := &ManagedAzureBlobStore{StorageAccount: b.storageAccount}
	if err := updateDBField(managedBucket, updateField); err != nil {
		log.Errorf("Error happened during persisting bucket description to DB %s", err.Error())
		return
	}
	exists, err := checkStorageAccountExistence(b)
	if !exists && err == nil {
		err = createStorageAccount(b)
		if err != nil {
			log.Error(err.Error())
			if e := deleteFromDbByPK(managedBucket); e != nil {
				log.Error(e.Error())
			}
			return
		}
	}
	if err != nil && !strings.Contains(err.Error(), "is already taken") {
		log.Error(err.Error())
		if e := deleteFromDbByPK(managedBucket); e != nil {
			log.Error(e.Error())
		}
		return
	}
	key, err := getStorageAccountKey(b.secret, b.resourceGroup, b.storageAccount)
	if err != nil {
		log.Error(err.Error())
		return
	}
	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(b.storageAccount, key), azblob.PipelineOptions{})
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", b.storageAccount, bucketName))
	containerURL := azblob.NewContainerURL(*URL, p)
	updateField = &ManagedAzureBlobStore{Name: bucketName, Location: b.location}
	if err := updateDBField(managedBucket, updateField); err != nil {
		log.Errorf("Error happened during persisting bucket description to DB %s", err.Error())
		return
	}
	_, err = containerURL.GetPropertiesAndMetadata(context.Background(), azblob.LeaseAccessConditions{})
	if err != nil && err.(azblob.StorageError).ServiceCode() == azblob.ServiceCodeContainerNotFound {
		_, err = containerURL.Create(context.Background(), azblob.Metadata{}, azblob.PublicAccessNone)
		if err != nil {
			log.Error(err.Error())
			if e := deleteFromDbByPK(managedBucket); e != nil {
				log.Error(e.Error())
			}
			return
		}
	}
	return
}

// DeleteBucket deletes the Azure storage container identified by the specified name
// under the current resource group, storage account provided the storage container is of 'managed` type
func (b *AzureObjectStore) DeleteBucket(bucketName string) error {

	managedBucket := &ManagedAzureBlobStore{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)

	log.Info("Looking up managed bucket: resource group=%s, storage account=%s, name=%s", b.resourceGroup, b.storageAccount, bucketName)

	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		return err
	}

	key, err := getStorageAccountKey(b.secret, b.resourceGroup, b.storageAccount)
	if err != nil {
		return err
	}

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(b.storageAccount, key), azblob.PipelineOptions{})
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", b.storageAccount, bucketName))
	containerURL := azblob.NewContainerURL(*URL, p)

	_, err = containerURL.Delete(context.Background(), azblob.ContainerAccessConditions{})

	if err != nil {
		return err
	}

	if err = deleteFromDbByPK(managedBucket); err != nil {
		log.Errorf("Deleting managed Azure bucket from database failed: %s", err.Error())
		return err
	}

	return nil
}

//CheckBucket check the status of the given Azure blob
func (b *AzureObjectStore) CheckBucket(bucketName string) error {
	managedBucket := &ManagedAzureBlobStore{}
	searchCriteria := b.newManagedBucketSearchCriteria(bucketName)
	log.Info("Looking up managed bucket: name=%s", bucketName)
	if err := getManagedBucket(searchCriteria, managedBucket); err != nil {
		return ManagedBucketNotFoundError{}
	}

	_, err := checkStorageAccountExistence(b)
	if err != nil && !strings.Contains(err.Error(), "is already taken") {
		log.Error(err.Error())
		return err
	}

	key, err := getStorageAccountKey(b.secret, b.resourceGroup, b.storageAccount)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(b.storageAccount, key), azblob.PipelineOptions{})
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", b.storageAccount, bucketName))
	containerURL := azblob.NewContainerURL(*URL, p)

	_, err = containerURL.GetPropertiesAndMetadata(context.TODO(), azblob.LeaseAccessConditions{})
	if err != nil {
		return err
	}
	return nil
}

// ListBuckets returns a list of Azure storage containers buckets that can be accessed with the credentials
// referenced by the secret field. Azure storage containers buckets that were created by a user in the current
// org are marked as 'managed`
func (b *AzureObjectStore) ListBuckets() ([]*pkgStorage.BucketInfo, error) {

	// get all resource groups
	log.Infof("Getting all resource groups for subscription id=%s", b.secret.GetValue(pkgSecret.AzureSubscriptionId))
	resourceGroups, err := getAllResourceGroups(b.secret)
	if err != nil {
		log.Errorf("Getting all resource groups for subscription id=%s failed: %s", b.secret.GetValue(pkgSecret.AzureSubscriptionId), err.Error())
		return nil, err
	}

	var buckets []*pkgStorage.BucketInfo
	// get all storage accounts
	for _, rg := range resourceGroups {
		log.Infof("Getting all storage accounts under resource group=%s", *(rg.Name))

		storageAccounts, err := getAllStorageAccounts(b.secret, *rg.Name)
		if err != nil {
			log.Errorf("Getting all storage accounts under resource group=%s failed: %s", *(rg.Name), err.Error())
			return nil, err
		}

		// get all Blob containers under the storage account
		for i := 0; i < len(*storageAccounts); i++ {
			accountName := *(*storageAccounts)[i].Name

			log.Infof("Getting all Blob containers under resource group=%s, storage account=%s", *(rg.Name), accountName)

			accountKey, err := getStorageAccountKey(b.secret, *rg.Name, accountName)
			if err != nil {
				log.Infof("Getting all Blob containers under resource group=%s, storage account=%s failed: %s", *(rg.Name), accountName, err.Error())
				return nil, err
			}

			blobContainers, err := getAllBlobContainers(accountName, accountKey)
			if err != nil {
				log.Infof("Getting all Blob containers under resource group=%s, storage account=%s failed: %s", *(rg.Name), accountName, err.Error())
				return nil, err
			}

			for i := 0; i < len(blobContainers); i++ {
				blobContainer := blobContainers[i]

				bucketInfo := &pkgStorage.BucketInfo{
					Name:    blobContainer.Name,
					Managed: false,
					Azure: &pkgAzure.BlobStoragePropsForAzure{
						StorageAccount: accountName,
						ResourceGroup:  *rg.Name,
					},
				}

				buckets = append(buckets, bucketInfo)
			}

		}
	}

	var managedAzureBlobStores []ManagedAzureBlobStore
	if err = queryWithOrderByDb(&ManagedAmazonBucket{OrgID: b.org.ID}, "resource_group asc, storage_account asc, name asc", &managedAzureBlobStores); err != nil {
		log.Errorf("Retrieving managed buckets in organisation id=%s failed: %s", err.Error())
		return nil, err
	}

	for _, bucketInfo := range buckets {
		// managedAzureBlobStores must be sorted in order to be able to perform binary search on it
		idx := sort.Search(len(managedAzureBlobStores), func(i int) bool {
			return strings.Compare(managedAzureBlobStores[i].ResourceGroup, bucketInfo.Azure.ResourceGroup) >= 0 &&
				strings.Compare(managedAzureBlobStores[i].StorageAccount, bucketInfo.Azure.StorageAccount) >= 0 &&
				strings.Compare(managedAzureBlobStores[i].Name, bucketInfo.Name) >= 0
		})

		if idx < len(managedAzureBlobStores) &&
			strings.Compare(managedAzureBlobStores[idx].ResourceGroup, bucketInfo.Azure.ResourceGroup) >= 0 &&
			strings.Compare(managedAzureBlobStores[idx].StorageAccount, bucketInfo.Azure.StorageAccount) >= 0 &&
			strings.Compare(managedAzureBlobStores[idx].Name, bucketInfo.Name) >= 0 {
			bucketInfo.Managed = true
		}

	}

	return buckets, nil
}

func getStorageAccountKey(s *secret.SecretItemResponse, resourceGroup, storageAccount string) (string, error) {
	client, err := createStorageAccountClient(s)
	if err != nil {
		return "", err
	}
	log.Infof("Getting key for storage account %s in resource group %s", storageAccount, resourceGroup)
	keys, err := client.ListKeys(context.TODO(), resourceGroup, storageAccount)
	if err != nil {
		log.Errorf("Error retrieving keys for StorageAccount %s", err.Error())
		return "", err
	}
	key := (*keys.Keys)[0].Value
	return *key, nil
}

func createStorageAccountClient(s *secret.SecretItemResponse) (*storage.AccountsClient, error) {
	accountClient := storage.NewAccountsClient(s.Values[pkgSecret.AzureSubscriptionId])

	authorizer, err := newAuthorizer(s)
	if err != nil {
		log.Errorf("Error happened during authentication %s", err.Error())
		return nil, err
	}
	accountClient.Authorizer = authorizer
	log.Info("Authenticating succeeded")
	return &accountClient, nil
}

func checkStorageAccountExistence(b *AzureObjectStore) (bool, error) {
	storageAccountsClient, err := createStorageAccountClient(b.secret)
	if err != nil {
		return false, err
	}
	log.Info("Checking StorageAccount existence")
	result, err := storageAccountsClient.CheckNameAvailability(
		context.TODO(),
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(b.storageAccount),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
		})
	if err != nil {
		log.Error(err)
		return false, err
	}
	if *result.NameAvailable == false {
		_, err := storageAccountsClient.GetProperties(context.TODO(), b.resourceGroup, b.storageAccount)
		if err != nil {
			log.Error("StorageAccount exists but not in your ResourceGroup")
			return false, err
		}
		log.Warnf("[%s] storage account name not available because %s", b.storageAccount, *result.Message)
		return true, fmt.Errorf("storage account name %s is already taken", b.storageAccount)
	}
	return false, nil
}

func createStorageAccount(b *AzureObjectStore) error {
	storageAccountsClient, err := createStorageAccountClient(b.secret)
	if err != nil {
		return err
	}
	log.Info("StorageAccount can be created")

	future, err := storageAccountsClient.Create(
		context.TODO(),
		b.resourceGroup,
		b.storageAccount,
		storage.AccountCreateParameters{
			Sku: &storage.Sku{
				Name: storage.StandardLRS},
			Kind:     storage.BlobStorage,
			Location: to.StringPtr(b.location),
			AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{
				AccessTier: storage.Hot,
			},
		})

	if err != nil {
		return fmt.Errorf("cannot create storage account: %v", err)
	}
	log.Info("StorageAccount creation request sent")
	if future.WaitForCompletion(context.TODO(), storageAccountsClient.Client) != nil {
		log.Error("Could not create a storage account.")
		return err
	}
	log.Info("StorageAccount created")
	return nil
}

func createResourceGroup(b *AzureObjectStore) error {
	gclient := resources.NewGroupsClient(b.secret.Values[pkgSecret.AzureSubscriptionId])

	authorizer, err := newAuthorizer(b.secret)
	if err != nil {
		log.Errorf("Authentication failed: %s", err.Error())
		return err
	}
	gclient.Authorizer = authorizer
	res, _ := gclient.Get(context.TODO(), b.resourceGroup)
	if res.StatusCode == http.StatusNotFound {
		result, err := gclient.CreateOrUpdate(context.TODO(), b.resourceGroup,
			resources.Group{Location: to.StringPtr(b.location)})
		if err != nil {
			log.Error(err)
			return err
		}
		log.Info(result.Status)
	}
	return nil
}

// getAllResourceGroups returns all resource groups using
// the Azure credentials referenced by the provided secret
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
// using the Azure credentials referenced by the provided secret
func getAllStorageAccounts(s *secret.SecretItemResponse, resourceGroupName string) (*[]storage.Account, error) {
	client, err := createStorageAccountClient(s)
	if err != nil {
		return nil, err
	}

	storageAccountList, err := client.ListByResourceGroup(context.TODO(), resourceGroupName)
	if err != nil {
		return nil, err
	}

	return storageAccountList.Value, nil
}

// getAllBlobContainers returns all blob container that belong to the specified storage account using
// the given storage account key
func getAllBlobContainers(storageAccountName, storageAccountKey string) ([]azblob.Container, error) {
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", storageAccountName))

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(storageAccountName, storageAccountKey), azblob.PipelineOptions{})
	serviceURL := azblob.NewServiceURL(*u, p)

	resp, err := serviceURL.ListContainers(context.Background(), azblob.Marker{}, azblob.ListContainersOptions{})
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

// newManagedBucketSearchCriteria returns the database search criteria to find managed bucket with the given name
// within the scope of the specified resource group and storage account
func (b *AzureObjectStore) newManagedBucketSearchCriteria(bucketName string) *ManagedAzureBlobStore {
	return &ManagedAzureBlobStore{
		OrgID:          b.org.ID,
		Name:           bucketName,
		ResourceGroup:  b.resourceGroup,
		StorageAccount: b.storageAccount,
	}
}

func generateResourceGroupName(location string) string {
	return fmt.Sprintf("pipeline-auto-%s", location)
}
