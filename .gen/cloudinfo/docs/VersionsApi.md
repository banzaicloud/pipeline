# \VersionsApi

All URIs are relative to *http://localhost/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetVersions**](VersionsApi.md#GetVersions) | **Get** /providers/{provider}/services/{service}/regions/{region}/versions | Provides a list of available versions on a given provider in a specific region for a service.


# **GetVersions**
> VersionsResponse GetVersions(ctx, provider, service, region)
Provides a list of available versions on a given provider in a specific region for a service.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **provider** | **string**|  | 
  **service** | **string**|  | 
  **region** | **string**|  | 

### Return type

[**VersionsResponse**](VersionsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

