package client

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/go-autorest/autorest"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/azure-aks-client/utils"
	banzaiTypesAzure "github.com/banzaicloud/banzai-types/components/azure"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

const BaseUrl = "https://management.azure.com"

type AKSClient struct {
	azureSdk *cluster.Sdk
	logger   *logrus.Logger
	clientId string
	secret   string
}

// GetAKSClient creates an *AKSClient instance with the passed credentials and default logger
func GetAKSClient(credentials *cluster.AKSCredential) (*AKSClient, error) {

	azureSdk, err := cluster.Authenticate(credentials)
	if err != nil {
		return nil, err
	}
	aksClient := &AKSClient{
		clientId: azureSdk.ServicePrincipal.ClientID,
		secret:   azureSdk.ServicePrincipal.ClientSecret,
		azureSdk: azureSdk,
		logger:   getDefaultLogger(),
	}
	if aksClient.clientId == "" {
		return nil, utils.NewErr("clientID is missing")
	}
	if aksClient.secret == "" {
		return nil, utils.NewErr("secret is missing")
	}
	return aksClient, nil
}

// With sets logger
func (a *AKSClient) With(i interface{}) {
	if a != nil {
		switch i.(type) {
		case logrus.Logger:
			logger := i.(logrus.Logger)
			a.logger = &logger
		case *logrus.Logger:
			a.logger = i.(*logrus.Logger)
		}
	}
}

// getDefaultLogger return the default logger
func getDefaultLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel
	logger.Formatter = new(logrus.JSONFormatter)
	return logger
}

// GetCluster gets the details of the managed cluster with a specified resource group and name.
//
// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?api-version=2017-08-31
func (a *AKSClient) GetCluster(name string, resourceGroup string) (*banzaiTypesAzure.ResponseWithValue, error) {

	a.logInfof("Start getting aks cluster: %s [%s]", name, resourceGroup)

	resp, errAz := a.callAzureGetCluster(name, resourceGroup)
	if errAz != nil {
		return nil, errAz
	}

	a.logDebugf("Read body: %v", resp.Body)
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error during get cluster in %s resource group", resourceGroup))
	}

	a.logInfof("Status code: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		// not ok, probably 404
		err := utils.CreateErrorFromValue(resp.StatusCode, value)
		return nil, err
	} else {
		// everything is ok
		a.logDebug("Create response model")
		v := banzaiTypesAzure.Value{}
		json.Unmarshal([]byte(value), &v)
		response := banzaiTypesAzure.ResponseWithValue{}
		response.Update(resp.StatusCode, v)
		return &response, nil
	}

}

// ListClusters is listing AKS clusters in the specified subscription and resource group
//
// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters?api-version=2017-08-31
func (a *AKSClient) ListClusters(resourceGroup string) (*banzaiTypesAzure.ListResponse, error) {

	a.logInfof("Start getting cluster list from %s resource group", resourceGroup)

	pathParam := map[string]interface{}{
		"subscription-id": a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

	a.logDebug("Create request")
	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		msg := fmt.Sprintf("error during listing clusters in %s resource group: %v", resourceGroup, err)
		return nil, utils.NewErr(msg)
	}

	a.logDebug("Send http request to azure")

	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		msg := fmt.Sprintf("error during listing clusters in %s resource group: %v", resourceGroup, err)
		return nil, utils.NewErr(msg)
	}

	a.logDebugf("Read response body %v", resp.Body)
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("error during listing clusters in %s resource group: %v", resourceGroup, err)
		return nil, utils.NewErr(msg)
	}

	a.logInfof("Status code %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		// not ok, probably 404
		return nil, utils.CreateErrorFromValue(resp.StatusCode, value)
	}

	a.logInfo("Create response model")

	azureListResponse := banzaiTypesAzure.Values{}
	json.Unmarshal([]byte(value), &azureListResponse)
	response := banzaiTypesAzure.ListResponse{StatusCode: resp.StatusCode, Value: azureListResponse}
	return &response, nil
}

// CreateUpdateCluster creates or updates a managed cluster
//
// PUT https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/ {resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?api-version=2017-08-31
func (a *AKSClient) CreateUpdateCluster(request cluster.CreateClusterRequest) (*banzaiTypesAzure.ResponseWithValue, error) {

	a.logInfo("Start create/update cluster")
	a.logDebugf("CreateRequest: %v", request)
	a.logInfo("Validate cluster create/update request")

	if err := request.Validate(); err != nil {
		return nil, err
	}
	a.logInfo("Validate passed")

	managedCluster := cluster.GetManagedCluster(request, a.clientId, a.secret)
	a.logDebugf("Created managed cluster model - %#v", &managedCluster)

	pathParam := map[string]interface{}{
		"subscription-id": a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   request.ResourceGroup,
		"resourceName":    request.Name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

	a.logDebug("Create http request")
	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsPut(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam),
		autorest.WithJSON(managedCluster),
		autorest.AsContentType("application/json"),
	)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error during create/update request %v", err))
	}

	_, err = json.Marshal(managedCluster)
	if err != nil {
		msg := fmt.Sprint("error during JSON marshal: ", err)
		return nil, utils.NewErr(msg)
	}

	a.logDebug("Send request to azure")
	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		msg := fmt.Sprint("error during cluster creation: ", err)
		return nil, utils.NewErr(msg)
	}

	defer resp.Body.Close()
	a.logDebugf("Read response body: %v", resp.Body)
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprint("error during cluster creation:", err)
		return nil, utils.NewErr(msg)
	}

	a.logInfo("Status code: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// something went wrong, create failed
		errResp := utils.CreateErrorFromValue(resp.StatusCode, value)
		return nil, errResp
	}

	a.logInfo("Create response model")
	v := banzaiTypesAzure.Value{}
	json.Unmarshal([]byte(value), &v)
	result := banzaiTypesAzure.ResponseWithValue{StatusCode: resp.StatusCode, Value: v}
	return &result, nil
}

// DeleteCluster deletes a managed AKS on Azure
//
// DELETE https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?api-version=2017-08-31
func (a *AKSClient) DeleteCluster(name string, resourceGroup string) error {

	a.logInfof("Start deleting cluster %s in %s resource group", name, resourceGroup)

	pathParam := map[string]interface{}{
		"subscription-id": a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup,
		"resourceName":    name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

	a.logDebug("Create http request")
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

	a.logDebug("Send request to azure")
	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	a.logDebugf("Read response body: %v", resp.Body)
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	a.logInfof("Status code: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusAccepted {
		err := utils.CreateErrorFromValue(resp.StatusCode, value)
		return err
	}

	return nil
}

//PollingCluster polling AKS on Azure
//
//GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?api-version=2017-08-31
func (a *AKSClient) PollingCluster(name string, resourceGroup string) (*banzaiTypesAzure.ResponseWithValue, error) {

	const stageSuccess = "Succeeded"
	const stageFailed = "Failed"
	const waitInSeconds = 10

	a.logInfof("Start polling cluster: %s [%s]", name, resourceGroup)

	pathParam := map[string]interface{}{
		"subscription-id": a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroup":   resourceGroup,
		"resourceName":    name}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

	a.logDebug("Create http request")
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

	a.logDebug("Start loop")

	result := banzaiTypesAzure.ResponseWithValue{}
	for isReady := false; !isReady; {

		a.logDebug("Send request to azure")
		resp, err := autorest.SendWithSender(groupClient.Client, req)
		if err != nil {
			return nil, err
		}

		statusCode := resp.StatusCode
		a.logInfof("Cluster polling status code: %d", statusCode)

		a.logDebug("Read response body")
		value, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		switch statusCode {
		case http.StatusOK:
			response := banzaiTypesAzure.Value{}
			json.Unmarshal([]byte(value), &response)

			stage := response.Properties.ProvisioningState
			a.logInfof("Cluster stage is %s", stage)

			switch stage {
			case stageSuccess:
				isReady = true
				result.Update(http.StatusCreated, response)
			case stageFailed:
				return nil, banzaiConstants.ErrorAzureCLusterStageFailed
			default:
				a.logInfo("Waiting for cluster ready...")
				time.Sleep(waitInSeconds * time.Second)
			}

		default:
			err := utils.CreateErrorFromValue(resp.StatusCode, value)
			return nil, err
		}
	}

	return &result, nil
}

//Get kubernetes cluster config
//
//GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}?api-version=2017-08-31
func (a *AKSClient) GetClusterConfig(name, resourceGroup, roleName string) (*banzaiTypesAzure.Config, error) {

	a.logInfof("Start getting %s cluster's config in %s, role name: %s", name, resourceGroup, roleName)

	pathParam := map[string]interface{}{
		"subscriptionId":    a.azureSdk.ServicePrincipal.SubscriptionID,
		"resourceGroupName": resourceGroup,
		"resourceName":      name,
		"roleName":          roleName,
	}
	queryParam := map[string]interface{}{"api-version": "2017-08-31"}

	groupClient := *a.azureSdk.ResourceGroup

	a.logDebug("Create http request")
	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/accessProfiles/{roleName}", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		return nil, err
	}

	a.logDebug("Send request to azure")
	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		return nil, err
	}

	a.logDebugf("Read response body: %v", resp.Body)
	value, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	a.logInfof("Status code: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		// not ok, probably 404
		err := utils.CreateErrorFromValue(resp.StatusCode, value)
		return nil, err
	} else {
		// everything is ok
		a.logInfo("Create response model")
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

	a.logDebug("Create http request")
	req, err := autorest.Prepare(&http.Request{},
		groupClient.WithAuthorization(),
		autorest.AsGet(),
		autorest.WithBaseURL(BaseUrl),
		autorest.WithPathParameters("/subscriptions/{subscription-id}/resourceGroups/{resourceGroup}/providers/Microsoft.ContainerService/managedClusters/{resourceName}", pathParam),
		autorest.WithQueryParameters(queryParam))

	if err != nil {
		return nil, err
	}

	a.logDebug("Send request to azure")
	resp, err := autorest.SendWithSender(groupClient.Client, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *AKSClient) logDebug(args ...interface{}) {
	if a.logger != nil {
		a.logger.Debug(args...)
	}
}
func (a *AKSClient) logInfo(args ...interface{}) {
	if a.logger != nil {
		a.logger.Info(args...)
	}
}
func (a *AKSClient) logWarn(args ...interface{}) {
	if a.logger != nil {
		a.logger.Warn(args...)
	}
}
func (a *AKSClient) logError(args ...interface{}) {
	if a.logger != nil {
		a.logger.Error(args...)
	}
}

func (a *AKSClient) logFatal(args ...interface{}) {
	if a.logger != nil {
		a.logger.Fatal(args...)
	}
}

func (a *AKSClient) logPanic(args ...interface{}) {
	if a.logger != nil {
		a.logger.Panic(args...)
	}
}

func (a *AKSClient) logDebugf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Debugf(format, args...)
	}
}

func (a *AKSClient) logInfof(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Infof(format, args...)
	}
}

func (a *AKSClient) logWarnf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Warnf(format, args...)
	}
}

func (a *AKSClient) logErrorf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Errorf(format, args...)
	}
}

func (a *AKSClient) logFatalf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Fatalf(format, args...)
	}
}

func (a *AKSClient) logPanicf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Panicf(format, args...)
	}
}
