package initapi

import (
	"github.com/banzaicloud/azure-aks-client/cluster"
	banzaiTypes "github.com/banzaicloud/banzai-types/components"
)

func Init() (*cluster.Sdk, *banzaiTypes.BanzaiResponse) {
	clusterSdk, err := Authenticate()
	return clusterSdk, err
}
