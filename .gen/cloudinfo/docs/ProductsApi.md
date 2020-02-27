# \ProductsApi

All URIs are relative to *http://localhost/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetProducts**](ProductsApi.md#GetProducts) | **Get** /providers/{provider}/services/{service}/regions/{region}/products | Provides a list of available machine types on a given provider in a specific region.



## GetProducts

> ProductDetailsResponse GetProducts(ctx, provider, service, region)

Provides a list of available machine types on a given provider in a specific region.

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**provider** | **string**|  | 
**service** | **string**|  | 
**region** | **string**|  | 

### Return type

[**ProductDetailsResponse**](ProductDetailsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

