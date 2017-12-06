package client

import (
	"encoding/json"
	"github.com/Azure/go-autorest/autorest"
	"github.com/banzaicloud/azure-aks-client/cluster"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"github.com/banzaicloud/azure-aks-client/initapi"
)

func init() {
	// Log as JSON
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	azureSdk, initError = initapi.Init()
	if azureSdk != nil {
		clientId = azureSdk.ServicePrincipal.ClientID
		secret = azureSdk.ServicePrincipal.ClientSecret
	}
}

var azureSdk *cluster.Sdk
var clientId string
var secret string
var initError *initapi.AzureErrorResponse

/**
GetCluster gets the details of the managed cluster with a specified resource group and name.
GET https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/
	{resourceName}?api-version=2017-08-31
 */
func GetCluster(name string, resourceGroup string) (*Response, *initapi.AzureErrorResponse) {

	if azureSdk == nil {
		return nil, initError
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		log.WithFields(log.Fields{"error": "environmental_error"}).Error(message)
		return nil, &initapi.AzureErrorResponse{StatusCode: initapi.InternalErrorCode, Message: message}
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
		autorest.WithBaseURL("https://management.azure.com"),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during listing clusters in ", resourceGroup, " resource group")
		return nil, createErrorResponse()
	}

	log.Info("Get cluster details with name: ", name, " in ", resourceGroup, " resource group")

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during listing clusters in ", resourceGroup, " resource group")
		return nil, createErrorResponse()
	}

	log.Info("Get Cluster response status code: ", resp.StatusCode)

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during get cluster in ", resourceGroup, " resource group")
		return nil, createErrorResponse()
	}

	if resp.StatusCode != initapi.OK {
		// not ok, probably 404
		type TempErrorResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}

		errResp := TempErrorResp{}
		json.Unmarshal([]byte(value), &errResp)
		return nil, &initapi.AzureErrorResponse{StatusCode: resp.StatusCode, Message: errResp.Error.Message}
	} else {
		// everything is ok
		v := Value{}
		json.Unmarshal([]byte(value), &v)
		response := Response{}
		response.update(resp.StatusCode, v)
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
func ListClusters(resourceGroup string) (*ListResponse, *initapi.AzureErrorResponse) {

	if azureSdk == nil {
		return nil, initError
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		log.WithFields(log.Fields{"error": "environmental_error"}).Error(message)
		return nil, &initapi.AzureErrorResponse{StatusCode: initapi.InternalErrorCode, Message: message}
	}

	pathParam := map[string]interface{}{
		"subscription-id": azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL("https://management.azure.com"),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during listing clusters in ", resourceGroup, " resource group")
		return nil, createErrorResponse()
	}

	log.Info("Start cluster listing in ", resourceGroup, " resource group")

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during listing clusters in ", resourceGroup, " resource group")
		return nil, createErrorResponse()
	}

	log.Info("Cluster list response status code: ", resp.StatusCode)

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during listing clusters in ", resourceGroup, " resource group")
		return nil, createErrorResponse()
	}

	azureListResponse := AzureListResponse{}
	json.Unmarshal([]byte(value), &azureListResponse)
	log.Info("List cluster result ", azureListResponse.toString())

	response := ListResponse{StatusCode: resp.StatusCode, Value: azureListResponse}
	return &response, nil
}

/*
CreateCluster creates a managed AKS on Azure
PUT https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?
	api-version=2017-08-31sdk *cluster.Sdk
*/
func CreateCluster(request cluster.CreateClusterRequest) (*Response, *initapi.AzureErrorResponse) {

	if azureSdk == nil {
		return nil, initError
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		log.WithFields(log.Fields{"error": "environmental_error"}).Error(message)
		return nil, &initapi.AzureErrorResponse{StatusCode: initapi.InternalErrorCode, Message: message}
	}

	if isValid, errMsg := request.Validate(); !isValid {
		return nil, &initapi.AzureErrorResponse{StatusCode: initapi.BadRequest, Message: errMsg}
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
		autorest.WithBaseURL("https://management.azure.com"),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam),
		autorest.WithJSON(managedCluster),
		autorest.AsContentType("application/json"),
	)

	_, err := json.Marshal(managedCluster)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during JSON marshal")
		return nil, createErrorResponse()
	}

	log.Info("Cluster creation start with name ", request.Name, " in ", request.ResourceGroup, " resource group")

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during cluster creation")
		return nil, createErrorResponse()
	}

	defer resp.Body.Close()
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during cluster creation")
		return nil, createErrorResponse()
	}

	log.Info("Cluster create response code: ", resp.StatusCode)

	v := Value{}
	json.Unmarshal([]byte(value), &v)
	log.Info("Cluster creation with name ", request.Name, " in ", request.ResourceGroup, " resource group has started")

	result := Response{StatusCode: resp.StatusCode, Value: v}
	return &result, nil
}

/*
DeleteCluster deletes a managed AKS on Azure
DELETE https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?
	api-version=2017-08-31
*/
func DeleteCluster(name string, resourceGroup string) (*Response, *initapi.AzureErrorResponse) {

	if azureSdk == nil {
		return nil, initError
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		log.WithFields(log.Fields{"error": "environmental_error"}).Error(message)
		return nil, &initapi.AzureErrorResponse{StatusCode: initapi.InternalErrorCode, Message: message}
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
		autorest.WithBaseURL("https://management.azure.com"),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam),
	)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during delete cluster")
		return nil, createErrorResponse()
	}

	log.Info("Cluster delete start with name ", name, " in ", resourceGroup, " resource group")

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during delete cluster")
		return nil, createErrorResponse()
	}

	log.Info("Cluster delete status code: ", resp.StatusCode)

	defer resp.Body.Close()
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during delete cluster")
		return nil, createErrorResponse()
	}

	log.Info("Delete cluster response ", string(value))

	result := Response{StatusCode: resp.StatusCode}
	return &result, nil
}

/*
PollingCluster polling AKS on Azure
GET https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?
	api-version=2017-08-31
 */
func PollingCluster(name string, resourceGroup string) (*Response, *initapi.AzureErrorResponse) {

	if azureSdk == nil {
		return nil, initError
	}

	if len(clientId) == 0 || len(secret) == 0 {
		message := "ClientId or secret is empty"
		log.WithFields(log.Fields{"error": "environmental_error"}).Error(message)
		return nil, &initapi.AzureErrorResponse{StatusCode: initapi.InternalErrorCode, Message: message}
	}

	const OK = 200
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
		autorest.WithBaseURL("https://management.azure.com"),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam),
	)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("error during cluster polling")
		return nil, createErrorResponse()
	}

	log.Info("Cluster polling start with name ", name, " in ", resourceGroup, " resource group")

	result := Response{}
	for isReady := false; !isReady; {

		resp, err := autorest.SendWithSender(groupClient.Client, req)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("error during cluster polling")
			return nil, createErrorResponse()
		}

		statusCode := resp.StatusCode
		log.Info("Cluster polling status code: ", statusCode)

		switch statusCode {
		case OK:
			value, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.WithFields(log.Fields{"error": err}).Error("error during cluster polling")
				return nil, createErrorResponse()
			}

			response := Value{}
			json.Unmarshal([]byte(value), &response)

			stage := response.Properties.ProvisioningState
			log.Info("Cluster stage is ", stage)

			switch stage {
			case stageSuccess:
				isReady = true
				result.update(statusCode, response)
			case stageFailed:
				return nil, createErrorResponse()
			default:
				log.Info("Waiting...")
				time.Sleep(waitInSeconds * time.Second)
			}

		default:
			return nil, createErrorResponseWithCode(statusCode)
		}
	}

	return &result, nil
}

type AzureListResponse struct {
	Value []Value `json:"value"`
}

type Value struct {
	Id         string     `json:"id"`
	Location   string     `json:"location"`
	Name       string     `json:"name"`
	Properties Properties `json:"properties"`
}

type Properties struct {
	ProvisioningState string    `json:"provisioningState"`
	AgentPoolProfiles []Profile `json:"agentPoolProfiles"`
}

type Profile struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Response struct {
	StatusCode int   `json:"status_code"`
	Value      Value `json:"message"`
}

type ListResponse struct {
	StatusCode int               `json:"status_code"`
	Value      AzureListResponse `json:"message"`
}

func (r AzureListResponse) toString() string {
	jsonResponse, _ := json.Marshal(r)
	return string(jsonResponse)
}

func (v Value) ToString() string {
	jsonResponse, _ := json.Marshal(v)
	return string(jsonResponse)
}

func (r Response) toString() string {
	jsonResponse, _ := json.Marshal(r)
	return string(jsonResponse)
}

func (r *Response) update(code int, Value Value) {
	r.Value = Value
	r.StatusCode = code
}

func createErrorResponse() *initapi.AzureErrorResponse {
	return createErrorResponseWithCode(initapi.InternalErrorCode)
}

func createErrorResponseWithCode(code int) *initapi.AzureErrorResponse {
	return &initapi.AzureErrorResponse{StatusCode: code}
}

func (r ListResponse) toString() string {
	jsonResponse, _ := json.Marshal(r)
	return string(jsonResponse)
}
