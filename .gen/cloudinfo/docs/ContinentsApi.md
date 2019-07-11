# \ContinentsApi

All URIs are relative to *http://localhost/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetContinents**](ContinentsApi.md#GetContinents) | **Get** /continents | 
[**GetContinentsData**](ContinentsApi.md#GetContinentsData) | **Get** /providers/{provider}/services/{service}/continents | 


# **GetContinents**
> []string GetContinents(ctx, )


Returns the supported continents

### Required Parameters
This endpoint does not need any parameter.

### Return type

**[]string**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetContinentsData**
> []Continent GetContinentsData(ctx, provider, service)


Provides the list of available continents and regions of a cloud provider

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **provider** | **string**|  | 
  **service** | **string**|  | 

### Return type

[**[]Continent**](Continent.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

