// +build !cadence

package azure

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CreateBucket creates an Azure Object Store Blob with the provided name
// within a generated/provided ResourceGroup and StorageAccount
func (s *ObjectStore) CreateBucket(bucketName string) error {
	resourceGroup := s.getResourceGroup()
	storageAccount := s.getStorageAccount()

	logger := s.getLogger(bucketName)

	bucket := &ObjectStoreBucketModel{}
	searchCriteria := s.searchCriteria(bucketName)

	if err := s.db.Where(searchCriteria).Find(bucket).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return errors.Wrap(err, "error happened during getting bucket from DB: %s")
		}
	}

	// TODO: create the bucket in the database later so that we don't have to roll back
	bucket.ResourceGroup = resourceGroup
	bucket.Organization = *s.org

	logger.Info("saving bucket in DB")

	err := s.db.Save(bucket).Error
	if err != nil {
		return errors.Wrap(err, "error happened during saving bucket in DB")
	}

	err = s.createResourceGroup(resourceGroup)
	if err != nil {
		return s.rollback(logger, "resource group creation failed", err, bucket)
	}

	// update here so the bucket list does not get inconsistent
	updateField := &ObjectStoreBucketModel{StorageAccount: s.storageAccount}
	err = s.db.Model(bucket).Update(updateField).Error
	if err != nil {
		return errors.Wrap(err, "error happened during updating storage account")
	}

	exists, err := s.checkStorageAccountExistence(resourceGroup, storageAccount)
	if !exists && err == nil {
		err = s.createStorageAccount(resourceGroup, storageAccount)
		if err != nil {
			return s.rollback(logger, "storage account creation failed", err, bucket)
		}
	}

	if err != nil && !strings.Contains(err.Error(), "is already taken") {
		return s.rollback(logger, "storage account is already taken", err, bucket)
	}

	key, err := GetStorageAccountKey(resourceGroup, storageAccount, s.secret, s.logger)
	if err != nil {
		return err
	}

	// update here so the bucket list does not get inconsistent
	updateField = &ObjectStoreBucketModel{Name: bucketName, Location: s.location}
	err = s.db.Model(bucket).Update(updateField).Error
	if err != nil {
		return errors.Wrap(err, "error happened during updating bucket name")
	}

	p := azblob.NewPipeline(azblob.NewSharedKeyCredential(storageAccount, key), azblob.PipelineOptions{})
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", storageAccount, bucketName))
	containerURL := azblob.NewContainerURL(*URL, p)

	_, err = containerURL.GetPropertiesAndMetadata(context.TODO(), azblob.LeaseAccessConditions{})
	if err != nil && err.(azblob.StorageError).ServiceCode() == azblob.ServiceCodeContainerNotFound {
		_, err = containerURL.Create(context.TODO(), azblob.Metadata{}, azblob.PublicAccessNone)
		if err != nil {
			return s.rollback(logger, "cannot access bucket", err, bucket)
		}
	}

	return nil
}

func (s *ObjectStore) rollback(logger logrus.FieldLogger, msg string, err error, bucket *ObjectStoreBucketModel) error {
	e := s.db.Delete(bucket).Error
	if e != nil {
		logger.Error(e.Error())
	}

	return errors.Wrapf(err, "%s (rolling back)", msg)
}

func (s *ObjectStore) createResourceGroup(resourceGroup string) error {
	logger := s.logger.WithField("resource_group", resourceGroup)

	logger.Info("creating resource group")

	client, err := NewAuthorizedResourceGroupClientFromSecret(s.secret.Values)
	if err != nil {
		return emperror.Wrap(err, "failed to create resource group client")
	}

	res, _ := client.Get(context.TODO(), resourceGroup)

	if res.StatusCode == http.StatusNotFound {
		result, err := client.CreateOrUpdate(
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

func (s *ObjectStore) createStorageAccount(resourceGroup string, storageAccount string) error {
	client, err := NewAuthorizedStorageAccountClientFromSecret(s.secret.Values)
	if err != nil {
		return err
	}

	logger := s.logger.WithFields(logrus.Fields{
		"resource_group":  resourceGroup,
		"storage_account": storageAccount,
	})

	logger.Info("creating storage account")

	future, err := client.Create(
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
	if future.WaitForCompletion(context.TODO(), client.Client) != nil {
		return err
	}

	logger.Info("storage account created")

	return nil
}
