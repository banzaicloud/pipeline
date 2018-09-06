package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

const storageAccountResourceType = "Microsoft.Storage/storageAccounts"

// NewStorageAccountClientFromSecret returns an Azure resource groups client from a secret.
func NewStorageAccountClientFromSecret(secret map[string]string) storage.AccountsClient {
	return storage.NewAccountsClient(secret[pkgSecret.AzureSubscriptionId])
}

// NewAuthorizedStorageAccountClientFromSecret returns an authorized Azure resource groups client from a secret.
func NewAuthorizedStorageAccountClientFromSecret(secret map[string]string) (storage.AccountsClient, error) {
	credentials := NewClientCredentialsConfigFromSecret(secret)

	client := NewStorageAccountClientFromSecret(secret)

	authorizer, err := credentials.Authorizer()
	if err != nil {
		return client, errors.Wrap(err, "failed to authorize")
	}

	client.Authorizer = authorizer

	return client, nil
}

// StorageAccountClientFactory creates a new resource group client.
type StorageAccountClientFactory struct {
	secrets secretClient
}

// NewStorageAccountClientFactory returns a new resource group client factory.
func NewStorageAccountClientFactory(secrets secretClient) *StorageAccountClientFactory {
	return &StorageAccountClientFactory{
		secrets: secrets,
	}
}

func (f *StorageAccountClientFactory) New(organizationID uint, secretID string) (storage.AccountsClient, error) {
	var client storage.AccountsClient

	secret, err := f.secrets.Get(organizationID, secretID)
	if err != nil {
		return client, emperror.With(
			emperror.Wrap(err, "failed to get secret"),
			"organization-id", organizationID,
			"secret-id", secretID,
		)
	}

	return NewAuthorizedStorageAccountClientFromSecret(secret)
}

// CreateStorageAccount creates a storage account in an idempotent way.
func CreateStorageAccount(
	ctx context.Context,
	client storage.AccountsClient,
	resourceGroup string,
	storageAccount string,
	location string,
) error {
	res, err := client.CheckNameAvailability(
		ctx,
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(storageAccount),
			Type: to.StringPtr(storageAccountResourceType),
		},
	)
	if err != nil {
		return emperror.With(
			errors.Wrap(err, "failed to check storage account availability"),
			"storage-account", storageAccount,
		)
	}

	if *res.NameAvailable == false {
		switch res.Reason {
		case storage.AccountNameInvalid:
			// TODO: return custom error type here to indicate invalid account name (which should terminate any workflow)
			return emperror.With(
				errors.Wrap(err, "invalid account name"),
				"storage-account", storageAccount,
			)

		case storage.AlreadyExists:
			// TODO: differentiate not found and unexpected errors
			_, err := client.GetProperties(ctx, resourceGroup, storageAccount)
			if err != nil {
				return emperror.With(
					errors.Wrap(err, "storage account exists but it is not in your resource group"),
					"resource-group", resourceGroup,
					"storage-account", storageAccount,
				)
			}

			// storage account already exists, but it's in your resource group
			return nil
		}
	}

	future, err := client.Create(
		ctx,
		resourceGroup,
		storageAccount,
		storage.AccountCreateParameters{
			Sku: &storage.Sku{
				Name: storage.StandardLRS,
			},
			Kind:     storage.BlobStorage,
			Location: to.StringPtr(location),
			AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{
				AccessTier: storage.Hot,
			},
		},
	)
	if err != nil {
		return emperror.With(
			errors.Wrap(err, "failed to create storage account"),
			"resource-group", resourceGroup,
			"storage-account", storageAccount,
			"location", location,
		)
	}

	if future.WaitForCompletion(ctx, client.Client) != nil {
		return emperror.With(
			errors.Wrap(err, "failed to create storage account"),
			"resource-group", resourceGroup,
			"storage-account", storageAccount,
			"location", location,
		)
	}

	return nil
}

// GetStorageAccountKey returns a key for a storage account.
func GetStorageAccountKey(
	ctx context.Context,
	client storage.AccountsClient,
	resourceGroup string,
	storageAccount string,
) (string, error) {
	keys, err := client.ListKeys(ctx, resourceGroup, storageAccount)
	if err != nil {
		return "", emperror.With(
			errors.Wrap(err, "failed to retrieve keys for storage account"),
			"storage-account", storageAccount,
		)
	}

	key := (*keys.Keys)[0].Value

	return *key, nil
}
