# \HpaApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**DeploymentAutoscalingDelete**](HpaApi.md#DeploymentAutoscalingDelete) | **Delete** /api/v1/orgs/{orgId}/clusters/{id}/hpa | Delete Deployment Scaling
[**DeploymentAutoscalingGet**](HpaApi.md#DeploymentAutoscalingGet) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/hpa | Get Deployment Scaling Info
[**DeploymentAutoscalingPut**](HpaApi.md#DeploymentAutoscalingPut) | **Put** /api/v1/orgs/{orgId}/clusters/{id}/hpa | Create / Update Deployment Scaling


# **DeploymentAutoscalingDelete**
> DeploymentAutoscalingDelete(ctx, orgId, id, scaleTarget)
Delete Deployment Scaling

Delete scaling info for a Helm deployment

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **scaleTarget** | **string**| Deployment or StatefulSet name | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeploymentAutoscalingGet**
> DeploymentScalingResponse DeploymentAutoscalingGet(ctx, orgId, id, scaleTarget)
Get Deployment Scaling Info

Get scaling info for a Helm deployment

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **scaleTarget** | **string**| Deployment or StatefulSet name | 

### Return type

[**DeploymentScalingResponse**](DeploymentScalingResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeploymentAutoscalingPut**
> DeploymentAutoscalingPut(ctx, orgId, id, deploymentScalingRequest)
Create / Update Deployment Scaling

Create / update scaling info for a Helm deployment

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **deploymentScalingRequest** | [**DeploymentScalingRequest**](DeploymentScalingRequest.md)|  | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

