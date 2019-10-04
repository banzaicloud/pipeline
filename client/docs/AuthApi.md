# \AuthApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateToken**](AuthApi.md#CreateToken) | **Post** /api/v1/tokens | Create token
[**DeleteToken**](AuthApi.md#DeleteToken) | **Delete** /api/v1/tokens/{tokenId} | Delete an API token
[**ListTokens**](AuthApi.md#ListTokens) | **Get** /api/v1/tokens | List all API tokens



## CreateToken

> TokenCreateResponse CreateToken(ctx, tokenCreateRequest)
Create token

Create token

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**tokenCreateRequest** | [**TokenCreateRequest**](TokenCreateRequest.md)|  | 

### Return type

[**TokenCreateResponse**](TokenCreateResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteToken

> DeleteToken(ctx, tokenId)
Delete an API token

Delete an API token

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**tokenId** | **string**| Token identification | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListTokens

> []TokenListResponseItem ListTokens(ctx, )
List all API tokens

List all API tokens

### Required Parameters

This endpoint does not need any parameter.

### Return type

[**[]TokenListResponseItem**](TokenListResponseItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

