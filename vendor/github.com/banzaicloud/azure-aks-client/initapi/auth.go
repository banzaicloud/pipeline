package initapi

import (
	"os"

	"github.com/Azure/azure-sdk-for-go/arm/examples/helpers"
	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/banzaicloud/azure-aks-client/cluster"
	banzaiTypes "github.com/banzaicloud/banzai-types/components"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	"encoding/json"
)

var sdk cluster.Sdk

const azureClientId = "AZURE_CLIENT_ID"
const azureClientSecret = "AZURE_CLIENT_SECRET"
const azureSubscriptionId = "AZURE_SUBSCRIPTION_ID"
const azureTenantId = "AZURE_TENANT_ID"

func Authenticate() (*cluster.Sdk, *banzaiTypes.BanzaiResponse) {
	clientId := os.Getenv(azureClientId)
	clientSecret := os.Getenv(azureClientSecret)
	subscriptionId := os.Getenv(azureSubscriptionId)
	tenantId := os.Getenv(azureTenantId)

	// ---- [Check Environmental variables] ---- //
	if len(clientId) == 0 {
		return nil, CreateEnvErrorResponse(azureClientId)
	}

	if len(clientSecret) == 0 {
		return nil, CreateEnvErrorResponse(azureClientSecret)
	}

	if len(subscriptionId) == 0 {
		return nil, CreateEnvErrorResponse(azureSubscriptionId)
	}

	if len(tenantId) == 0 {
		return nil, CreateEnvErrorResponse(azureTenantId)
	}

	sdk = cluster.Sdk{
		ServicePrincipal: &cluster.ServicePrincipal{
			ClientID:       clientId,
			ClientSecret:   clientSecret,
			SubscriptionID: subscriptionId,
			TenantId:       tenantId,
			HashMap: map[string]string{
				"AZURE_CLIENT_ID":       clientId,
				"AZURE_CLIENT_SECRET":   clientSecret,
				"AZURE_SUBSCRIPTION_ID": subscriptionId,
				"AZURE_TENANT_ID":       tenantId,
			},
		},
	}

	authenticatedToken, err := helpers.NewServicePrincipalTokenFromCredentials(sdk.ServicePrincipal.HashMap, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, CreateAuthErrorResponse(err)
	}

	sdk.ServicePrincipal.AuthenticatedToken = authenticatedToken

	resourceGroup := resources.NewGroupsClient(sdk.ServicePrincipal.SubscriptionID)
	resourceGroup.Authorizer = autorest.NewBearerAuthorizer(sdk.ServicePrincipal.AuthenticatedToken)
	sdk.ResourceGroup = &resourceGroup

	return &sdk, nil
}

func GetSdk() *cluster.Sdk {
	return &sdk
}

type AzureServerError struct {
	Message string `json:"message"`
}

func CreateErrorFromValue(statusCode int, v []byte) AzureServerError {
	if statusCode == banzaiConstants.BadRequest {
		ase := AzureServerError{}
		json.Unmarshal([]byte(v), &ase)
		return ase
	} else {
		type TempError struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		tempError := TempError{}
		json.Unmarshal([]byte(v), &tempError)
		return AzureServerError{Message: tempError.Error.Message}
	}
}

func CreateEnvErrorResponse(env string) *banzaiTypes.BanzaiResponse {
	message := "Environmental variable is empty: " + env
	banzaiUtils.LogError(banzaiConstants.TagInit, "environmental_error")
	return &banzaiTypes.BanzaiResponse{StatusCode: banzaiConstants.InternalErrorCode, Message: message}
}

func CreateAuthErrorResponse(err error) *banzaiTypes.BanzaiResponse {
	errMsg := "Failed to authenticate with Azure"
	banzaiUtils.LogError(banzaiConstants.TagAuth, "Authentication error:", err)
	return &banzaiTypes.BanzaiResponse{StatusCode: banzaiConstants.InternalErrorCode, Message: errMsg}
}
