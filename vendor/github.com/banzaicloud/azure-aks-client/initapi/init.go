package initapi

import (
	"github.com/banzaicloud/azure-aks-client/cluster"
)

func Init() (*cluster.Sdk, *AzureErrorResponse) {
	clusterSdk, err := Authenticate()
	return clusterSdk, err
}
