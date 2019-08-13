# \CommonApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiVersionGet**](CommonApi.md#ApiVersionGet) | **Get** /api/version | 
[**ListEndpoints**](CommonApi.md#ListEndpoints) | **Get** /api | List Pipeline API endpoints


# **ApiVersionGet**
> VersionResponse ApiVersionGet(ctx, )


Return Pipeline build and deployment info

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**VersionResponse**](VersionResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListEndpoints**
> []string ListEndpoints(ctx, )
List Pipeline API endpoints

Listing Pipeline API endpoint

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

