package initapi

import (
	client "github.com/banzaicloud/azure-aks-client/client"
	cluster "github.com/banzaicloud/azure-aks-client/cluster"
)

func Init() *cluster.Sdk {

	var sdk cluster.Sdk
	sdk = *client.Authenticate()

	return &sdk
}
