# \DefaultApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetOauthToken**](DefaultApi.md#GetOauthToken) | **Post** /oauth/token | 
[**HealthNoop**](DefaultApi.md#HealthNoop) | **Get** /health | 
[**Ping**](DefaultApi.md#Ping) | **Get** / | 
[**QueryImagesByPackage**](DefaultApi.md#QueryImagesByPackage) | **Get** /query/images/by_package | List of images containing given package
[**VersionNoop**](DefaultApi.md#VersionNoop) | **Get** /version | 



## GetOauthToken

> TokenResponse GetOauthToken(ctx, )


Request a jwt token for subsequent operations, this request is authenticated with normal HTTP auth

### Required Parameters

This endpoint does not need any parameter.

### Return type

[**TokenResponse**](TokenResponse.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HealthNoop

> HealthNoop(ctx, )


Health check, returns 200 and no body if service is running

### Required Parameters

This endpoint does not need any parameter.

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## Ping

> string Ping(ctx, )


Simple status check

### Required Parameters

This endpoint does not need any parameter.

### Return type

**string**

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## QueryImagesByPackage

> PaginatedImageList QueryImagesByPackage(ctx, name, optional)
List of images containing given package

Filterable query interface to search for images containing specified package

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string**| Name of package to search for (e.g. sed) | 
 **optional** | ***QueryImagesByPackageOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a QueryImagesByPackageOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **packageType** | **optional.String**| Type of package to filter on (e.g. dpkg) | 
 **version** | **optional.String**| Version of named package to filter on (e.g. 4.4-1) | 
 **page** | **optional.String**| The page of results to fetch. Pages start at 1 | 
 **limit** | **optional.Int32**| Limit the number of records for the requested page. If omitted or set to 0, return all results in a single page | 
 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**PaginatedImageList**](PaginatedImageList.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## VersionNoop

> ServiceVersion VersionNoop(ctx, )


Returns the version object for the service, including db schema version info

### Required Parameters

This endpoint does not need any parameter.

### Return type

[**ServiceVersion**](ServiceVersion.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

