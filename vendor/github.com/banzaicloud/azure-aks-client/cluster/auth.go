package cluster

import (
	"os"

	"fmt"
	"github.com/Azure/azure-sdk-for-go/arm/examples/helpers"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
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
	ServicePrincipal *ServicePrincipal
	ResourceGroup    *resources.GroupsClient
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

	authenticatedToken, err := helpers.NewServicePrincipalTokenFromCredentials(sdk.ServicePrincipal.HashMap, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, utils.NewErr(fmt.Sprintf("authentication error: %s", err))
	}

	sdk.ServicePrincipal.AuthenticatedToken = authenticatedToken

	resourceGroup := resources.NewGroupsClient(sdk.ServicePrincipal.SubscriptionID)
	resourceGroup.Authorizer = autorest.NewBearerAuthorizer(sdk.ServicePrincipal.AuthenticatedToken)
	sdk.ResourceGroup = &resourceGroup

	return &sdk, nil
}
