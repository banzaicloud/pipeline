# \InfoApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateResourceGroup**](InfoApi.md#CreateResourceGroup) | **Post** /api/v1/orgs/{orgId}/azure/resourcegroups | Create resource groups
[**DeleteResourceGroup**](InfoApi.md#DeleteResourceGroup) | **Delete** /api/v1/orgs/{orgId}/azure/resourcegroups/{name} | Delete resource group
[**GetResourceGroups**](InfoApi.md#GetResourceGroups) | **Get** /api/v1/orgs/{orgId}/azure/resourcegroups | Get all resource groups


# **CreateResourceGroup**
> ResourceGroupCreated CreateResourceGroup(ctx, orgId, createResourceGroup)
Create resource groups

Create resource groups

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **createResourceGroup** | [**CreateResourceGroup**](CreateResourceGroup.md)|  | 

### Return type

[**ResourceGroupCreated**](ResourceGroupCreated.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteResourceGroup**
> DeleteResourceGroup(ctx, orgId, name, secretId)
Delete resource group

Delete resource group

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **name** | **int32**| Resource group name | 
  **secretId** | **string**| Secret identifier | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetResourceGroups**
> []string GetResourceGroups(ctx, orgId, secretId)
Get all resource groups

Get all resource groups

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identifier | 

### Return type

**[]string**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

