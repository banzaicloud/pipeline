# \InfoApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetAmazonConfig**](InfoApi.md#GetAmazonConfig) | **Get** /api/v1/orgs/{orgId}/cloudinfo/amazon | Get all amazon config
[**GetAzureConfig**](InfoApi.md#GetAzureConfig) | **Get** /api/v1/orgs/{orgId}/cloudinfo/azure | Get all azure config
[**GetGoogleConfig**](InfoApi.md#GetGoogleConfig) | **Get** /api/v1/orgs/{orgId}/cloudinfo/google | Get all google config
[**GetSupportedClouds**](InfoApi.md#GetSupportedClouds) | **Get** /api/v1/orgs/{orgId}/cloudinfo | Get supported cloud types


# **GetAmazonConfig**
> AmazonConfigResponse GetAmazonConfig(ctx, orgId, secretId, fields, tags, location)
Get all amazon config

Get all amazon config

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identifier | 
  **fields** | **string**| Filter fields | 
  **tags** | **string**| Tags filter | 
  **location** | **string**| Location filter | 

### Return type

[**AmazonConfigResponse**](AmazonConfigResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetAzureConfig**
> AzureConfigResponse GetAzureConfig(ctx, orgId, secretId, fields, location)
Get all azure config

Get all azure config

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identifier | 
  **fields** | **string**| Filter fields | 
  **location** | **string**| Location filter | 

### Return type

[**AzureConfigResponse**](AzureConfigResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetGoogleConfig**
> GoogleConfigResponse GetGoogleConfig(ctx, orgId, secretId, fields, location)
Get all google config

Get all google config

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identifier | 
  **fields** | **string**| Filter fields | 
  **location** | **string**| Location filter | 

### Return type

[**GoogleConfigResponse**](GoogleConfigResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetSupportedClouds**
> SupportedCloudsResponse GetSupportedClouds(ctx, orgId)
Get supported cloud types

Get supported cloud types

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 

### Return type

[**SupportedCloudsResponse**](SupportedCloudsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

