package objectstore

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
	"net/url"
	"strings"
)

type AzureObjectStore struct {
	storageAccount string
	secret         *secret.SecretsItemResponse
	resourceGroup  string
	location       string
}

func (b *AzureObjectStore) CreateBucket(bucketName string) error {
	exists, err := checkStorageAccountExistence(b)
	if !exists && err == nil {
		err = createStorageAccount(b)
		if err != nil {
			return err
		}
	}
	if err != nil && !strings.Contains(err.Error(), "is already taken") {
		return err
	}
	key, err := getStorageAccountKey(b)
	if err != nil {
		return err
	}
	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(b.storageAccount, key), azblob.PipelineOptions{})
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", b.storageAccount, bucketName))
	containerURL := azblob.NewContainerURL(*URL, p)
	_, err = containerURL.Create(context.Background(), azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		return err
	}
	return nil
}

func (b *AzureObjectStore) DeleteBucket(bucketName string) error {
	log := logger.WithFields(logrus.Fields{"tag": "DeleteAzureBlobContainer"})
	key, err := getStorageAccountKey(b)
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

	log.Infof("Azure Blob Container %s deleted", URL)

	return nil
}

func (b *AzureObjectStore) ListBuckets() error {
	return nil
}

func getStorageAccountKey(b *AzureObjectStore) (string, error) {
	log := logger.WithFields(logrus.Fields{"tag": "GetStorageAccountKey"})
	client, err := createStorageAccountClient(b.secret)
	if err != nil {
		return "", err
	}
	log.Infof("Getting key for storage account %s in resource group %s", b.storageAccount, b.resourceGroup)
	keys, err := client.ListKeys(context.TODO(), b.resourceGroup, b.storageAccount)
	if err != nil {
		log.Errorf("Error retrieving keys for StorageAccount %s", err.Error())
		return "", err
	}
	key := (*keys.Keys)[0].Value
	return *key, nil
}

func createStorageAccountClient(s *secret.SecretsItemResponse) (*storage.AccountsClient, error) {
	log := logger.WithFields(logrus.Fields{"tag": "CreateStorageAccountClient"})
	accountClient := storage.NewAccountsClient(s.Values[secret.AzureSubscriptionId])

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
	log := logger.WithFields(logrus.Fields{"tag": "CheckStorageAccount"})
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
	if *result.NameAvailable != true {
		log.Warnf("[%s] storage account name not available because %s", b.storageAccount, *result.Message)
		return true, fmt.Errorf("storage account name %s is already taken", b.storageAccount)
	}
	return false, nil
}

func createStorageAccount(b *AzureObjectStore) error {
	log := logger.WithFields(logrus.Fields{"tag": "CreateStorageAccount"})
	storageAccountsClient, err := createStorageAccountClient(b.secret)
	if err != nil {
		return err
	}
	log.Info("StorageAccount can be created")

	_, err = storageAccountsClient.Create(
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
	return nil
}

func createResourceGroup(b *AzureObjectStore) error {
	log := logger.WithFields(logrus.Fields{"tag": "CreateResourceGroup"})
	gclient := resources.NewGroupsClient(b.secret.Values[secret.AzureSubscriptionId])

	authorizer, err := newAuthorizer(b.secret)
	if err != nil {
		log.Errorf("Error happened during authentication %s", err.Error())
		return err
	}
	gclient.Authorizer = authorizer
	result, err := gclient.CreateOrUpdate(context.TODO(), b.resourceGroup,
		resources.Group{Location: to.StringPtr(b.location)})
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info(result.Status)
	return nil
}

func newAuthorizer(s *secret.SecretsItemResponse) (autorest.Authorizer, error) {
	log := logger.WithFields(logrus.Fields{"tag": "AzureAuthorizer"})
	log.Info("Authenticating...")

	authorizer, err := auth.NewClientCredentialsConfig(
		s.Values[secret.AzureClientId],
		s.Values[secret.AzureClientSecret],
		s.Values[secret.AzureTenantId]).Authorizer()

	if err != nil {
		return nil, err
	}

	return authorizer, nil
}
