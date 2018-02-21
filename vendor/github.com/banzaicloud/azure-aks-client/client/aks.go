package client

import (
	"encoding/json"
	"github.com/Azure/go-autorest/autorest"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"io/ioutil"
	"net/http"
	"time"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiTypesAzure "github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/azure-aks-client/utils"
	"github.com/pkg/errors"
	"fmt"
	"github.com/sirupsen/logrus"
)

const BaseUrl = "https://management.azure.com"

var logger logrus.FieldLogger

type AKSClient struct {
	azureSdk *cluster.Sdk
	clientId string
	secret   string
}

func SetLogger(l logrus.FieldLogger) {
	logger = l
}

func GetAKSClient(credentials *cluster.AKSCredential) (*AKSClient, error) {

	azureSdk, err := cluster.Authenticate(credentials)
	if err != nil {
		return nil, err
	}
	aksClient := &AKSClient{
		clientId: azureSdk.ServicePrincipal.ClientID,
		secret:   azureSdk.ServicePrincipal.ClientSecret,
		azureSdk: azureSdk,
	}
	if aksClient.clientId == "" {
		return nil, utils.NewErr("clientID is missing")
	}
	if aksClient.secret == "" {
		return nil, utils.NewErr("secret is missing")
	}
	return aksClient, nil
}

/**
GetCluster gets the details of the managed cluster with a specified resource group and name.
GET https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/
	{resourceName}?api-version=2017-08-31
 */
func (a *AKSClient) GetCluster(name string, resourceGroup string) (*banzaiTypesAzure.ResponseWithValue, error) {

	resp, errAz := a.callAzureGetCluster(name, resourceGroup)
	if errAz != nil {
		return nil, errAz
	}

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error during get cluster in %s resource group", resourceGroup))
	}

	if resp.StatusCode != http.StatusOK {
		// not ok, probably 404
		err := utils.CreateErrorFromValue(resp.StatusCode, value)
		return nil, err
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
func (a *AKSClient) ListClusters(resourceGroup string) (*banzaiTypesAzure.ListResponse, error) {

	pathParam := map[string]interface{}{
		"subscription-id": a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		msg := fmt.Sprint("error during listing clusters in ", resourceGroup, " resource group: ", err)
		return nil, utils.NewErr(msg)
	}
	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		msg := fmt.Sprint("error during listing clusters in ", resourceGroup, " resource group:", err)
		return nil, utils.NewErr(msg)
	}

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprint("error during listing clusters in ", resourceGroup, " resource group:", err)
		return nil, utils.NewErr(msg)
	}

	if resp.StatusCode != http.StatusOK {
		// not ok, probably 404
		return nil, utils.CreateErrorFromValue(resp.StatusCode, value)
	}
	azureListResponse := banzaiTypesAzure.Values{}
	json.Unmarshal([]byte(value), &azureListResponse)
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
func (a *AKSClient) CreateUpdateCluster(request cluster.CreateClusterRequest) (*banzaiTypesAzure.ResponseWithValue, error) {

	if err := request.Validate(); err != nil {
		return nil, err
	}

	managedCluster := cluster.GetManagedCluster(request, a.clientId, a.secret)

	pathParam := map[string]interface{}{
		"subscription-id": a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   request.ResourceGroup,
		"resourceName":    request.Name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

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
		msg := fmt.Sprint("error during JSON marshal: ", err)
		return nil, utils.NewErr(msg)
	}

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		msg := fmt.Sprint("error during cluster creation: ", err)
		return nil, utils.NewErr(msg)
	}

	defer resp.Body.Close()
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprint("error during cluster creation:", err)
		return nil, utils.NewErr(msg)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// something went wrong, create failed
		errResp := utils.CreateErrorFromValue(resp.StatusCode, value)
		return nil, errResp
	}

	v := banzaiTypesAzure.Value{}
	json.Unmarshal([]byte(value), &v)
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
func (a *AKSClient) DeleteCluster(name string, resourceGroup string) (error) {

	pathParam := map[string]interface{}{
		"subscription-id": a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup,
		"resourceName":    name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsDelete(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam),
	)

	if err != nil {
		return err
	}
	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusAccepted {
		err := utils.CreateErrorFromValue(resp.StatusCode, value)
		return err
	}

	return nil
}

/*
PollingCluster polling AKS on Azure
GET https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?
	api-version=2017-08-31
 */
func (a *AKSClient) PollingCluster(name string, resourceGroup string) (*banzaiTypesAzure.ResponseWithValue, error) {

	const stageSuccess = "Succeeded"
	const stageFailed = "Failed"
	const waitInSeconds = 10

	if logger != nil {
		logger.Infof("Start polling cluster: %s [%s]", name, resourceGroup)
	}

	pathParam := map[string]interface{}{
		"subscription-id": a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup,
		"resourceName":    name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam),
	)

	if err != nil {
		return nil, err
	}

	result := banzaiTypesAzure.ResponseWithValue{}
	for isReady := false; !isReady; {

		resp, err := autorest.SendWithSender(groupClient.Client, req)
		if err != nil {
			return nil, err
		}

		statusCode := resp.StatusCode
		if logger != nil {
			logger.Infof("Cluster polling status code: %d", statusCode)
		}

		value, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		switch statusCode {
		case http.StatusOK:
			response := banzaiTypesAzure.Value{}
			json.Unmarshal([]byte(value), &response)

			stage := response.Properties.ProvisioningState
			if logger != nil {
				logger.Infof("Cluster stage is %s", stage)
			}

			switch stage {
			case stageSuccess:
				isReady = true
				result.Update(http.StatusCreated, response)
			case stageFailed:
				return nil, banzaiConstants.ErrorAzureCLusterStageFailed
			default:
				if logger != nil {
					logger.Info("Waiting for cluster ready...")
				}
				time.Sleep(waitInSeconds * time.Second)
			}

		default:
			err := utils.CreateErrorFromValue(resp.StatusCode, value)
			return nil, err
		}
	}

	return &result, nil
}

/**
Get kubernetes cluster config
GET https://management.azure.com/subscriptions/
	{subscriptionId}/resourceGroups/
	{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/
	{resourceName}?api-version=2017-08-31
 */
func (a *AKSClient) GetClusterConfig(name, resourceGroup, roleName string) (*banzaiTypesAzure.Config, error) {

	pathParam := map[string]interface{}{
		"subscriptionId":    a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroupName": resourceGroup,
		"resourceName":      name,
		"roleName":          roleName,
	}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/accessProfiles/{roleName}", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		return nil, err
	}
	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		return nil, err
	}

	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// not ok, probably 404
		err := utils.CreateErrorFromValue(resp.StatusCode, value)
		return nil, err
	} else {
		// everything is ok
		res := banzaiTypesAzure.Config{}
		json.Unmarshal([]byte(value), &res)
		return &res, nil
	}

}

func (a *AKSClient) callAzureGetCluster(name, resourceGroup string) (*http.Response, error) {

	pathParam := map[string]interface{}{
		"subscription-id": a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup,
		"resourceName":    name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		return nil, err
	}

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
