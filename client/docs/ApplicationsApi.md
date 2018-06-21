# \ApplicationsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateApplication**](ApplicationsApi.md#CreateApplication) | **Post** /api/v1/orgs/{orgId}/applications | Create new application based on catalog
[**GetApplication**](ApplicationsApi.md#GetApplication) | **Get** /api/v1/orgs/{orgId}/applications/{appId} | Get application details
[**ListApplications**](ApplicationsApi.md#ListApplications) | **Get** /api/v1/orgs/{orgId}/applications | List application catalogs


# **CreateApplication**
> map[string]interface{} CreateApplication(ctx, orgId, createApplicationRequest)
Create new application based on catalog

Create new application based on Catalog definition s

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **createApplicationRequest** | [**CreateApplicationRequest**](CreateApplicationRequest.md)|  | 

### Return type

[**map[string]interface{}**](map[string]interface{}.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetApplication**
> ApplicationDetailsResponse GetApplication(ctx, orgId, appId)
Get application details

Get application details with deployments

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **appId** | **int32**| Application identification | 

### Return type

[**ApplicationDetailsResponse**](ApplicationDetailsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListApplications**
> ApplicationListResponse ListApplications(ctx, orgId)
List application catalogs

List all available application for lunch

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 

### Return type

[**ApplicationListResponse**](ApplicationListResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

