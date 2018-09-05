package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

// NewResourceGroupClientFromSecret returns an Azure resource groups client from a secret.
func NewResourceGroupClientFromSecret(secret map[string]string) resources.GroupsClient {
	return resources.NewGroupsClient(secret[pkgSecret.AzureSubscriptionId])
}

// NewAuthorizedResourceGroupClientFromSecret returns an authorized Azure resource groups client from a secret.
func NewAuthorizedResourceGroupClientFromSecret(secret map[string]string) (resources.GroupsClient, error) {
	credentials := NewClientCredentialsConfigFromSecret(secret)

	client := NewResourceGroupClientFromSecret(secret)

	authorizer, err := credentials.Authorizer()
	if err != nil {
		return client, errors.Wrap(err, "failed to authorize")
	}

	client.Authorizer = authorizer

	return client, nil
}

// ResourceGroupClientFactory creates a new resource group client.
type ResourceGroupClientFactory struct {
	secrets secretClient
}

// NewResourceGroupClientFactory returns a new resource group client factory.
func NewResourceGroupClientFactory(secrets secretClient) *ResourceGroupClientFactory {
	return &ResourceGroupClientFactory{
		secrets: secrets,
	}
}

func (f *ResourceGroupClientFactory) New(organizationID uint, secretID string) (resources.GroupsClient, error) {
	var client resources.GroupsClient

	secret, err := f.secrets.Get(organizationID, secretID)
	if err != nil {
		return client, emperror.With(
			emperror.Wrap(err, "failed to get secret"),
			"organization-id", organizationID,
			"secret-id", secretID,
		)
	}

	return NewAuthorizedResourceGroupClientFromSecret(secret)
}
