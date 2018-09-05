package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

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
