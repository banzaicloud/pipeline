# \DeploymentsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateDeployment**](DeploymentsApi.md#CreateDeployment) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/deployments | Create a Helm deployment
[**GetTillerStatus**](DeploymentsApi.md#GetTillerStatus) | **Head** /api/v1/orgs/{orgId}/clusters/{id}/deployments | Get tiller status
[**ListDeployments**](DeploymentsApi.md#ListDeployments) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/deployments | List deployments


# **CreateDeployment**
> CreateUpdateDeploymentResponse CreateDeployment(ctx, orgId, id, createUpdateDeploymentRequest)
Create a Helm deployment

Creating a Helm deployment

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **createUpdateDeploymentRequest** | [**CreateUpdateDeploymentRequest**](CreateUpdateDeploymentRequest.md)|  | 

### Return type

[**CreateUpdateDeploymentResponse**](CreateUpdateDeploymentResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTillerStatus**
> GetTillerStatus(ctx, orgId, id)
Get tiller status

Checking if tiller ready to accept deployments

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListDeployments**
> ListDeploymentsResponse ListDeployments(ctx, orgId, id)
List deployments

Listing Helm deployments

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**ListDeploymentsResponse**](ListDeploymentsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

