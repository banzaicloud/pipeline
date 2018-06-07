package cloud

import (
	"github.com/banzaicloud/azure-aks-client/client"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/pipeline/constants"
)

// aksVerify for validation AKS credentials
type aksVerify struct {
	credential *cluster.AKSCredential
}

// CreateAKSSecret create a new 'aksVerify' instance
func CreateAKSSecret(values map[string]string) *aksVerify {
	return &aksVerify{
		credential: CreateAKSCredentials(values),
	}
}

// VerifySecret validates AKS credentials
func (a *aksVerify) VerifySecret() (err error) {
	manager, err := client.GetAKSClient(a.credential)
	if err != nil {
		return
	}

	return client.ValidateCredentials(manager)
}

// CreateAKSCredentials create an 'AKSCredential' instance from secret's values
func CreateAKSCredentials(values map[string]string) *cluster.AKSCredential {
	return &cluster.AKSCredential{
		ClientId:       values[constants.AzureClientId],
		ClientSecret:   values[constants.AzureClientSecret],
		SubscriptionId: values[constants.AzureSubscriptionId],
		TenantId:       values[constants.AzureTenantId],
	}
}
