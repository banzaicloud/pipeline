package cluster

import (
	"os"

	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/banzaicloud/azure-aks-client/utils"
)

const AzureClientId = "AZURE_CLIENT_ID"
const AzureClientSecret = "AZURE_CLIENT_SECRET"
const AzureSubscriptionId = "AZURE_SUBSCRIPTION_ID"
const AzureTenantId = "AZURE_TENANT_ID"

type AKSCredential struct {
	ClientId       string
	ClientSecret   string
	SubscriptionId string
	TenantId       string
}

type Sdk struct {
	ServicePrincipal        *ServicePrincipal
	ManagedClusterClient    *containerservice.ManagedClustersClient
	VMSizeClient            *compute.VirtualMachineSizesClient
	SubscriptionsClient     *subscriptions.Client
	ContainerServicesClient *containerservice.ContainerServicesClient
}

type ServicePrincipal struct {
	ClientID           string
	ClientSecret       string
	SubscriptionID     string
	TenantId           string
	HashMap            map[string]string
	AuthenticatedToken *adal.ServicePrincipalToken
}

func (a *AKSCredential) Validate() error {
	msg := "missing credential: "
	if len(a.ClientId) == 0 {
		return utils.NewErr(msg + "clientId")
	}
	if len(a.ClientSecret) == 0 {
		return utils.NewErr(msg + "ClientSecret")
	}
	if len(a.SubscriptionId) == 0 {
		return utils.NewErr(msg + "SubscriptionId")
	}
	if len(a.TenantId) == 0 {
		return utils.NewErr(msg + "TenantId")
	}
	return nil
}

func Authenticate(credentials *AKSCredential) (*Sdk, error) {
	var AKSCred *AKSCredential
	if credentials != nil {
		AKSCred = credentials
	} else {
		AKSCred = &AKSCredential{}
		AKSCred.ClientId = os.Getenv(AzureClientId)
		AKSCred.ClientSecret = os.Getenv(AzureClientSecret)
		AKSCred.SubscriptionId = os.Getenv(AzureSubscriptionId)
		AKSCred.TenantId = os.Getenv(AzureTenantId)
	}

	err := AKSCred.Validate()
	if err != nil {
		return nil, err
	}

	sdk := Sdk{
		ServicePrincipal: &ServicePrincipal{
			ClientID:       AKSCred.ClientId,
			ClientSecret:   AKSCred.ClientSecret,
			SubscriptionID: AKSCred.SubscriptionId,
			TenantId:       AKSCred.TenantId,
			HashMap: map[string]string{
				AzureClientId:       AKSCred.ClientId,
				AzureClientSecret:   AKSCred.ClientSecret,
				AzureSubscriptionId: AKSCred.SubscriptionId,
				AzureTenantId:       AKSCred.TenantId,
			},
		},
	}
	authorizer, err := auth.NewClientCredentialsConfig(AKSCred.ClientId, AKSCred.ClientSecret, AKSCred.TenantId).Authorizer()
	if err != nil {
		return nil, utils.NewErr(fmt.Sprintf("authentication error: %s", err))
	}

	subscriptionId := sdk.ServicePrincipal.SubscriptionID
	managedClusterClient := containerservice.NewManagedClustersClient(subscriptionId)
	vmSizesClient := compute.NewVirtualMachineSizesClient(subscriptionId)
	subscriptionsClient := subscriptions.NewClient()
	containerServicesClient := containerservice.NewContainerServicesClient(subscriptionId)

	managedClusterClient.Authorizer = authorizer
	vmSizesClient.Authorizer = authorizer
	subscriptionsClient.Authorizer = authorizer
	containerServicesClient.Authorizer = authorizer

	sdk.ManagedClusterClient = &managedClusterClient
	sdk.VMSizeClient = &vmSizesClient
	sdk.SubscriptionsClient = &subscriptionsClient
	sdk.ContainerServicesClient = &containerServicesClient

	return &sdk, nil
}
