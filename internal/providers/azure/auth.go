package azure

import (
	"github.com/Azure/go-autorest/autorest/azure/auth"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
)

// NewClientCredentialsConfigFromSecret returns an Azure client credential config from a secret.
//
// TODO: implement validation for the secret?
func NewClientCredentialsConfigFromSecret(secret map[string]string) auth.ClientCredentialsConfig {
	return auth.NewClientCredentialsConfig(
		secret[pkgSecret.AzureClientId],
		secret[pkgSecret.AzureClientSecret],
		secret[pkgSecret.AzureTenantId],
	)
}
