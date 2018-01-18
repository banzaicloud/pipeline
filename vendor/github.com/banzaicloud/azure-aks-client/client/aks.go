package client

import (
	"encoding/json"
	"github.com/Azure/go-autorest/autorest"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"io/ioutil"
	"net/http"
	"time"
	"github.com/banzaicloud/azure-aks-client/initapi"
	"errors"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiTypes "github.com/banzaicloud/banzai-types/components"
	banzaiTypesAzure "github.com/banzaicloud/banzai-types/components/azure"
)

func init() {
	azureSdk, initError = initapi.Init()
	if azureSdk != nil {
		clientId = azureSdk.ServicePrincipal.ClientID
		secret = azureSdk.ServicePrincipal.ClientSecret
	}
}

const BaseUrl = "https://management.azure.com"

var azureSdk *cluster.Sdk
var clientId string
var secret string
var initError *banzaiTypes.BanzaiResponse

/**
GetCluster gets the details of the managed cluster with a specified resource group and name.
GET https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/
	{resourceName}?api-version=2017-08-31
 */
func GetCluster(name string, resourceGroup string) (*banzaiTypesAzure.ResponseWithValue, *banzaiTypes.BanzaiResponse) {

	resp, errAz := callAzureGetCluster(name, resourceGroup)
	if errAz != nil {
		return nil, errAz
	}

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "error during get cluster in ", resourceGroup, " resource group:", err)
		return nil, createErrorResponseFromError(err)
	}

	if resp.StatusCode != banzaiConstants.OK {
		// not ok, probably 404
		errResp := initapi.CreateErrorFromValue(resp.StatusCode, value)
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Get cluster failed with message: ", errResp.Message)
		return nil, &banzaiTypes.BanzaiResponse{StatusCode: resp.StatusCode, Message: errResp.Message}
	} else {
		// everything is ok
		v := banzaiTypesAzure.Value{}
		json.Unmarshal([]byte(value), &v)
		response := banzaiTypesAzure.ResponseWithValue{}
		response.Update(resp.StatusCode, v)
		return &response, nil
	}

}

/*
ListClusters is listing AKS clusters in the specified subscription and resource group
GET https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters?
	api-version=2017-08-31
*/
func ListClusters(resourceGroup string) (*banzaiTypesAzure.ListResponse, *banzaiTypes.BanzaiResponse) {

	if azureSdk == nil {
		return nil, initError
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		banzaiUtils.LogError(banzaiConstants.TagListClusters, "environmental_error")
		return nil, &banzaiTypes.BanzaiResponse{StatusCode: banzaiConstants.InternalErrorCode, Message: message}
	}

	pathParam := map[string]interface{}{
		"subscription-id": azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagListClusters, "error during listing clusters in ", resourceGroup, " resource group:", err)
		return nil, createErrorResponseFromError(err)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagListClusters, "Start cluster listing in ", resourceGroup, " resource group")

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagListClusters, "error during listing clusters in ", resourceGroup, " resource group:", err)
		return nil, createErrorResponseFromError(err)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagListClusters, "Cluster list response status code: ", resp.StatusCode)

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagListClusters, "error during listing clusters in ", resourceGroup, " resource group:", err)
		return nil, createErrorResponseFromError(err)
	}

	if resp.StatusCode != banzaiConstants.OK {
		// not ok, probably 404
		errResp := initapi.CreateErrorFromValue(resp.StatusCode, value)
		banzaiUtils.LogInfo(banzaiConstants.TagListClusters, "Listing clusters failed with message: ", errResp.Message)
		return nil, &banzaiTypes.BanzaiResponse{StatusCode: resp.StatusCode, Message: errResp.Message}
	}

	azureListResponse := banzaiTypesAzure.Values{}
	json.Unmarshal([]byte(value), &azureListResponse)
	banzaiUtils.LogInfo(banzaiConstants.TagListClusters, "List cluster result ", &azureListResponse)

	response := banzaiTypesAzure.ListResponse{StatusCode: resp.StatusCode, Value: azureListResponse}
	return &response, nil
}

/*
CreateUpdateCluster creates or updates a managed cluster
PUT https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?
	api-version=2017-08-31sdk *cluster.Sdk
*/
func CreateUpdateCluster(request cluster.CreateClusterRequest) (*banzaiTypesAzure.ResponseWithValue, *banzaiTypes.BanzaiResponse) {

	if azureSdk == nil {
		return nil, initError
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		banzaiUtils.LogError(banzaiConstants.TagCreateCluster, "environmental_error")
		return nil, &banzaiTypes.BanzaiResponse{StatusCode: banzaiConstants.InternalErrorCode, Message: message}
	}

	if isValid, errMsg := request.Validate(); !isValid {
		return nil, &banzaiTypes.BanzaiResponse{StatusCode: banzaiConstants.BadRequest, Message: errMsg}
	}

	managedCluster := cluster.GetManagedCluster(request, clientId, secret)

	pathParam := map[string]interface{}{
		"subscription-id": azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   request.ResourceGroup,
		"resourceName":    request.Name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *azureSdk.ResourceGroup

	req, _ := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsPut(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam),
		autorest.WithJSON(managedCluster),
		autorest.AsContentType("application/json"),
	)

	_, err := json.Marshal(managedCluster)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagCreateCluster, "error during JSON marshal:", err)
		return nil, createErrorResponseFromError(err)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Cluster creation start with name ", request.Name, " in ", request.ResourceGroup, " resource group")

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagCreateCluster, "error during cluster creation:", err)
		return nil, createErrorResponseFromError(err)
	}

	defer resp.Body.Close()
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagCreateCluster, "error during cluster creation:", err)
		return nil, createErrorResponseFromError(err)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Cluster create response code: ", resp.StatusCode)

	if resp.StatusCode != banzaiConstants.OK && resp.StatusCode != banzaiConstants.Created {
		// something went wrong, create failed
		errResp := initapi.CreateErrorFromValue(resp.StatusCode, value)
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Cluster creation failed with message: ", errResp.Message)
		return nil, &banzaiTypes.BanzaiResponse{StatusCode: resp.StatusCode, Message: errResp.Message}
	}

	v := banzaiTypesAzure.Value{}
	json.Unmarshal([]byte(value), &v)
	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Cluster creation with name ", request.Name, " in ", request.ResourceGroup, " resource group has started")

	result := banzaiTypesAzure.ResponseWithValue{StatusCode: resp.StatusCode, Value: v}
	return &result, nil
}

/*
DeleteCluster deletes a managed AKS on Azure
DELETE https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?
	api-version=2017-08-31
*/
func DeleteCluster(name string, resourceGroup string) (*banzaiTypes.BanzaiResponse, bool) {

	if azureSdk == nil {
		return initError, false
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		banzaiUtils.LogError(banzaiConstants.TagDeleteCluster, "environmental_error")
		return &banzaiTypes.BanzaiResponse{StatusCode: banzaiConstants.InternalErrorCode, Message: message}, false
	}

	pathParam := map[string]interface{}{
		"subscription-id": azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup,
		"resourceName":    name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsDelete(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam),
	)

	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagDeleteCluster, "error during delete cluster:", err)
		return createErrorResponseFromError(err), false
	}

	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Cluster delete start with name ", name, " in ", resourceGroup, " resource group")

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagDeleteCluster, "error during delete cluster:", err)
		return createErrorResponseFromError(err), false
	}

	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Cluster delete status code: ", resp.StatusCode)

	defer resp.Body.Close()
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagDeleteCluster, "error during delete cluster:", err)
		return createErrorResponseFromError(err), false
	}

	if resp.StatusCode != banzaiConstants.OK && resp.StatusCode != banzaiConstants.NoContent && resp.StatusCode != banzaiConstants.Accepted {
		errResp := initapi.CreateErrorFromValue(resp.StatusCode, value)
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Delete cluster failed with message: ", errResp.Message)
		return &banzaiTypes.BanzaiResponse{StatusCode: resp.StatusCode, Message: errResp.Message}, false
	}

	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Delete cluster response ", string(value))

	result := banzaiTypes.BanzaiResponse{StatusCode: resp.StatusCode}
	return &result, true
}

/*
PollingCluster polling AKS on Azure
GET https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?
	api-version=2017-08-31
 */
func PollingCluster(name string, resourceGroup string) (*banzaiTypesAzure.ResponseWithValue, *banzaiTypes.BanzaiResponse) {

	if azureSdk == nil {
		return nil, initError
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		banzaiUtils.LogError(banzaiConstants.TagGetClusterInfo, "environmental_error")
		return nil, &banzaiTypes.BanzaiResponse{StatusCode: banzaiConstants.InternalErrorCode, Message: message}
	}

	const stageSuccess = "Succeeded"
	const stageFailed = "Failed"
	const waitInSeconds = 10

	pathParam := map[string]interface{}{
		"subscription-id": azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup,
		"resourceName":    name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam),
	)

	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagGetClusterInfo, "error during cluster polling:", err)
		return nil, createErrorResponseFromError(err)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Cluster polling start with name ", name, " in ", resourceGroup, " resource group")

	result := banzaiTypesAzure.ResponseWithValue{}
	for isReady := false; !isReady; {

		resp, err := autorest.SendWithSender(groupClient.Client, req)
		if err != nil {
			banzaiUtils.LogError(banzaiConstants.TagGetClusterInfo, "error during cluster polling:", err)
			return nil, createErrorResponseFromError(err)
		}

		statusCode := resp.StatusCode
		banzaiUtils.LogDebug(banzaiConstants.TagGetClusterInfo, "Cluster polling status code: ", statusCode)

		value, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			banzaiUtils.LogError(banzaiConstants.TagGetClusterInfo, "error during cluster polling:", err)
			return nil, createErrorResponseFromError(err)
		}

		switch statusCode {
		case banzaiConstants.OK:
			response := banzaiTypesAzure.Value{}
			json.Unmarshal([]byte(value), &response)

			stage := response.Properties.ProvisioningState
			banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Cluster stage is ", stage)

			switch stage {
			case stageSuccess:
				isReady = true
				result.Update(statusCode, response)
			case stageFailed:
				return nil, createErrorResponseFromError(errors.New("cluster stage is 'Failed'"))
			default:
				banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Waiting...")
				time.Sleep(waitInSeconds * time.Second)
			}

		default:
			errResp := initapi.CreateErrorFromValue(resp.StatusCode, value)
			banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Delete cluster failed with message: ", errResp.Message)
			return nil, &banzaiTypes.BanzaiResponse{StatusCode: resp.StatusCode, Message: errResp.Message}
		}
	}

	return &result, nil
}

func createErrorResponseFromError(err error) *banzaiTypes.BanzaiResponse {
	return &banzaiTypes.BanzaiResponse{
		StatusCode: banzaiConstants.InternalErrorCode,
		Message:    err.Error(),
	}
}

/**
Get kubernetes cluster config
GET https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/
	{resourceName}?api-version=2017-08-31
 */
func GetClusterConfig(name, resourceGroup, roleName string) (*banzaiTypesAzure.Config, *banzaiTypes.BanzaiResponse) {

	if azureSdk == nil {
		return nil, initError
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "environmental_error")
		return nil, &banzaiTypes.BanzaiResponse{StatusCode: banzaiConstants.InternalErrorCode, Message: message}
	}

	pathParam := map[string]interface{}{
		"subscriptionId":    azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroupName": resourceGroup,
		"resourceName":      name,
		"roleName":          roleName,
	}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/accessProfiles/{roleName}", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "error during get cluster in", resourceGroup, "resource group", err)
		return nil, createErrorResponseFromError(err)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Get cluster with name:", name, " in ", resourceGroup, "resource group")

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "error during get clusters in", resourceGroup, "resource group:", err)
		return nil, createErrorResponseFromError(err)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Get Cluster response status code:", resp.StatusCode)

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "error during get cluster in ", resourceGroup, " resource group:", err)
		return nil, createErrorResponseFromError(err)
	}

	if resp.StatusCode != banzaiConstants.OK {
		// not ok, probably 404
		errResp := initapi.CreateErrorFromValue(resp.StatusCode, value)
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Get k8s config failed with message: ", errResp.Message)
		return nil, &banzaiTypes.BanzaiResponse{StatusCode: resp.StatusCode, Message: errResp.Message}
	} else {
		// everything is ok
		res := banzaiTypesAzure.Config{}
		json.Unmarshal([]byte(value), &res)
		return &res, nil
	}

}

func callAzureGetCluster(name, resourceGroup string) (*http.Response, *banzaiTypes.BanzaiResponse) {

	if azureSdk == nil {
		return nil, initError
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "environmental_error")
		return nil, &banzaiTypes.BanzaiResponse{StatusCode: banzaiConstants.InternalErrorCode, Message: message}
	}

	pathParam := map[string]interface{}{
		"subscription-id": azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup,
		"resourceName":    name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "error during get cluster in", resourceGroup, "resource group", err)
		return nil, createErrorResponseFromError(err)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Get cluster with name:", name, " in ", resourceGroup, "resource group")

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "error during get clusters in", resourceGroup, "resource group:", err)
		return nil, createErrorResponseFromError(err)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Get Cluster response status code:", resp.StatusCode)
	return resp, nil
}
