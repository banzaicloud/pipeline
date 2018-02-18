package cluster

import (
	"os"

	"github.com/Azure/azure-sdk-for-go/arm/examples/helpers"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/banzaicloud/azure-aks-client/utils"
	"fmt"
	"github.com/Azure/go-autorest/autorest/adal"
)

const AzureClientId = "AZURE_CLIENT_ID"
const AzureClientSecret = "AZURE_CLIENT_SECRET"
const AzureSubscriptionId = "AZURE_SUBSCRIPTION_ID"
const AzureTenantId = "AZURE_TENANT_ID"

type AKSCredential struct {
	clientId string
	clientSecret string
	subscriptionId string
	tenantId string
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

func (a *AKSCredential)Validate() error {
	msg := "missing credential: "
	if len(a.clientId) == 0 {
		return utils.NewErr(msg+"clientId")
	}
	if len(a.clientSecret) == 0 {
		return utils.NewErr(msg+"clientSecret")
	}
	if len(a.subscriptionId) == 0 {
		return utils.NewErr(msg+"subscriptionId")
	}
	if len(a.tenantId) == 0 {
		return utils.NewErr(msg+"tenantId")
	}
	return nil
}

func Authenticate(credentials *AKSCredential) (*Sdk, error) {
	var AKSCred *AKSCredential
	if credentials != nil {
		AKSCred = credentials
	} else {
		AKSCred = &AKSCredential{}
		AKSCred.clientId = os.Getenv(AzureClientId)
		AKSCred.clientSecret = os.Getenv(AzureClientSecret)
		AKSCred.subscriptionId = os.Getenv(AzureSubscriptionId)
		AKSCred.tenantId = os.Getenv(AzureTenantId)
	}

	err := AKSCred.Validate()
	if err != nil {
		return nil, err
	}


	sdk := Sdk{
		ServicePrincipal: &ServicePrincipal{
			ClientID:       AKSCred.clientId,
			ClientSecret:   AKSCred.clientSecret,
			SubscriptionID: AKSCred.subscriptionId,
			TenantId:       AKSCred.tenantId,
			HashMap: map[string]string{
				AzureClientId:       AKSCred.clientId,
				AzureClientSecret:   AKSCred.clientSecret,
				AzureSubscriptionId: AKSCred.subscriptionId,
				AzureTenantId:       AKSCred.tenantId,
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