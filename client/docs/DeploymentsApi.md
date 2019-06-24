# \DeploymentsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateDeployment**](DeploymentsApi.md#CreateDeployment) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/deployments | Create a Helm deployment
[**DeleteDeployment**](DeploymentsApi.md#DeleteDeployment) | **Delete** /api/v1/orgs/{orgId}/clusters/{id}/deployments/{name} | Delete deployment
[**GetDeployment**](DeploymentsApi.md#GetDeployment) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/deployments/{name} | Get deployment details
[**GetDeploymentImages**](DeploymentsApi.md#GetDeploymentImages) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/deployments/{name}/images | Get deployment K8s images
[**GetDeploymentResource**](DeploymentsApi.md#GetDeploymentResource) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/deployments/{name}/resources | Get deployment K8s resources
[**GetTillerStatus**](DeploymentsApi.md#GetTillerStatus) | **Head** /api/v1/orgs/{orgId}/clusters/{id}/deployments | Get tiller status
[**HelmDeploymentStatus**](DeploymentsApi.md#HelmDeploymentStatus) | **Head** /api/v1/orgs/{orgId}/clusters/{id}/deployments/{name} | Check deployment status
[**ListDeployments**](DeploymentsApi.md#ListDeployments) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/deployments | List deployments
[**UpdateDeployment**](DeploymentsApi.md#UpdateDeployment) | **Put** /api/v1/orgs/{orgId}/clusters/{id}/deployments/{name} | Update deployment


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

# **DeleteDeployment**
> DeleteDeploymentResponse DeleteDeployment(ctx, orgId, id, name)
Delete deployment

Deleting a Helm deployment

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **name** | **string**| Deployment name | 

### Return type

[**DeleteDeploymentResponse**](DeleteDeploymentResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetDeployment**
> GetDeploymentResponse GetDeployment(ctx, orgId, id, name, optional)
Get deployment details

Retrieves the details of a deployment

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **name** | **string**| Deployment name | 
 **optional** | ***GetDeploymentOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetDeploymentOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **tag** | **optional.String**| Deployment tag | 

### Return type

[**GetDeploymentResponse**](GetDeploymentResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetDeploymentImages**
> []ClusterImage GetDeploymentImages(ctx, orgId, id, name)
Get deployment K8s images

Retrieves the K8s images of a deployment

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **name** | **string**| Deployment name | 

### Return type

[**[]ClusterImage**](ClusterImage.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetDeploymentResource**
> []map[string]interface{} GetDeploymentResource(ctx, orgId, id, name, optional)
Get deployment K8s resources

Retrieves the K8s resources of a deployment

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **name** | **string**| Deployment name | 
 **optional** | ***GetDeploymentResourceOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetDeploymentResourceOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **resourceTypes** | **optional.String**| K8s reource type | 

### Return type

[**[]map[string]interface{}**](map[string]interface{}.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
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

# **HelmDeploymentStatus**
> HelmDeploymentStatus(ctx, orgId, id, name)
Check deployment status

Checking the status of a deployment through the Helm client API

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **name** | **string**| Deployment name | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListDeployments**
> []ListDeploymentsResponseItem ListDeployments(ctx, orgId, id, optional)
List deployments

Listing Helm deployments

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
 **optional** | ***ListDeploymentsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ListDeploymentsOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **tag** | **optional.String**| Deployment tag to filter for | 

### Return type

[**[]ListDeploymentsResponseItem**](ListDeploymentsResponseItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateDeployment**
> CreateUpdateDeploymentResponse UpdateDeployment(ctx, orgId, id, name, createUpdateDeploymentRequest)
Update deployment

Updating a Helm deployment

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **name** | **string**| Deployment name | 
  **createUpdateDeploymentRequest** | [**CreateUpdateDeploymentRequest**](CreateUpdateDeploymentRequest.md)|  | 

### Return type

[**CreateUpdateDeploymentResponse**](CreateUpdateDeploymentResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

