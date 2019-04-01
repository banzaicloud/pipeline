# \ImagesApi

All URIs are relative to *http://localhost/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetImages**](ImagesApi.md#GetImages) | **Get** /providers/{provider}/services/{service}/regions/{region}/images | Provides a list of available images on a given provider in a specific region for a service.


# **GetImages**
> ImagesResponse GetImages(ctx, provider, service, region, optional)
Provides a list of available images on a given provider in a specific region for a service.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **provider** | **string**|  | 
  **service** | **string**|  | 
  **region** | **string**|  | 
 **optional** | ***GetImagesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetImagesOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **gpu** | **optional.String**|  | 
 **version** | **optional.String**|  | 

### Return type

[**ImagesResponse**](ImagesResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

