# \AuthApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**DeleteToken**](AuthApi.md#DeleteToken) | **Delete** /auth/tokens/{tokenId} | Delete an API token
[**GenerateToken**](AuthApi.md#GenerateToken) | **Post** /auth/tokens | Generate token
[**GetTokens**](AuthApi.md#GetTokens) | **Get** /auth/tokens | List all API tokens


# **DeleteToken**
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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GenerateToken**
> TokenCreateResponse GenerateToken(ctx, tokenCreateRequest)
Generate token

Generate token

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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetTokens**
> []TokenListResponseItem GetTokens(ctx, )
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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

